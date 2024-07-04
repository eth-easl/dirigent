package control_plane

import (
	"cluster_manager/internal/control_plane/control_plane/autoscalers/default_autoscaler"
	predictive_autoscaler2 "cluster_manager/internal/control_plane/control_plane/autoscalers/predictive_autoscaler"
	"cluster_manager/internal/control_plane/control_plane/core"
	"cluster_manager/internal/control_plane/control_plane/endpoint_placer"
	"cluster_manager/internal/control_plane/control_plane/image_storage"
	"cluster_manager/internal/control_plane/control_plane/persistence"
	"cluster_manager/internal/control_plane/control_plane/service_state"
	"cluster_manager/internal/control_plane/workflow"
	"cluster_manager/pkg/config"
	_map "cluster_manager/pkg/map"
	"cluster_manager/pkg/synchronization"
	"cluster_manager/pkg/tracing"
	"cluster_manager/pkg/utils"
	"cluster_manager/proto"
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/emptypb"
)

type AutoscalingInterface interface {
	Create(functionState *service_state.ServiceState)
	PanicPoke(functionName string, previousValue int32)
	Poke(functionName string, previousValue int32)
	ForwardDataplaneMetrics(dataplaneMetrics *proto.MetricsPredictiveAutoscaler) error
	Stop(functionName string)
}

type ControlPlane struct {
	DataPlaneConnections synchronization.SyncStructure[string, core.DataPlaneInterface]
	NIStorage            synchronization.SyncStructure[string, core.WorkerNodeInterface]
	SIStorage            synchronization.SyncStructure[string, *endpoint_placer.EndpointPlacer]
	WIStorage            synchronization.SyncStructure[string, *workflow.StorageTacker]
	imageStorage         image_storage.ImageStorage

	ColdStartTracing *tracing.TracingService[tracing.ColdStartLogEntry] `json:"-"`
	PersistenceLayer persistence.PersistenceLayer

	dataPlaneCreator  core.DataplaneFactory
	workerNodeCreator core.WorkerNodeFactory

	Config *config.ControlPlaneConfig

	autoscalingManager AutoscalingInterface
}

func NewControlPlane(client persistence.PersistenceLayer, outputFile string, dataplaneCreator core.DataplaneFactory, workerNodeCreator core.WorkerNodeFactory, cfg *config.ControlPlaneConfig) *ControlPlane {

	if !isValidAutoscaler(cfg.Autoscaler) {
		logrus.Fatalf("Invalid autoscaler type %s", cfg.Autoscaler)
	}

	var autoscalingManager AutoscalingInterface

	if cfg.Autoscaler == utils.DEFAULT_AUTOSCALER {
		autoscalingManager = default_autoscaler.NewMultiscaler(cfg.AutoscalingPeriod)
	} else {
		autoscalingManager = predictive_autoscaler2.NewMultiScaler(cfg)
	}

	return &ControlPlane{
		DataPlaneConnections: synchronization.NewControlPlaneSyncStructure[string, core.DataPlaneInterface](),
		NIStorage:            synchronization.NewControlPlaneSyncStructure[string, core.WorkerNodeInterface](),
		SIStorage:            synchronization.NewControlPlaneSyncStructure[string, *endpoint_placer.EndpointPlacer](),
		WIStorage:            synchronization.NewControlPlaneSyncStructure[string, *workflow.StorageTacker](),
		imageStorage:         image_storage.ParseImageStorage(cfg.ImageStorage),

		ColdStartTracing: tracing.NewColdStartTracingService(outputFile),
		PersistenceLayer: client,

		dataPlaneCreator:  dataplaneCreator,
		workerNodeCreator: workerNodeCreator,

		Config: cfg,

		autoscalingManager: autoscalingManager,
	}
}

// Dataplanes functions

func (c *ControlPlane) registerDataplane(ctx context.Context, in *proto.DataplaneInfo) (*proto.ActionStatus, error, bool) {
	logrus.Infof("Received a data plane registration with ip : %s", in.IP)

	dataplaneInfo := proto.DataplaneInformation{
		Address:   in.IP,
		ApiPort:   strconv.Itoa(int(in.APIPort)),
		ProxyPort: strconv.Itoa(int(in.ProxyPort)),
	}

	key := in.IP
	dataplaneConnection := c.dataPlaneCreator(dataplaneInfo.Address, dataplaneInfo.ApiPort, dataplaneInfo.ProxyPort)

	c.WIStorage.Lock()
	defer c.WIStorage.Unlock()

	c.SIStorage.Lock()
	defer c.SIStorage.Unlock()

	c.DataPlaneConnections.Lock()
	defer c.DataPlaneConnections.Unlock()

	if _, present := c.DataPlaneConnections.Get(key); present {
		logrus.Tracef("Dataplane with ip %s already registered - update last heartbeat timestamp", dataplaneInfo.Address)

		c.DataPlaneConnections.GetMap()[key].UpdateHeartBeat()

		return &proto.ActionStatus{Success: true}, nil, true
	}

	if c.DataPlaneConnections.Len() >= 1 && c.Config.Autoscaler != utils.DEFAULT_AUTOSCALER {
		errorText := fmt.Sprintf("Current autoscaling policy %s supports only one data plane", c.Config.Autoscaler)
		logrus.Error(errorText)
		return &proto.ActionStatus{Success: false}, errors.New(errorText), false
	}

	err := c.PersistenceLayer.StoreDataPlaneInformation(ctx, &dataplaneInfo)
	if err != nil {
		logrus.Errorf("Failed to store information to persistence layer (error : %s)", err.Error())
		c.DataPlaneConnections.Remove(key)
		return &proto.ActionStatus{Success: false}, err, false
	}

	err = dataplaneConnection.InitializeDataPlaneConnection(dataplaneInfo.Address, dataplaneInfo.ApiPort)
	if err != nil {
		logrus.Errorf("Failed to initialize dataplane connection (error : %s)", err.Error())
		c.DataPlaneConnections.Remove(key)
		return &proto.ActionStatus{Success: false}, err, false
	}

	c.DataPlaneConnections.Set(key, dataplaneConnection)

	for _, service := range c.SIStorage.GetMap() {
		if service.ServiceState.TaskInfo == nil {
			if _, err = c.DataPlaneConnections.GetNoCheck(key).AddDeployment(ctx, service.ServiceState.FunctionInfo[0]); err != nil {
				logrus.Errorf("Failed to add deployement : %s", err.Error())
				c.DataPlaneConnections.Remove(key)
				return &proto.ActionStatus{Success: false}, err, false
			}
		}

		service.SingleThreadUpdateEndpoints(service.PrepareCurrentEndpointInfoList())
	}

	for _, wfObject := range c.WIStorage.GetMap() {
		if wfObject.IsWorkflow() {
			if _, err = c.DataPlaneConnections.GetNoCheck(key).AddWorkflowDeployment(ctx, wfObject.GetWorkflowInfo()); err != nil {
				logrus.Errorf("Failed to add workflow deployment : %s", err.Error())
				c.DataPlaneConnections.Remove(key)
				return &proto.ActionStatus{Success: false}, err, false
			}
		}
	}

	return &proto.ActionStatus{Success: true}, nil, false
}

func (c *ControlPlane) deregisterDataplane(ctx context.Context, in *proto.DataplaneInfo) (*proto.ActionStatus, error) {
	logrus.Info("Received a data plane deregistration")

	dataplaneInfo := proto.DataplaneInformation{
		Address:   in.IP,
		ApiPort:   strconv.Itoa(int(in.APIPort)),
		ProxyPort: strconv.Itoa(int(in.ProxyPort)),
	}

	c.SIStorage.Lock()
	defer c.SIStorage.Unlock()

	c.DataPlaneConnections.Lock()
	defer c.DataPlaneConnections.Unlock()

	if _, present := c.DataPlaneConnections.Get(dataplaneInfo.Address); present {
		err := c.PersistenceLayer.DeleteDataPlaneInformation(ctx, &dataplaneInfo)
		if err != nil {
			logrus.Errorf("Failed to store information to persistence layer (error : %s)", err.Error())
			return &proto.ActionStatus{Success: false}, err
		}

		for _, value := range c.SIStorage.GetMap() {
			value.ServiceState.RemoveDataplane(in.IP)
		}

		c.DataPlaneConnections.Remove(dataplaneInfo.Address)
	}

	return &proto.ActionStatus{Success: true}, nil
}

// Node functions

func (c *ControlPlane) getHAProxyConfig() *proto.HAProxyConfig {
	var dataPlanes []string
	registrationServers := append(
		[]string{c.Config.RegistrationServer},
		c.Config.RegistrationServerReplicas...,
	)

	c.DataPlaneConnections.Lock()
	for _, conn := range c.DataPlaneConnections.GetMap() {
		dataPlanes = append(dataPlanes, fmt.Sprintf("%s:%s", conn.GetIP(), conn.GetProxyPort()))
	}
	c.DataPlaneConnections.Unlock()

	return &proto.HAProxyConfig{
		Dataplanes:          dataPlanes,
		RegistrationServers: registrationServers,
	}
}

func (c *ControlPlane) registerNode(ctx context.Context, in *proto.NodeInfo) (*proto.ActionStatus, error) {
	logrus.Infof("Received a node registration with name : %s", in.NodeID)

	wn := c.workerNodeCreator(core.WorkerNodeConfiguration{
		Name:   in.NodeID,
		IP:     in.IP,
		Port:   strconv.Itoa(int(in.Port)),
		Cpu:    in.CpuCores,
		Memory: in.MemorySize,
	})

	c.NIStorage.Lock()
	defer c.NIStorage.Unlock()

	if _, present := c.NIStorage.Get(in.NodeID); present {
		return &proto.ActionStatus{
			Success: false,
			Message: "Node registration failed. Node with the same name already exists.",
		}, nil
	}

	err := c.PersistenceLayer.StoreWorkerNodeInformation(ctx, &proto.WorkerNodeInformation{
		Name:     in.NodeID,
		Ip:       in.IP,
		Port:     strconv.Itoa(int(in.Port)),
		CpuCores: in.CpuCores,
		Memory:   in.MemorySize,
	})
	if err != nil {
		logrus.Errorf("Failed to store information to persistence layer (error : %s)", err.Error())
		c.NIStorage.Remove(in.NodeID)
		return &proto.ActionStatus{Success: false}, err
	}
	c.imageStorage.Lock()
	for _, image := range in.Images {
		c.imageStorage.RegisterNoFetch(image.URL, image.Size, wn)
	}
	c.imageStorage.Unlock()

	c.NIStorage.Set(in.NodeID, wn)

	wn.ConnectToWorker()
	wn.SetSchedulability(true)

	logrus.Infof("Node %s has been successfully registered with the control plane and cluster has %d nodes", in.NodeID, c.NIStorage.Len())

	return &proto.ActionStatus{Success: true}, nil
}

func (c *ControlPlane) deregisterNode(_ context.Context, in *proto.NodeInfo) (*proto.ActionStatus, error) {
	logrus.Infof("Received a node deregistration with name : %s", in.NodeID)

	c.NIStorage.Lock()
	defer c.NIStorage.Unlock()

	if _, present := c.NIStorage.Get(in.NodeID); present {
		err := c.PersistenceLayer.DeleteWorkerNodeInformation(context.Background(), in.NodeID)
		if err != nil {
			logrus.Errorf("Failed to disconnect registered worker (error : %s)", err.Error())
			return &proto.ActionStatus{Success: false}, err
		}

		c.removeEndpointsAssociatedWithNode(in.NodeID)
		c.NIStorage.Remove(in.NodeID)

		logrus.Info("Node '", in.NodeID, "' has been successfully deregistered with the control plane")
		return &proto.ActionStatus{Success: true}, nil
	}

	return &proto.ActionStatus{
		Success: false,
		Message: "Node registration failed. Node doesn't exists.",
	}, nil
}

func (c *ControlPlane) nodeHeartbeat(_ context.Context, in *proto.NodeHeartbeatMessage) (*proto.ActionStatus, error) {
	// TODO: conscious concurrency bug
	//c.NIStorage.Lock()
	//defer c.NIStorage.Unlock()

	if present := c.NIStorage.Present(in.NodeID); !present {
		logrus.Debug("Received a heartbeat for non-registered node")
		return &proto.ActionStatus{Success: false}, nil
	}

	c.NIStorage.GetMap()[in.NodeID].UpdateLastHearBeat()
	c.NIStorage.GetMap()[in.NodeID].SetCpuUsed(in.CpuUsage)
	c.NIStorage.GetMap()[in.NodeID].SetMemoryUsed(in.MemoryUsage)
	c.NIStorage.GetMap()[in.NodeID].SetSchedulability(true)

	logrus.Tracef("Heartbeat received from %s with %d milli-CPU used and %d MiB memory used", in.NodeID, in.CpuUsage, in.MemoryUsage)

	return &proto.ActionStatus{Success: true}, nil
}

// Service functions

func (c *ControlPlane) registerService(ctx context.Context, serviceInfo *proto.ServiceInfo) (*proto.ActionStatus, error) {
	logrus.Infof("Received a service registration with name : %s", serviceInfo.Name)

	err := c.prepullImagesForService(ctx, serviceInfo.PrepullConfig, serviceInfo.Image)
	if err != nil {
		logrus.Warnf("Failed to prepull images: %s", err.Error())
		return &proto.ActionStatus{Success: false}, err
	}

	c.SIStorage.Lock()
	defer c.SIStorage.Unlock()

	if _, present := c.SIStorage.Get(serviceInfo.Name); present {
		logrus.Errorf("Service with name %s is already registered", serviceInfo.Name)
		return &proto.ActionStatus{Success: false}, errors.New("service is already registered")
	}

	err = c.PersistenceLayer.StoreServiceInformation(ctx, serviceInfo)
	if err != nil {
		logrus.Errorf("Failed to store information to persistence layer (error : %s)", err.Error())
		c.SIStorage.Remove(serviceInfo.Name)
		return &proto.ActionStatus{Success: false}, err
	}

	err = c.notifyDataplanesAndStartScalingLoop(ctx, serviceInfo)
	if err != nil {
		logrus.Warnf("Failed to connect registered service (error : %s)", err.Error())
		c.SIStorage.Remove(serviceInfo.Name)

		return &proto.ActionStatus{Success: false}, err
	}

	if c.Config.PrecreateSnapshots {
		c.precreateSnapshots(serviceInfo)
	}

	logrus.Debugf("Successfully registered function %s.", serviceInfo.Name)
	return &proto.ActionStatus{Success: true}, nil
}

func (c *ControlPlane) deregisterService(ctx context.Context, serviceInfo *proto.ServiceInfo) (*proto.ActionStatus, error) {
	logrus.Infof("Received a service deregistration with name : %s", serviceInfo.Name)

	c.SIStorage.Lock()
	defer c.SIStorage.Unlock()

	if service, ok := c.SIStorage.Get(serviceInfo.Name); ok {

		err := c.PersistenceLayer.DeleteServiceInformation(ctx, serviceInfo)
		if err != nil {
			logrus.Errorf("Failed to delete information to persistence layer (error : %s)", err.Error())
			return &proto.ActionStatus{Success: false}, err
		}

		err = c.removeServiceFromDataplane(ctx, serviceInfo)
		if err != nil {
			logrus.Warnf("Failed to connect registered service (error : %s)", err.Error())
			return &proto.ActionStatus{Success: false}, err
		}

		close(service.ServiceState.DesiredStateChannel)
		c.SIStorage.Remove(serviceInfo.Name)

		c.autoscalingManager.Stop(serviceInfo.Name)

		return &proto.ActionStatus{Success: true}, nil
	}

	logrus.Errorf("Service with name %s is not registered", serviceInfo.Name)
	return &proto.ActionStatus{Success: false}, errors.New("service is not registered")
}

func (c *ControlPlane) listServices(_ context.Context, _ *emptypb.Empty) (*proto.ServiceList, error) {
	c.SIStorage.RLock()
	defer c.SIStorage.RUnlock()
	return &proto.ServiceList{Service: _map.Keys(c.SIStorage.GetMap())}, nil
}

func (c *ControlPlane) hasService(_ context.Context, service *proto.ServiceIdentifier) (*proto.HasServiceResult, error) {
	return &proto.HasServiceResult{HasService: c.SIStorage.Present(service.Name)}, nil
}

func (c *ControlPlane) setInvocationsMetrics(_ context.Context, metric *proto.AutoscalingMetric) (*proto.ActionStatus, error) {
	storage, ok := c.SIStorage.AtomicGet(metric.ServiceName)
	if !ok {
		logrus.Warn("SIStorage does not exist for '", metric.ServiceName, "'")
		return &proto.ActionStatus{Success: false}, nil
	}

	previousValue := storage.ServiceState.CachedScalingMetrics

	storage.ServiceState.SetCachedScalingMetrics(metric)

	logrus.Debug("Scaling metric for '", storage.ServiceState.ServiceName, "' is ", metric.InflightRequests)

	// Notify autoscaler we received metrics
	c.autoscalingManager.Poke(metric.ServiceName, previousValue)

	return &proto.ActionStatus{Success: true}, nil
}

func (c *ControlPlane) setBackgroundMetrics(_ context.Context, in *proto.MetricsPredictiveAutoscaler) (*proto.ActionStatus, error) {
	if err := c.autoscalingManager.ForwardDataplaneMetrics(in); err != nil {
		logrus.Errorf("Failed to forward dataplane metrics (error : %s)", err.Error())
		return &proto.ActionStatus{Success: false}, err
	}

	return &proto.ActionStatus{Success: true}, nil
}

// Workflow functions

// registerWorkflowTask assumes that a lock on WIStorage was acquired beforehand
func (c *ControlPlane) registerWorkflowTask(ctx context.Context, taskInfo *proto.WorkflowTaskInfo, parentWorkflow string) error {

	if _, present := c.WIStorage.Get(taskInfo.Name); present {
		logrus.Errorf("Workflow object with name %s is already registered", taskInfo.Name)
		return errors.New("workflow object is already registered")
	}

	c.SIStorage.Lock()
	defer c.SIStorage.Unlock()

	taskFunctions := make([]*proto.ServiceInfo, len(taskInfo.Functions))
	for i, taskFuncName := range taskInfo.Functions {
		taskPlacer, registered := c.SIStorage.Get(taskFuncName)
		if !registered {
			return fmt.Errorf("task function with name %s is not registered", taskFuncName)
		}
		taskFunctions[i] = taskPlacer.ServiceState.FunctionInfo[0]
	}

	err := c.PersistenceLayer.StoreWorkflowTaskInformation(ctx, taskInfo)
	if err != nil {
		logrus.Errorf("Failed to store information to persistence layer (error : %s)", err.Error())
		c.WIStorage.Remove(taskInfo.Name)
		return err
	}

	functionState := service_state.NewTaskState(taskInfo, taskFunctions)
	c.autoscalingManager.Create(functionState)
	placementPolicy, evictionPolicy := parsePlacementEvictionPolicies(c.Config)

	c.SIStorage.Set(taskInfo.Name, &endpoint_placer.EndpointPlacer{
		ServiceState:            functionState,
		ColdStartTracingChannel: c.ColdStartTracing.InputChannel,
		PlacementPolicy:         placementPolicy,
		EvictionPolicy:          evictionPolicy,
		PersistenceLayer:        c.PersistenceLayer,
		NIStorage:               c.NIStorage,
		DataPlaneConnections:    c.DataPlaneConnections,
		DandelionNodes:          synchronization.NewControlPlaneSyncStructure[string, bool](),
	})

	go c.SIStorage.GetNoCheck(taskInfo.Name).ScalingControllerLoop()

	c.WIStorage.Set(taskInfo.Name, workflow.NewTask(parentWorkflow))

	return nil
}

// deregisterWorkflowTask assumes that a lock on WIStorage was acquired beforehand
func (c *ControlPlane) deregisterWorkflowTask(ctx context.Context, taskName string) error {
	if st, ok := c.WIStorage.Get(taskName); ok && st.IsTask() {
		err := c.PersistenceLayer.DeleteWorkflowTaskInformation(ctx, taskName)
		if err != nil {
			logrus.Errorf("Failed to delete information to persistence layer (error : %s)", err.Error())
			return err
		}

		taskPlacer := c.SIStorage.GetNoCheck(taskName)
		close(taskPlacer.ServiceState.DesiredStateChannel)
		c.SIStorage.Remove(taskName)
		c.autoscalingManager.Stop(taskName)

		c.WIStorage.Remove(taskName)

		return nil
	}

	logrus.Errorf("No workflow task with name %s registered", taskName)
	return errors.New("workflow task is not registered")
}

func (c *ControlPlane) registerWorkflow(ctx context.Context, wfInfo *proto.WorkflowInfo) (*proto.ActionStatus, error) {
	logrus.Infof("Received a workflow registration with name : %s", wfInfo.Name)

	c.WIStorage.Lock()
	defer c.WIStorage.Unlock()

	if _, present := c.WIStorage.Get(wfInfo.Name); present {
		logrus.Errorf("Workflow object with name %s is already registered", wfInfo.Name)
		return &proto.ActionStatus{Success: false}, errors.New("workflow object is already registered")
	}

	// register workflow tasks
	taskTracker := workflow.NewWorkflow(wfInfo)
	var err error
	var errIdx int
	for taskIdx, task := range wfInfo.Tasks {
		err = c.registerWorkflowTask(ctx, task, wfInfo.Name)
		taskTracker.SetTask(taskIdx, task.Name)
		if err != nil {
			errIdx = taskIdx
			break
		}
	}

	// register workflow
	if err == nil {
		err = c.PersistenceLayer.StoreWorkflowInformation(ctx, wfInfo)
		if err != nil {
			logrus.Errorf("Failed to store information to persistence layer (error : %s)", err.Error())
			c.WIStorage.Remove(wfInfo.Name)
			errIdx = len(wfInfo.Tasks)
		}
	}

	// cleanup if something failed
	if err != nil {
		logrus.Errorf("Failed to register workflow task '%s' (error : %s)", wfInfo.Name, err.Error())
		for _, task := range wfInfo.Tasks[:errIdx] {
			err = c.deregisterWorkflowTask(ctx, task.Name)
			if err != nil {
				logrus.Errorf("Failed to deregister workflow task '%s' (error : %s)", task.Name, err.Error())
			}
		}
		return &proto.ActionStatus{Success: false}, err
	}

	// notify dataplane(s)
	c.DataPlaneConnections.Lock()
	for _, conn := range c.DataPlaneConnections.GetMap() {
		_, err = conn.AddWorkflowDeployment(ctx, wfInfo)
		if err != nil {
			logrus.Warnf("Failed to add deployment to data plane %s - %v", conn.GetIP(), err)
		}
	}
	c.DataPlaneConnections.Unlock()

	c.WIStorage.Set(wfInfo.Name, taskTracker)

	return &proto.ActionStatus{Success: true}, nil
}

func (c *ControlPlane) deregisterWorkflow(ctx context.Context, wfId *proto.WorkflowObjectIdentifier) (*proto.ActionStatus, error) {
	logrus.Infof("Received a workflow deregistration with name : %s", wfId.Name)

	c.WIStorage.Lock()
	defer c.WIStorage.Unlock()

	if st, ok := c.WIStorage.Get(wfId.Name); ok && st.IsWorkflow() {

		// TODO: improve this (if tasks cannot be deleted they will remain in persistence without a reference)
		var err error
		wfTasks := st.GetTasks()
		for _, task := range wfTasks {
			tmpErr := c.PersistenceLayer.DeleteWorkflowTaskInformation(ctx, task)
			if tmpErr != nil {
				err = tmpErr
				logrus.Errorf("Failed to delete workflow task information for task '%s' in persistence layer (error : %s)", task, err.Error())
			}
		}

		err = c.PersistenceLayer.DeleteWorkflowInformation(ctx, wfId.Name)
		if err != nil {
			logrus.Errorf("Failed to delete workflow information for workflow '%s' in persistence layer (error : %s)", wfId.Name, err.Error())
			return &proto.ActionStatus{Success: false}, err
		}

		// notify dataplane(s)
		c.DataPlaneConnections.Lock()
		for _, conn := range c.DataPlaneConnections.GetMap() {
			_, err = conn.DeleteWorkflowDeployment(ctx, wfId)
			if err != nil {
				logrus.Warnf("Failed to add deployment to data plane %s - %v", conn.GetIP(), err)
			}
		}
		c.DataPlaneConnections.Unlock()

		c.WIStorage.Remove(wfId.Name)

		return &proto.ActionStatus{Success: true}, nil
	}

	logrus.Errorf("No workflow with name %s registered", wfId.Name)
	return &proto.ActionStatus{Success: false}, errors.New("workflow is not registered")
}

// Monitoring

func (c *ControlPlane) startNodeMonitoring() chan struct{} {
	stopCh := make(chan struct{})
	go c.checkPeriodicallyWorkerNodes(stopCh)

	return stopCh
}

func (c *ControlPlane) checkPeriodicallyWorkerNodes(stopCh chan struct{}) {
	for {
		select {
		case <-time.After(utils.HeartbeatInterval):
			var events []*proto.Failure

			c.NIStorage.Lock()
			for _, workerNode := range c.NIStorage.GetMap() {
				workerNode.SetSchedulability(true)

				if time.Since(workerNode.GetLastHeartBeat()) > utils.TolerateHeartbeatMisses*utils.HeartbeatInterval {
					// Propagate endpoint removal from the data planes
					events = append(events, c.createWorkerNodeFailureEvents(workerNode)...)
					workerNode.SetSchedulability(false)

					logrus.Warnf("Node %s is unschedulable", workerNode.GetName())
				}
			}
			c.NIStorage.Unlock()

			// the following call requires a lock on NIStorage
			c.HandleFailure(events)
		case <-stopCh:
			logrus.Infof("Stopping node monitoring from the previous leader's term.")
			close(stopCh)

			return
		}
	}
}

func (c *ControlPlane) checkPeriodicallyDataplanes() {
	for {
		c.DataPlaneConnections.Lock()
		for _, dataplaneConnection := range c.DataPlaneConnections.GetMap() {
			if time.Since(dataplaneConnection.GetLastHeartBeat()) > utils.TolerateHeartbeatMisses*utils.HeartbeatInterval {
				apiPort, _ := strconv.ParseInt(dataplaneConnection.GetApiPort(), 10, 64)
				proxyPort, _ := strconv.ParseInt(dataplaneConnection.GetProxyPort(), 10, 64)

				// Trigger a dataplane deregistration
				_, err := c.deregisterDataplane(context.Background(), &proto.DataplaneInfo{
					APIPort:   int32(apiPort),
					ProxyPort: int32(proxyPort),
				})

				if err != nil {
					logrus.Errorf("Failed to deregister dataplane (error : %s)", err.Error())
				}
			}
		}
		c.DataPlaneConnections.Unlock()
		time.Sleep(utils.HeartbeatInterval)
	}
}

// Fault detection

func (c *ControlPlane) createWorkerNodeFailureEvents(wn core.WorkerNodeInterface) []*proto.Failure {
	var failures []*proto.Failure
	c.SIStorage.Lock()
	defer c.SIStorage.Unlock()

	for _, value := range c.SIStorage.GetMap() {
		failureMetadata := &proto.Failure{
			Type:        proto.FailureType_WORKER_NODE_FAILURE,
			ServiceName: value.ServiceState.ServiceName,
			SandboxIDs:  c.getServicesOnWorkerNode(value, wn),
		}

		if len(failureMetadata.SandboxIDs) > 0 {
			failures = append(failures, failureMetadata)
		}
	}

	return failures
}

func (c *ControlPlane) getServicesOnWorkerNode(_ *endpoint_placer.EndpointPlacer, wn core.WorkerNodeInterface) []string {
	toRemove, ok := c.NIStorage.AtomicGet(wn.GetName())
	if !ok {
		return []string{}
	}

	var cIDs []string

	toRemove.GetEndpointMap().RLock()
	for key := range toRemove.GetEndpointMap().GetMap() {
		cIDs = append(cIDs, key.SandboxID)
	}
	toRemove.GetEndpointMap().RUnlock()

	return cIDs
}

// precreateSnapshots assumes that a lock on SIStorage was acquired beforehand
func (c *ControlPlane) precreateSnapshots(info *proto.ServiceInfo) {
	ss, ok := c.SIStorage.Get(info.Name)
	if !ok {
		logrus.Errorf("Cannot find information in SIStorage for %s", info.Name)
		return
	}

	c.NIStorage.RLock()
	defer c.NIStorage.RUnlock()

	wg := &sync.WaitGroup{}
	for _, node := range c.NIStorage.GetMap() {
		wg.Add(1)

		go func(node core.WorkerNodeInterface) {
			defer wg.Done()

			var sandboxInfo *proto.SandboxCreationStatus
			var err error
			if ss.ServiceState.TaskInfo == nil {
				sandboxInfo, err = node.CreateSandbox(context.Background(), ss.ServiceState.FunctionInfo[0])
			} else {
				sandboxInfo, err = node.CreateTaskSandbox(context.Background(), ss.ServiceState.TaskInfo)
			}
			if err != nil {
				logrus.Warnf("Failed to create a image prewarming sandbox for function %s on node %s.", info.Name, node.GetName())
				return
			}

			c.imageStorage.Lock()
			err = c.imageStorage.RegisterWithFetch(ss.FunctionState.ServiceInfo.Image, node)
			c.imageStorage.Unlock()
			if err != nil {
				logrus.Warnf("Failed to fetch image size for %s on node %s while precreating snapshots: %s", ss.FunctionState.ServiceInfo.Image, node.GetName(), err.Error())
			}
			msg, err := node.DeleteSandbox(context.Background(), &proto.SandboxID{
				ID:       sandboxInfo.ID,
				HostPort: sandboxInfo.PortMappings.HostPort,
			})
			if err != nil || !msg.Success {
				logrus.Warnf("Failed to delete an image prewarming sandbox for function %s on node %s.", info.Name, node.GetName())
			}

			logrus.Debugf("Successfully created an image prewarming sandbox for function %s on node %s.", info.Name, node.GetName())
		}(node)
	}

	wg.Wait()
}

func (c *ControlPlane) stopAllScalingLoops() {
	c.SIStorage.Lock()
	defer c.SIStorage.Unlock()

	for name, _ := range c.SIStorage.GetMap() {
		c.autoscalingManager.Stop(name)
	}
}

func (c *ControlPlane) reviseDataplanesInLB(callback func([]string) bool) bool {
	c.DataPlaneConnections.RLock()
	defer c.DataPlaneConnections.RUnlock()

	var dataplanes []string
	for _, dp := range c.DataPlaneConnections.GetMap() {
		dataplanes = append(dataplanes, fmt.Sprintf("%s:%s", dp.GetIP(), dp.GetProxyPort()))
	}

	return callback(dataplanes)
}

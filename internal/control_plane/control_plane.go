package control_plane

import (
	"cluster_manager/api/proto"
	"cluster_manager/internal/control_plane/autoscaling"
	"cluster_manager/internal/control_plane/core"
	"cluster_manager/internal/control_plane/persistence"
	"cluster_manager/internal/control_plane/placement_policy"
	"cluster_manager/pkg/atomic_map"
	config2 "cluster_manager/pkg/config"
	"cluster_manager/pkg/tracing"
	"cluster_manager/pkg/utils"
	"context"
	"errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/emptypb"
	"strconv"
	"sync/atomic"
	"time"
)

type ControlPlane struct {
	DataPlaneConnections *atomic_map.AtomicMap[string, core.DataPlaneInterface]

	NIStorage *atomic_map.AtomicMap[string, core.WorkerNodeInterface]
	SIStorage *atomic_map.AtomicMap[string, *ServiceInfoStorage]

	WorkerEndpoints *atomic_map.AtomicMap[string, *atomic_map.AtomicMap[*core.Endpoint, string]]

	ColdStartTracing *tracing.TracingService[tracing.ColdStartLogEntry] `json:"-"`
	PlacementPolicy  placement_policy.PlacementPolicy
	PersistenceLayer persistence.PersistenceLayer

	dataPlaneCreator  core.DataplaneFactory
	workerNodeCreator core.WorkerNodeFactory
}

func NewControlPlane(client persistence.PersistenceLayer, outputFile string, placementPolicy placement_policy.PlacementPolicy, dataplaneCreator core.DataplaneFactory, workerNodeCreator core.WorkerNodeFactory) *ControlPlane {
	return &ControlPlane{
		NIStorage:            atomic_map.NewAtomicMap[string, core.WorkerNodeInterface](),
		SIStorage:            atomic_map.NewAtomicMap[string, *ServiceInfoStorage](),
		DataPlaneConnections: atomic_map.NewAtomicMap[string, core.DataPlaneInterface](),
		ColdStartTracing:     tracing.NewColdStartTracingService(outputFile),
		PlacementPolicy:      placementPolicy,
		PersistenceLayer:     client,
		WorkerEndpoints:      atomic_map.NewAtomicMap[string, *atomic_map.AtomicMap[*core.Endpoint, string]](),
		dataPlaneCreator:     dataplaneCreator,
		workerNodeCreator:    workerNodeCreator,
	}
}

func (c *ControlPlane) GetNumberConnectedWorkers() int {
	return c.NIStorage.Len()
}

func (c *ControlPlane) GetNumberDataplanes() int {
	return c.DataPlaneConnections.Len()
}

func (c *ControlPlane) GetNumberServices() int {
	return c.SIStorage.Len()
}

func (c *ControlPlane) getServicesOnWorkerNode(ss *ServiceInfoStorage, wn core.WorkerNodeInterface) []string {
	toRemove, ok := ss.WorkerEndpoints.Get(wn.GetName())
	if !ok {
		return []string{}
	}

	var cIDs []string
	for _, v := range toRemove.Keys() {
		cIDs = append(cIDs, v.SandboxID)
	}

	return cIDs
}

func (c *ControlPlane) createWorkerNodeFailureEvents(wn core.WorkerNodeInterface) []*proto.Failure {
	var failures []*proto.Failure
	c.SIStorage.Range(func(key, value any) bool {
		ss := value.(*ServiceInfoStorage)

		failureMetadata := &proto.Failure{
			Type:        proto.FailureType_WORKER_NODE_FAILURE,
			ServiceName: ss.ServiceInfo.Name,
			SandboxIDs:  c.getServicesOnWorkerNode(ss, wn),
		}

		if len(failureMetadata.SandboxIDs) > 0 {
			failures = append(failures, failureMetadata)
		}

		return true
	})

	return failures
}

func (c *ControlPlane) handleNodeFailure(wn core.WorkerNodeInterface) {
	c.HandleFailure(c.createWorkerNodeFailureEvents(wn))
}

func (c *ControlPlane) CheckPeriodicallyWorkerNodes() {
	for {
		for _, workerNode := range c.NIStorage.Values() {
			if time.Since(workerNode.GetLastHeartBeat()) > utils.TolerateHeartbeatMisses*utils.HeartbeatInterval {
				// Propagate endpoint removal from the data planes
				c.handleNodeFailure(workerNode)

				// Trigger a node deregistration
				err := c.deregisterNode(workerNode)
				if err != nil {
					logrus.Errorf("Failed to deregister node (error : %s)", err.Error())
				}
			}
		}

		time.Sleep(utils.HeartbeatInterval)
	}
}

func (c *ControlPlane) CheckPeriodicallyDataplanes() {
	for {
		for _, datapalaneConnection := range c.DataPlaneConnections.Values() {
			if time.Since(datapalaneConnection.GetLastHeartBeat()) > utils.TolerateHeartbeatMisses*utils.HeartbeatInterval {
				apiPort, _ := strconv.ParseInt(datapalaneConnection.GetApiPort(), 10, 64)
				proxyPort, _ := strconv.ParseInt(datapalaneConnection.GetProxyPort(), 10, 64)
				// Trigger a dataplane deregistration
				_, err := c.DeregisterDataplane(context.Background(), &proto.DataplaneInfo{
					APIPort:   int32(apiPort),
					ProxyPort: int32(proxyPort),
				})

				if err != nil {
					logrus.Errorf("Failed to deregister dataplane (error : %s)", err.Error())
				}
			}
		}

		time.Sleep(utils.HeartbeatInterval)
	}
}

func (c *ControlPlane) OnMetricsReceive(_ context.Context, metric *proto.AutoscalingMetric) (*proto.ActionStatus, error) {
	storage, ok := c.SIStorage.Get(metric.ServiceName)
	if !ok {
		logrus.Warn("SIStorage does not exist for '", metric.ServiceName, "'")
		return &proto.ActionStatus{Success: false}, nil
	}

	storage.Controller.ScalingMetadata.SetCachedScalingMetric(float64(metric.Metric))
	logrus.Debug("Scaling metric for '", storage.ServiceInfo.Name, "' is ", metric.Metric)

	storage.Controller.Start()

	return &proto.ActionStatus{Success: true}, nil
}

func (c *ControlPlane) ListServices(_ context.Context, _ *emptypb.Empty) (*proto.ServiceList, error) {
	return &proto.ServiceList{Service: c.SIStorage.Keys()}, nil
}

func (c *ControlPlane) RegisterNode(ctx context.Context, in *proto.NodeInfo) (*proto.ActionStatus, error) {
	_, ok := c.NIStorage.Get(in.NodeID)
	if ok {
		return &proto.ActionStatus{
			Success: false,
			Message: "Node registration failed. Node with the same name already exists.",
		}, nil
	}

	wn := c.workerNodeCreator(core.WorkerNodeConfiguration{
		Name:     in.NodeID,
		IP:       in.IP,
		Port:     strconv.Itoa(int(in.Port)),
		CpuCores: in.CpuCores,
		Memory:   in.MemorySize,
	})

	err := c.PersistenceLayer.StoreWorkerNodeInformation(ctx, &proto.WorkerNodeInformation{
		Name:     in.NodeID,
		Ip:       in.IP,
		Port:     strconv.Itoa(int(in.Port)),
		CpuCores: in.CpuCores,
		Memory:   in.MemorySize,
	})
	if err != nil {
		logrus.Errorf("Failed to store information to persistence layer (error : %s)", err.Error())
		return &proto.ActionStatus{Success: false}, err
	}

	c.connectToRegisteredWorker(wn)

	logrus.Info("Node '", in.NodeID, "' has been successfully register with the control plane")

	return &proto.ActionStatus{Success: true}, nil
}

func (c *ControlPlane) connectToRegisteredWorker(wn core.WorkerNodeInterface) {
	c.NIStorage.Set(wn.GetName(), wn)
	c.WorkerEndpoints.Set(wn.GetName(), atomic_map.NewAtomicMap[*core.Endpoint, string]())

	go wn.GetAPI()
}

func (c *ControlPlane) DeregisterNode(_ context.Context, in *proto.NodeInfo) (*proto.ActionStatus, error) {
	_, ok := c.NIStorage.Get(in.NodeID)
	if !ok {
		return &proto.ActionStatus{
			Success: false,
			Message: "Node registration failed. Node doesn't exists.",
		}, nil
	}

	wn := c.workerNodeCreator(core.WorkerNodeConfiguration{
		Name:     in.NodeID,
		IP:       in.NodeID,
		Port:     strconv.Itoa(int(in.Port)),
		CpuCores: in.CpuCores,
		Memory:   in.MemorySize,
	})

	err := c.deregisterNode(wn)
	if err != nil {
		logrus.Errorf("Failed to disconnect registered worker (error : %s)", err.Error())
		return &proto.ActionStatus{Success: false}, err
	}

	logrus.Info("Node '", in.NodeID, "' has been successfully deregistered with the control plane")

	return &proto.ActionStatus{Success: true}, nil
}

func (c *ControlPlane) deregisterNode(wn core.WorkerNodeInterface) error {
	err := c.PersistenceLayer.DeleteWorkerNodeInformation(context.Background(), &proto.WorkerNodeInformation{
		Name:     wn.GetName(),
		Ip:       wn.GetIP(),
		Port:     wn.GetPort(),
		CpuCores: wn.GetCpuCores(),
		Memory:   wn.GetMemory(),
	})
	if err != nil {
		return err
	}

	c.disconnectRegisteredWorker(wn)

	return nil
}

func (c *ControlPlane) disconnectRegisteredWorker(wn core.WorkerNodeInterface) {
	c.NIStorage.Delete(wn.GetName())
}

func (c *ControlPlane) NodeHeartbeat(_ context.Context, in *proto.NodeHeartbeatMessage) (*proto.ActionStatus, error) {
	n, ok := c.NIStorage.Get(in.NodeID)
	if !ok {
		logrus.Debug("Received a heartbeat for non-registered node")

		return &proto.ActionStatus{Success: false}, nil
	}

	c.updateWorkerNodeInformation(n, in)

	logrus.Debugf("Heartbeat received from %s with %d percent cpu usage and %d percent memory usage", in.NodeID, in.CpuUsage, in.MemoryUsage)

	return &proto.ActionStatus{Success: true}, nil
}

func (c *ControlPlane) updateWorkerNodeInformation(workerNode core.WorkerNodeInterface, in *proto.NodeHeartbeatMessage) {
	workerNode.UpdateLastHearBeat()
	workerNode.SetCpuUsage(in.CpuUsage)
	workerNode.SetMemoryUsage(in.MemoryUsage)
}

func (c *ControlPlane) RegisterService(ctx context.Context, serviceInfo *proto.ServiceInfo) (*proto.ActionStatus, error) {
	_, ok := c.SIStorage.Get(serviceInfo.Name)
	if ok {
		logrus.Errorf("Service with name %s is already registered", serviceInfo.Name)
		return &proto.ActionStatus{Success: false}, errors.New("service is already registered")
	}

	err := c.PersistenceLayer.StoreServiceInformation(ctx, serviceInfo)
	if err != nil {
		logrus.Errorf("Failed to store information to persistence layer (error : %s)", err.Error())
		return &proto.ActionStatus{Success: false}, err
	}

	err = c.connectToRegisteredService(ctx, serviceInfo, false)
	if err != nil {
		logrus.Warnf("Failed to connect registered service (error : %s)", err.Error())
		return &proto.ActionStatus{Success: false}, err
	}

	return &proto.ActionStatus{Success: true}, nil
}

func (c *ControlPlane) connectToRegisteredService(ctx context.Context, serviceInfo *proto.ServiceInfo, reconstructFromPersistence bool) error {
	for _, conn := range c.DataPlaneConnections.Values() {
		_, err := conn.AddDeployment(ctx, serviceInfo)
		if err != nil {
			return err
		}
	}

	startTime := time.Time{}
	if reconstructFromPersistence {
		startTime = time.Now()
	}

	scalingChannel := make(chan int)

	service := &ServiceInfoStorage{
		ServiceInfo:             serviceInfo,
		ControlPlane:            c,
		Controller:              autoscaling.NewPerFunctionStateController(&scalingChannel, serviceInfo),
		ColdStartTracingChannel: &c.ColdStartTracing.InputChannel,
		PlacementPolicy:         c.PlacementPolicy,
		PersistenceLayer:        c.PersistenceLayer,
		WorkerEndpoints:         c.WorkerEndpoints,
		StartTime:               startTime,
	}

	c.SIStorage.Set(serviceInfo.Name, service)

	go service.ScalingControllerLoop(c.NIStorage, c.DataPlaneConnections)

	return nil
}

func (c *ControlPlane) processDataplaneRequest(_ context.Context, in *proto.DataplaneInfo) (proto.DataplaneInformation, error) {
	return proto.DataplaneInformation{
		Address:   in.IP,
		ApiPort:   strconv.Itoa(int(in.APIPort)),
		ProxyPort: strconv.Itoa(int(in.ProxyPort)),
	}, nil
}

func (c *ControlPlane) RegisterDataplane(ctx context.Context, in *proto.DataplaneInfo) (*proto.ActionStatus, error) {
	logrus.Trace("Received a control plane registration")

	dataplaneInfo, err := c.processDataplaneRequest(ctx, in)
	if err != nil {
		return &proto.ActionStatus{Success: false}, err
	}

	if found := c.DataPlaneConnections.Find(dataplaneInfo.Address); found {
		logrus.Debugf("Dataplane with ip %s already registered - update last heartbeat timestamp", dataplaneInfo.Address)
		dataplane, _ := c.DataPlaneConnections.Get(dataplaneInfo.Address)
		dataplane.UpdateHeartBeat()
		return &proto.ActionStatus{Success: true}, err
	}

	err = c.PersistenceLayer.StoreDataPlaneInformation(ctx, &dataplaneInfo)
	if err != nil {
		logrus.Errorf("Failed to store information to persistence layer (error : %s)", err.Error())
		return &proto.ActionStatus{Success: false}, err
	}

	dataplaneConnection := c.dataPlaneCreator(dataplaneInfo.Address, dataplaneInfo.ApiPort, dataplaneInfo.ProxyPort)

	err = dataplaneConnection.InitializeDataPlaneConnection(dataplaneInfo.Address, dataplaneInfo.ApiPort)
	if err != nil {
		return &proto.ActionStatus{Success: false}, err
	}

	c.DataPlaneConnections.Set(dataplaneInfo.Address, dataplaneConnection)

	return &proto.ActionStatus{Success: true}, nil
}

func (c *ControlPlane) DeregisterDataplane(ctx context.Context, in *proto.DataplaneInfo) (*proto.ActionStatus, error) {
	logrus.Trace("Received a data plane deregistration")

	dataplaneInfo, err := c.processDataplaneRequest(ctx, in)
	if err != nil {
		return &proto.ActionStatus{Success: false}, err
	}

	err = c.PersistenceLayer.DeleteDataPlaneInformation(ctx, &dataplaneInfo)
	if err != nil {
		logrus.Errorf("Failed to store information to persistence layer (error : %s)", err.Error())
		return &proto.ActionStatus{Success: false}, err
	}

	c.deregisterDataplane(&dataplaneInfo)

	return &proto.ActionStatus{Success: true}, nil
}

func (c *ControlPlane) deregisterDataplane(information *proto.DataplaneInformation) {
	c.DataPlaneConnections.Delete(information.Address)
}

func (c *ControlPlane) ReconstructState(ctx context.Context, config config2.ControlPlaneConfig) error {
	if !config.Reconstruct {
		return nil
	}

	{
		start := time.Now()
		if err := c.reconstructDataplaneState(ctx); err != nil {
			return err
		}
		duration := time.Since(start)
		logrus.Infof("Data planes reconstruction took : %s", duration)
	}
	{
		start := time.Now()
		if err := c.reconstructWorkersState(ctx); err != nil {
			return err
		}
		duration := time.Since(start)
		logrus.Infof("Worker nodes reconstruction took : %s", duration)
	}
	{
		start := time.Now()
		if err := c.reconstructServiceState(ctx); err != nil {
			return err
		}
		duration := time.Since(start)
		logrus.Infof("Services reconstruction took : %s", duration)
	}
	{
		start := time.Now()
		if err := c.reconstructEndpointsState(ctx, c.DataPlaneConnections); err != nil {
			return err
		}
		duration := time.Since(start)
		logrus.Infof("Endpoints reconstruction took : %s", duration)
	}

	return nil
}

func (c *ControlPlane) reconstructDataplaneState(ctx context.Context) error {
	dataPlaneValues, err := c.PersistenceLayer.GetDataPlaneInformation(ctx)
	if err != nil {
		return err
	}

	for _, dataplaneInfo := range dataPlaneValues {
		dataplaneConnection := c.dataPlaneCreator(dataplaneInfo.Address, dataplaneInfo.ApiPort, dataplaneInfo.ProxyPort)
		dataplaneConnection.InitializeDataPlaneConnection(dataplaneInfo.Address, dataplaneInfo.ApiPort)

		c.DataPlaneConnections.Set(dataplaneInfo.Address, dataplaneConnection)
	}

	return nil
}

func (c *ControlPlane) reconstructWorkersState(ctx context.Context) error {
	workers, err := c.PersistenceLayer.GetWorkerNodeInformation(ctx)
	if err != nil {
		return err
	}

	for _, worker := range workers {
		c.connectToRegisteredWorker(c.workerNodeCreator(core.WorkerNodeConfiguration{
			Name:     worker.Name,
			IP:       worker.Ip,
			Port:     worker.Port,
			CpuCores: worker.CpuCores,
			Memory:   worker.Memory,
		}))
	}

	return nil
}

func (c *ControlPlane) reconstructServiceState(ctx context.Context) error {
	services, err := c.PersistenceLayer.GetServiceInformation(ctx)
	if err != nil {
		return err
	}

	for _, service := range services {
		err := c.connectToRegisteredService(ctx, service, true)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *ControlPlane) reconstructEndpointsState(ctx context.Context, dpiClients *atomic_map.AtomicMap[string, core.DataPlaneInterface]) error {
	endpoints := make([]*proto.Endpoint, 0)

	for _, workerNode := range c.NIStorage.Values() {
		list, err := workerNode.ListEndpoints(ctx, &emptypb.Empty{})
		if err != nil {
			return err
		}

		endpoints = append(endpoints, list.Endpoint...)
	}

	logrus.Tracef("Found %d endpoints", len(endpoints))

	for _, endpoint := range endpoints {
		controlPlaneEndpoint := &core.Endpoint{
			SandboxID: endpoint.SandboxID,
			URL:       endpoint.URL,
			Node:      c.NIStorage.GetUnsafe(endpoint.NodeName),
			HostPort:  endpoint.HostPort,
		}

		val, found := c.SIStorage.Get(endpoint.ServiceName)
		if !found {
			return errors.New("element not found in map")
		}

		val.reconstructEndpointInController(controlPlaneEndpoint, dpiClients)
	}

	return nil
}

func removeEndpoints(ss *ServiceInfoStorage, dpConns *atomic_map.AtomicMap[string, core.DataPlaneInterface], endpoints []string) {
	ss.Controller.EndpointLock.Lock()
	defer ss.Controller.EndpointLock.Unlock()

	toRemove := make(map[*core.Endpoint]struct{})
	for _, cid := range endpoints {
		endpoint := searchEndpointByContainerName(ss.Controller.Endpoints, cid)
		if endpoint == nil {
			logrus.Warn("Endpoint ", cid, " not found for removal.")
			continue
		}

		toRemove[endpoint] = struct{}{}
		ss.removeEndpointFromWNStruct(endpoint)

		logrus.Warn("Control plane notified of failure of '", endpoint.SandboxID, "'. Decrementing actual scale and removing the endpoint.")
	}

	atomic.AddInt64(&ss.Controller.ScalingMetadata.ActualScale, -int64(len(toRemove)))
	ss.Controller.Endpoints = ss.excludeEndpoints(ss.Controller.Endpoints, toRemove)

	ss.updateEndpoints(dpConns, ss.prepareUrlList())
}

func (c *ControlPlane) HandleFailure(failures []*proto.Failure) bool {
	for _, failure := range failures {
		switch failure.Type {
		case proto.FailureType_SANDBOX_FAILURE, proto.FailureType_WORKER_NODE_FAILURE:
			// RECOVERY ACTION: reduce the actual scale by one and propagate endpoint removal
			serviceName := failure.ServiceName
			sandboxIDs := failure.SandboxIDs

			// TODO: all endpoints removals can be batched into a single call
			if ss, ok := c.SIStorage.Get(serviceName); ok {
				removeEndpoints(ss, c.DataPlaneConnections, sandboxIDs)
			}
		}
	}

	return true
}

func searchEndpointByContainerName(endpoints []*core.Endpoint, cid string) *core.Endpoint {
	for _, e := range endpoints {
		if e.SandboxID == cid {
			return e
		}
	}

	return nil
}

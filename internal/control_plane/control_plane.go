package control_plane

import (
	"cluster_manager/api/proto"
	"cluster_manager/internal/control_plane/core"
	"cluster_manager/internal/control_plane/persistence"
	"cluster_manager/internal/control_plane/placement_policy"
	"cluster_manager/pkg/config"
	_map "cluster_manager/pkg/map"
	"cluster_manager/pkg/synchronization"
	"cluster_manager/pkg/tracing"
	"cluster_manager/pkg/utils"
	"context"
	"errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/emptypb"
	"log"
	"os"
	"strconv"
	"sync"
	"time"
)

type ControlPlane struct {
	DataPlaneConnections synchronization.SyncStructure[string, core.DataPlaneInterface]
	NIStorage            synchronization.SyncStructure[string, core.WorkerNodeInterface]
	SIStorage            synchronization.SyncStructure[string, *ServiceInfoStorage]

	ColdStartTracing *tracing.TracingService[tracing.ColdStartLogEntry] `json:"-"`
	PlacementPolicy  placement_policy.PlacementPolicy
	PersistenceLayer persistence.PersistenceLayer

	dataPlaneCreator  core.DataplaneFactory
	workerNodeCreator core.WorkerNodeFactory

	shouldTrace                    bool
	traceSandboxCreationsInTxtFile *os.File

	config *config.ControlPlaneConfig
}

func NewControlPlane(client persistence.PersistenceLayer, outputFile string, placementPolicy placement_policy.PlacementPolicy,
	dataplaneCreator core.DataplaneFactory, workerNodeCreator core.WorkerNodeFactory, cfg *config.ControlPlaneConfig) *ControlPlane {
	var (
		file *os.File
		err  error
	)

	if cfg.TraceSandboxCreation {
		file, err = os.OpenFile("notes.txt", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
		if err != nil {
			log.Fatal(err)
		}
	}

	return &ControlPlane{
		NIStorage:                      synchronization.NewControlPlaneSyncStructure[string, core.WorkerNodeInterface](),
		SIStorage:                      synchronization.NewControlPlaneSyncStructure[string, *ServiceInfoStorage](),
		DataPlaneConnections:           synchronization.NewControlPlaneSyncStructure[string, core.DataPlaneInterface](),
		ColdStartTracing:               tracing.NewColdStartTracingService(outputFile),
		PlacementPolicy:                placementPolicy,
		PersistenceLayer:               client,
		dataPlaneCreator:               dataplaneCreator,
		workerNodeCreator:              workerNodeCreator,
		shouldTrace:                    cfg.TraceSandboxCreation,
		traceSandboxCreationsInTxtFile: file,
		config:                         cfg,
	}
}

// Dataplanes functions

func (c *ControlPlane) RegisterDataplane(ctx context.Context, in *proto.DataplaneInfo) (*proto.ActionStatus, error) {
	logrus.Tracef("Received a data plane registration with ip : %s", in.IP)

	dataplaneInfo := proto.DataplaneInformation{
		Address:   in.IP,
		ApiPort:   strconv.Itoa(int(in.APIPort)),
		ProxyPort: strconv.Itoa(int(in.ProxyPort)),
	}

	key := in.IP
	dataplaneConnection := c.dataPlaneCreator(dataplaneInfo.Address, dataplaneInfo.ApiPort, dataplaneInfo.ProxyPort)

	if enter, timestamp := c.DataPlaneConnections.SetIfAbsent(key, dataplaneConnection); enter {
		err := c.PersistenceLayer.StoreDataPlaneInformation(ctx, &dataplaneInfo, timestamp)
		if err != nil {
			logrus.Errorf("Failed to store information to persistence layer (error : %s)", err.Error())
			c.DataPlaneConnections.Remove(key)
			return &proto.ActionStatus{Success: false}, err
		}

		err = dataplaneConnection.InitializeDataPlaneConnection(dataplaneInfo.Address, dataplaneInfo.ApiPort)
		if err != nil {
			logrus.Errorf("Failed to initialize dataplane connection (error : %s)", err.Error())
			c.DataPlaneConnections.Remove(key)
			return &proto.ActionStatus{Success: false}, err
		}

		return &proto.ActionStatus{Success: true}, nil
	}

	// If the dataplane is already registered, we update the heartbeat
	logrus.Debugf("Dataplane with ip %s already registered - update last heartbeat timestamp", dataplaneInfo.Address)

	c.DataPlaneConnections.Lock()
	defer c.DataPlaneConnections.Unlock()

	c.DataPlaneConnections.GetMap()[key].UpdateHeartBeat()

	return &proto.ActionStatus{Success: true}, nil
}

func (c *ControlPlane) DeregisterDataplane(ctx context.Context, in *proto.DataplaneInfo) (*proto.ActionStatus, error) {
	logrus.Trace("Received a data plane deregistration")

	dataplaneInfo := proto.DataplaneInformation{
		Address:   in.IP,
		ApiPort:   strconv.Itoa(int(in.APIPort)),
		ProxyPort: strconv.Itoa(int(in.ProxyPort)),
	}

	if enter, timestamp := c.DataPlaneConnections.RemoveIfPresent(dataplaneInfo.Address); enter {
		err := c.PersistenceLayer.DeleteDataPlaneInformation(ctx, &dataplaneInfo, timestamp)
		if err != nil {
			logrus.Errorf("Failed to store information to persistence layer (error : %s)", err.Error())
			return &proto.ActionStatus{Success: false}, err
		}
	}

	return &proto.ActionStatus{Success: true}, nil
}

// Node functions

func (c *ControlPlane) RegisterNode(ctx context.Context, in *proto.NodeInfo) (*proto.ActionStatus, error) {
	wn := c.workerNodeCreator(core.WorkerNodeConfiguration{
		Name:     in.NodeID,
		IP:       in.IP,
		Port:     strconv.Itoa(int(in.Port)),
		CpuCores: in.CpuCores,
		Memory:   in.MemorySize,
	})

	if enter, timestamp := c.NIStorage.SetIfAbsent(in.NodeID, wn); enter {
		err := c.PersistenceLayer.StoreWorkerNodeInformation(ctx, &proto.WorkerNodeInformation{
			Name:     in.NodeID,
			Ip:       in.IP,
			Port:     strconv.Itoa(int(in.Port)),
			CpuCores: in.CpuCores,
			Memory:   in.MemorySize,
		}, timestamp)
		if err != nil {
			logrus.Errorf("Failed to store information to persistence layer (error : %s)", err.Error())
			c.NIStorage.AtomicRemove(in.NodeID)
			return &proto.ActionStatus{Success: false}, err
		}

		go wn.ConnectToWorker()
		wn.SetSchedulability(true)

		logrus.Info("Node '", in.NodeID, "' has been successfully registered with the control plane")

		return &proto.ActionStatus{Success: true}, nil
	}

	return &proto.ActionStatus{
		Success: false,
		Message: "Node registration failed. Node with the same name already exists.",
	}, nil
}

func (c *ControlPlane) DeregisterNode(ctx context.Context, in *proto.NodeInfo) (*proto.ActionStatus, error) {
	if enter, timestamp := c.NIStorage.RemoveIfPresent(in.NodeID); enter {
		err := c.PersistenceLayer.DeleteWorkerNodeInformation(context.Background(), in.NodeID, timestamp)
		if err != nil {
			logrus.Errorf("Failed to disconnect registered worker (error : %s)", err.Error())
			return &proto.ActionStatus{Success: false}, err
		}
		// TODO: Fix concurrency here
		c.removeEndointsAssociatedWithNode(in.NodeID)

		logrus.Info("Node '", in.NodeID, "' has been successfully deregistered with the control plane")
		return &proto.ActionStatus{Success: true}, nil
	}

	return &proto.ActionStatus{
		Success: false,
		Message: "Node registration failed. Node doesn't exists.",
	}, nil
}

func (c *ControlPlane) NodeHeartbeat(_ context.Context, in *proto.NodeHeartbeatMessage) (*proto.ActionStatus, error) {
	// TODO: conscious concurrency bug
	//c.NIStorage.Lock()
	//defer c.NIStorage.Unlock()

	if present := c.NIStorage.Present(in.NodeID); !present {
		logrus.Debug("Received a heartbeat for non-registered node")
		return &proto.ActionStatus{Success: false}, nil
	}

	c.NIStorage.GetMap()[in.NodeID].UpdateLastHearBeat()
	c.NIStorage.GetMap()[in.NodeID].SetCpuUsage(in.CpuUsage)
	c.NIStorage.GetMap()[in.NodeID].SetMemoryUsage(in.MemoryUsage)
	c.NIStorage.GetMap()[in.NodeID].SetSchedulability(true)

	logrus.Debugf("Heartbeat received from %s with %d percent cpu usage and %d percent memory usage", in.NodeID, in.CpuUsage, in.MemoryUsage)

	return &proto.ActionStatus{Success: true}, nil
}

// Service functions

func (c *ControlPlane) RegisterService(ctx context.Context, serviceInfo *proto.ServiceInfo) (*proto.ActionStatus, error) {
	if enter, timestamp := c.SIStorage.SetIfAbsent(serviceInfo.Name, &ServiceInfoStorage{}); enter {
		err := c.PersistenceLayer.StoreServiceInformation(ctx, serviceInfo, timestamp)
		if err != nil {
			logrus.Errorf("Failed to store information to persistence layer (error : %s)", err.Error())
			c.SIStorage.AtomicRemove(serviceInfo.Name)
			return &proto.ActionStatus{Success: false}, err
		}

		err = c.notifyDataplanesAndStartScalingLoop(ctx, serviceInfo, false)
		if err != nil {
			logrus.Warnf("Failed to connect registered service (error : %s)", err.Error())
			c.SIStorage.AtomicRemove(serviceInfo.Name)

			return &proto.ActionStatus{Success: false}, err
		}

		if c.config.PrecreateSnapshots {
			c.precreateSnapshots(serviceInfo)
		}

		return &proto.ActionStatus{Success: true}, nil
	}

	logrus.Errorf("Service with name %s is already registered", serviceInfo.Name)
	return &proto.ActionStatus{Success: false}, errors.New("service is already registered")
}

func (c *ControlPlane) DeregisterService(ctx context.Context, serviceInfo *proto.ServiceInfo) (*proto.ActionStatus, error) {

	if enter, timestamp := c.SIStorage.RemoveIfPresent(serviceInfo.Name); enter {
		err := c.PersistenceLayer.DeleteServiceInformation(ctx, serviceInfo, timestamp)
		if err != nil {
			logrus.Errorf("Failed to delete information to persistence layer (error : %s)", err.Error())
			return &proto.ActionStatus{Success: false}, err
		}

		err = c.removeServiceFromDataplaneAndStopLoop(ctx, serviceInfo, false)
		if err != nil {
			logrus.Warnf("Failed to connect registered service (error : %s)", err.Error())
			c.SIStorage.AtomicRemove(serviceInfo.Name)
			return &proto.ActionStatus{Success: false}, err
		}

		return &proto.ActionStatus{Success: true}, nil
	}

	logrus.Errorf("Service with name %s is not registered", serviceInfo.Name)
	return &proto.ActionStatus{Success: false}, errors.New("service is not registered")
}

func (c *ControlPlane) ListServices(_ context.Context, _ *emptypb.Empty) (*proto.ServiceList, error) {
	c.SIStorage.RLock()
	defer c.SIStorage.RUnlock()
	return &proto.ServiceList{Service: _map.Keys(c.SIStorage.GetMap())}, nil
}

func (c *ControlPlane) OnMetricsReceive(_ context.Context, metric *proto.AutoscalingMetric) (*proto.ActionStatus, error) {
	storage, ok := c.SIStorage.AtomicGet(metric.ServiceName)
	if !ok {
		logrus.Warn("SIStorage does not exist for '", metric.ServiceName, "'")
		return &proto.ActionStatus{Success: false}, nil
	}

	storage.Controller.ScalingMetadata.SetCachedScalingMetric(metric)
	logrus.Debug("Scaling metric for '", storage.ServiceInfo.Name, "' is ", metric.InflightRequests)

	storage.Controller.Start()

	return &proto.ActionStatus{Success: true}, nil
}

// Monitoring

func (c *ControlPlane) CheckPeriodicallyWorkerNodes() {
	for {
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

		time.Sleep(utils.HeartbeatInterval)
	}
}

func (c *ControlPlane) CheckPeriodicallyDataplanes() {
	for {
		c.DataPlaneConnections.Lock()
		for _, datapalaneConnection := range c.DataPlaneConnections.GetMap() {
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
			ServiceName: value.ServiceInfo.Name,
			SandboxIDs:  c.getServicesOnWorkerNode(value, wn),
		}

		if len(failureMetadata.SandboxIDs) > 0 {
			failures = append(failures, failureMetadata)
		}
	}

	return failures
}

func (c *ControlPlane) getServicesOnWorkerNode(ss *ServiceInfoStorage, wn core.WorkerNodeInterface) []string {
	toRemove, ok := c.NIStorage.AtomicGet(wn.GetName())
	if !ok {
		return []string{}
	}

	var cIDs []string

	toRemove.GetEndpointMap().RLock()
	for key, _ := range toRemove.GetEndpointMap().GetMap() {
		cIDs = append(cIDs, key.SandboxID)
	}
	toRemove.GetEndpointMap().RUnlock()

	return cIDs
}

func (c *ControlPlane) precreateSnapshots(info *proto.ServiceInfo) {
	c.SIStorage.Lock()
	defer c.SIStorage.Unlock()

	ss, _ := c.SIStorage.Get(info.Name)

	wg := &sync.WaitGroup{}

	for _, node := range c.NIStorage.GetMap() {
		wg.Add(1)

		go func(node core.WorkerNodeInterface) {
			defer wg.Done()

			sandboxInfo, err := node.CreateSandbox(context.Background(), ss.ServiceInfo)
			if err != nil {
				logrus.Warnf("Failed to create a image prewarming sandbox for function %s on node %s.", info.Name, node.GetName())
				return
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

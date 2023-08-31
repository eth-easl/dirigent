package control_plane

import (
	"cluster_manager/api/proto"
	"cluster_manager/internal/control_plane/persistence"
	"cluster_manager/internal/data_plane/function_metadata"
	"cluster_manager/pkg/atomic_map"
	config2 "cluster_manager/pkg/config"
	"cluster_manager/pkg/grpc_helpers"
	"cluster_manager/pkg/tracing"
	"cluster_manager/pkg/utils"
	"context"
	"errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/emptypb"
	"strconv"
	"sync"
	"time"
)

type ControlPlane struct {
	DataPlaneConnections atomic_map.AtomicMap[string, *function_metadata.DataPlaneConnectionInfo]

	NIStorage atomic_map.AtomicMap[string, *WorkerNode]
	SIStorage atomic_map.AtomicMap[string, *ServiceInfoStorage]
	sisLock   sync.RWMutex

	WorkerEndpoints atomic_map.AtomicMap[string, atomic_map.AtomicMap[*Endpoint, string]]

	ColdStartTracing *tracing.TracingService[tracing.ColdStartLogEntry] `json:"-"`
	PlacementPolicy  PlacementPolicy
	PersistenceLayer persistence.PersistenceLayer
}

func NewControlPlane(client persistence.PersistenceLayer, outputFile string, placementPolicy PlacementPolicy) *ControlPlane {
	return &ControlPlane{
		NIStorage:            atomic_map.NewAtomicMap[string, *WorkerNode](),
		SIStorage:            atomic_map.NewAtomicMap[string, *ServiceInfoStorage](),
		DataPlaneConnections: atomic_map.NewAtomicMap[string, *function_metadata.DataPlaneConnectionInfo](),
		ColdStartTracing:     tracing.NewColdStartTracingService(outputFile),
		PlacementPolicy:      placementPolicy,
		PersistenceLayer:     client,
		WorkerEndpoints:      atomic_map.NewAtomicMap[string, atomic_map.AtomicMap[*Endpoint, string]](),
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

func (c *ControlPlane) CheckPeriodicallyWorkerNodes() {
	for {
		for _, workerNode := range c.NIStorage.Values() {
			if time.Since(workerNode.LastHeartbeat) > 3*utils.HeartbeatInterval {
				// Triger a node deregistation
				err := c.deregisterNode(workerNode)
				if err != nil {
					logrus.Errorf("Failed to deregister node (error : %s)", err.Error())
				}
			}
		}
		time.Sleep(5 * time.Second)
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

	ipAddress, ok := grpc_helpers.GetIPAddressFromGRPCCall(ctx)
	if !ok {
		return &proto.ActionStatus{
			Success: false,
			Message: "Node registration failed. Error getting IP address from context.",
		}, nil
	}

	wn := &WorkerNode{
		Name:          in.NodeID,
		IP:            ipAddress,
		Port:          strconv.Itoa(int(in.Port)),
		CpuCores:      int(in.CpuCores),
		Memory:        int(in.MemorySize),
		LastHeartbeat: time.Now(),
	}

	err := c.PersistenceLayer.StoreWorkerNodeInformation(ctx, &proto.WorkerNodeInformation{
		Name:     wn.Name,
		Ip:       wn.IP,
		Port:     wn.Port,
		CpuCores: int32(wn.CpuCores),
		Memory:   int32(wn.Memory),
	})
	if err != nil {
		logrus.Errorf("Failed to store information to persistence layer (error : %s)", err.Error())
		return &proto.ActionStatus{Success: false}, err
	}

	c.connectToRegisteredWorker(wn)

	logrus.Info("Node '", in.NodeID, "' has been successfully register with the control plane")

	return &proto.ActionStatus{Success: true}, nil
}

func (c *ControlPlane) connectToRegisteredWorker(wn *WorkerNode) {
	c.NIStorage.Set(wn.Name, wn)
	c.WorkerEndpoints.Set(wn.Name, atomic_map.NewAtomicMap[*Endpoint, string]())

	go wn.GetAPI()
}

func (c *ControlPlane) DeregisterNode(ctx context.Context, in *proto.NodeInfo) (*proto.ActionStatus, error) {
	_, ok := c.NIStorage.Get(in.NodeID)
	if !ok {
		return &proto.ActionStatus{
			Success: false,
			Message: "Node registration failed. Node doesn't exists.",
		}, nil
	}

	ipAddress, ok := grpc_helpers.GetIPAddressFromGRPCCall(ctx)
	if !ok {
		return &proto.ActionStatus{
			Success: false,
			Message: "Node registration failed. Error getting IP address from context.",
		}, nil
	}

	wn := &WorkerNode{
		Name:     in.NodeID,
		IP:       ipAddress,
		Port:     strconv.Itoa(int(in.Port)),
		CpuCores: int(in.CpuCores),
		Memory:   int(in.MemorySize),
	}

	err := c.deregisterNode(wn)
	if err != nil {
		logrus.Errorf("Failed to disconnect registered worker (error : %s)", err.Error())
		return &proto.ActionStatus{Success: false}, err
	}

	logrus.Info("Node '", in.NodeID, "' has been successfully deregistered with the control plane")

	return &proto.ActionStatus{Success: true}, nil
}

func (c *ControlPlane) deregisterNode(wn *WorkerNode) error {
	err := c.PersistenceLayer.DeleteWorkerNodeInformation(context.Background(), &proto.WorkerNodeInformation{
		Name:     wn.Name,
		Ip:       wn.IP,
		Port:     wn.Port,
		CpuCores: int32(wn.CpuCores),
		Memory:   int32(wn.Memory),
	})
	if err != nil {
		return err
	}

	c.disconnectRegisteredWorker(wn)

	return nil
}

func (c *ControlPlane) disconnectRegisteredWorker(wn *WorkerNode) {
	c.NIStorage.Delete(wn.Name)
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

func (c *ControlPlane) updateWorkerNodeInformation(workerNode *WorkerNode, in *proto.NodeHeartbeatMessage) {
	workerNode.LastHeartbeat = time.Now()
	workerNode.CpuUsage = int(in.CpuUsage)
	workerNode.MemoryUsage = int(in.MemoryUsage)
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

	for _, conn := range c.DataPlaneConnections.Values() {
		resp, err := conn.Iface.AddDeployment(ctx, serviceInfo)
		if err != nil || !resp.Success {
			logrus.Warn("Failed to propagate service registration to the data plane")
			return &proto.ActionStatus{Success: false}, err
		}
	}

	err = c.connectToRegisteredService(ctx, serviceInfo)
	if err != nil {
		return &proto.ActionStatus{Success: false}, err
	}

	return &proto.ActionStatus{Success: true}, nil
}

func (c *ControlPlane) connectToRegisteredService(ctx context.Context, serviceInfo *proto.ServiceInfo) error {
	scalingChannel := make(chan int)

	service := &ServiceInfoStorage{
		ServiceInfo:             serviceInfo,
		ControlPlane:            c,
		Controller:              NewPerFunctionStateController(&scalingChannel, serviceInfo),
		ColdStartTracingChannel: &c.ColdStartTracing.InputChannel,
		PlacementPolicy:         c.PlacementPolicy,
		PersistenceLayer:        c.PersistenceLayer,
		WorkerEndpoints:         &c.WorkerEndpoints,
	}

	c.SIStorage.Set(serviceInfo.Name, service)

	go service.ScalingControllerLoop(&c.NIStorage, &c.DataPlaneConnections)

	return nil
}

func (c *ControlPlane) processDataplaneRequest(ctx context.Context, in *proto.DataplaneInfo) (proto.DataplaneInformation, error) {
	ipAddress, ok := grpc_helpers.GetIPAddressFromGRPCCall(ctx)
	if !ok {
		logrus.Debug("Failed to extract IP address from data plane registration request")
		return proto.DataplaneInformation{}, errors.New("Failed to extract IP address from data plane registration request")
	}

	return proto.DataplaneInformation{
		Address:   ipAddress,
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
		logrus.Infof("Dataplane with ip %s already registered - request ignored", dataplaneInfo.Address)
		return &proto.ActionStatus{Success: true}, err
	}

	err = c.PersistenceLayer.StoreDataPlaneInformation(ctx, &dataplaneInfo)
	if err != nil {
		logrus.Errorf("Failed to store information to persistence layer (error : %s)", err.Error())
		return &proto.ActionStatus{Success: false}, err
	}

	c.connectToRegisteredDataplane(&dataplaneInfo)

	return &proto.ActionStatus{Success: true}, nil
}

func (c *ControlPlane) connectToRegisteredDataplane(information *proto.DataplaneInformation) {
	c.registerDataplane(
		grpc_helpers.InitializeDataPlaneConnection(information.Address, information.ApiPort),
		information.Address,
		information.ApiPort,
		information.ProxyPort,
	)
}

func (c *ControlPlane) registerDataplane(iface proto.DpiInterfaceClient, ip, apiPort, proxyPort string) {
	c.DataPlaneConnections.Set(ip, &function_metadata.DataPlaneConnectionInfo{
		Iface:     iface,
		IP:        ip,
		APIPort:   apiPort,
		ProxyPort: proxyPort,
	})
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
		if err := c.reconstructEndpointsState(ctx, &c.DataPlaneConnections); err != nil {
			return err
		}
		duration := time.Since(start)
		logrus.Infof("Endpoints reconstruction took : %s", duration)
	}

	return nil
}

func (c *ControlPlane) reconstructDataplaneState(ctx context.Context) error {
	dataplanesValues, err := c.PersistenceLayer.GetDataPlaneInformation(ctx)
	if err != nil {
		return err
	}

	for _, dataplane := range dataplanesValues {
		c.connectToRegisteredDataplane(dataplane)
	}

	return nil
}

func (c *ControlPlane) reconstructWorkersState(ctx context.Context) error {
	workers, err := c.PersistenceLayer.GetWorkerNodeInformation(ctx)
	if err != nil {
		return err
	}

	for _, worker := range workers {
		c.connectToRegisteredWorker(&WorkerNode{
			Name:          worker.Name,
			IP:            worker.Ip,
			Port:          worker.Port,
			CpuCores:      int(worker.CpuCores),
			Memory:        int(worker.Memory),
			LastHeartbeat: time.Now(),
		})
	}

	return nil
}

func (c *ControlPlane) reconstructServiceState(ctx context.Context) error {
	services, err := c.PersistenceLayer.GetServiceInformation(ctx)
	if err != nil {
		return err
	}

	for _, service := range services {
		err := c.connectToRegisteredService(ctx, service)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *ControlPlane) reconstructEndpointsState(ctx context.Context, dpiClients *atomic_map.AtomicMap[string, *function_metadata.DataPlaneConnectionInfo]) error {
	endpoints := make([]*proto.Endpoint, 0)

	for _, workerNode := range c.NIStorage.Values() {
		list, err := workerNode.GetAPI().ListEndpoints(ctx, &emptypb.Empty{})
		if err != nil {
			return err
		}

		endpoints = append(endpoints, list.Endpoint...)
	}

	for _, endpoint := range endpoints {
		controlPlaneEndpoint := &Endpoint{
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

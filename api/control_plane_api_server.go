package api

import (
	"cluster_manager/api/proto"
	"cluster_manager/internal/control_plane"
	"cluster_manager/internal/control_plane/autoscaling"
	"cluster_manager/internal/control_plane/persistence"
	"cluster_manager/internal/data_plane/function_metadata"
	"cluster_manager/pkg/grpc_helpers"
	_map "cluster_manager/pkg/map"
	"cluster_manager/pkg/tracing"
	"cluster_manager/pkg/utils"
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/emptypb"
)

type CpApiServer struct {
	proto.UnimplementedCpiInterfaceServer

	DataPlaneConnections map[string]*function_metadata.DataPlaneConnectionInfo

	NIStorage control_plane.NodeInfoStorage
	SIStorage map[string]*control_plane.ServiceInfoStorage
	sisLock   sync.RWMutex

	WorkerEndpoints     map[string]map[*control_plane.Endpoint]string
	WorkerEndpointsLock *sync.Mutex // TODO: Better synchronization mechanism --> maybe sync.Map

	ColdStartTracing *tracing.TracingService[tracing.ColdStartLogEntry] `json:"-"`
	PlacementPolicy  control_plane.PlacementPolicy
	PersistenceLayer persistence.RedisClient
}

func CreateNewCpApiServer(client persistence.RedisClient, outputFile string, placementPolicy control_plane.PlacementPolicy) *CpApiServer {
	return &CpApiServer{
		NIStorage: control_plane.NodeInfoStorage{
			NodeInfo: make(map[string]*control_plane.WorkerNode),
		},
		SIStorage:            make(map[string]*control_plane.ServiceInfoStorage),
		DataPlaneConnections: make(map[string]*function_metadata.DataPlaneConnectionInfo),
		ColdStartTracing:     tracing.NewColdStartTracingService(outputFile),
		PlacementPolicy:      placementPolicy,
		PersistenceLayer:     client,
		WorkerEndpoints:      make(map[string]map[*control_plane.Endpoint]string),
		WorkerEndpointsLock:  &sync.Mutex{},
	}
}

func (c *CpApiServer) CheckPeriodicallyWorkerNodes() {
	for {
		c.NIStorage.Lock()

		for _, workerNode := range c.NIStorage.NodeInfo {
			if time.Since(workerNode.LastHeartbeat) > 3*utils.HeartbeatInterval {
				// Trigger a node deregistation
				c.deregisterNode(workerNode)
			}
		}
		c.NIStorage.Unlock()

		time.Sleep(5 * time.Second)
	}
}

func (c *CpApiServer) OnMetricsReceive(_ context.Context, metric *proto.AutoscalingMetric) (*proto.ActionStatus, error) {
	storage, ok := c.SIStorage[metric.ServiceName]
	if !ok {
		logrus.Warn("SIStorage does not exist for '", metric.ServiceName, "'")
		return &proto.ActionStatus{Success: false}, nil
	}

	storage.Controller.ScalingMetadata.SetCachedScalingMetric(float64(metric.Metric))
	logrus.Debug("Scaling metric for '", storage.ServiceInfo.Name, "' is ", metric.Metric)

	storage.Controller.Start()

	return &proto.ActionStatus{Success: true}, nil
}

func (c *CpApiServer) ListServices(_ context.Context, _ *emptypb.Empty) (*proto.ServiceList, error) {
	return &proto.ServiceList{Service: _map.Keys(c.SIStorage)}, nil
}

func (c *CpApiServer) RegisterNode(ctx context.Context, in *proto.NodeInfo) (*proto.ActionStatus, error) {
	c.NIStorage.Lock()
	defer c.NIStorage.Unlock()

	_, ok := c.NIStorage.NodeInfo[in.NodeID]
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

	wn := &control_plane.WorkerNode{
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

func (c *CpApiServer) connectToRegisteredWorker(wn *control_plane.WorkerNode) {
	c.NIStorage.NodeInfo[wn.Name] = wn

	c.WorkerEndpointsLock.Lock()
	defer c.WorkerEndpointsLock.Unlock()

	c.WorkerEndpoints[wn.Name] = make(map[*control_plane.Endpoint]string)

	go wn.GetAPI()
}

func (c *CpApiServer) DeregisterNode(ctx context.Context, in *proto.NodeInfo) (*proto.ActionStatus, error) {
	c.NIStorage.Lock()
	defer c.NIStorage.Unlock()

	_, ok := c.NIStorage.NodeInfo[in.NodeID]
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

	wn := &control_plane.WorkerNode{
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

func (c *CpApiServer) deregisterNode(wn *control_plane.WorkerNode) error {
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

	err = c.disconnectRegisteredWorker(wn)
	if err != nil {
		return err
	}

	return nil
}

func (c *CpApiServer) disconnectRegisteredWorker(wn *control_plane.WorkerNode) error {
	delete(c.NIStorage.NodeInfo, wn.Name)

	for endpoint, serviceName := range c.WorkerEndpoints[wn.Name] {
		err := c.PersistenceLayer.DeleteEndpoint(context.Background(), serviceName, endpoint.Node.Name)
		if err != nil {
			return err
		}

		err = c.SIStorage[serviceName].RemoveEndpoint(endpoint, c.DataPlaneConnections)
		if err != nil {
			return err
		}
	}

	c.WorkerEndpointsLock.Lock()
	defer c.WorkerEndpointsLock.Unlock()

	delete(c.WorkerEndpoints, wn.Name)

	return nil
}

func (c *CpApiServer) NodeHeartbeat(_ context.Context, in *proto.NodeHeartbeatMessage) (*proto.ActionStatus, error) {
	c.NIStorage.Lock()
	defer c.NIStorage.Unlock()

	n, ok := c.NIStorage.NodeInfo[in.NodeID]
	if !ok {
		logrus.Debug("Received a heartbeat for non-registered node")

		return &proto.ActionStatus{Success: false}, nil
	}

	c.updateWorkerNodeInformation(n, in)

	logrus.Debugf("Heartbeat received from %s with %d percent cpu usage and %d percent memory usage", in.NodeID, in.CpuUsage, in.MemoryUsage)

	return &proto.ActionStatus{Success: true}, nil
}

func (c *CpApiServer) updateWorkerNodeInformation(workerNode *control_plane.WorkerNode, in *proto.NodeHeartbeatMessage) {
	workerNode.LastHeartbeat = time.Now()
	workerNode.CpuUsage = int(in.CpuUsage)
	workerNode.MemoryUsage = int(in.MemoryUsage)
}

func (c *CpApiServer) RegisterService(ctx context.Context, serviceInfo *proto.ServiceInfo) (*proto.ActionStatus, error) {
	c.sisLock.RLock() // TODO: François maybe set a readlock here
	_, ok := c.SIStorage[serviceInfo.Name]
	if ok {
		c.sisLock.RUnlock()

		logrus.Errorf("Service with name %s is already registered", serviceInfo.Name)
		return &proto.ActionStatus{Success: false}, errors.New("service is already registered")
	}
	c.sisLock.RUnlock()

	err := c.PersistenceLayer.StoreServiceInformation(ctx, serviceInfo)
	if err != nil {
		logrus.Errorf("Failed to store information to persistence layer (error : %s)", err.Error())
		return &proto.ActionStatus{Success: false}, err
	}

	for _, conn := range c.DataPlaneConnections {
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

func (c *CpApiServer) connectToRegisteredService(ctx context.Context, serviceInfo *proto.ServiceInfo) error {
	scalingChannel := make(chan int)

	service := &control_plane.ServiceInfoStorage{
		ServiceInfo: serviceInfo,
		Controller: &control_plane.PFStateController{
			DesiredStateChannel: &scalingChannel,
			Period:              2 * time.Second, // TODO: hardcoded autoscaling period for now
			ScalingMetadata: autoscaling.AutoscalingMetadata{
				AutoscalingConfig: serviceInfo.AutoscalingConfig,
			},
		},
		ColdStartTracingChannel: &c.ColdStartTracing.InputChannel,
		PlacementPolicy:         c.PlacementPolicy,
		PersistenceLayer:        c.PersistenceLayer,
		WorkerEndpoints:         c.WorkerEndpoints,
		WorkerEndpointsLock:     c.WorkerEndpointsLock,
	}

	c.sisLock.Lock()
	c.SIStorage[serviceInfo.Name] = service
	c.sisLock.Unlock()

	go service.ScalingControllerLoop(&c.NIStorage, c.DataPlaneConnections)

	return nil
}

func (c *CpApiServer) RegisterDataplane(ctx context.Context, in *proto.DataplaneInfo) (*proto.ActionStatus, error) {
	logrus.Trace("Revieved a control plane registration")

	ipAddress, ok := grpc_helpers.GetIPAddressFromGRPCCall(ctx)
	if !ok {
		logrus.Debug("Failed to extract IP address from data plane registration request")
		return &proto.ActionStatus{Success: false}, nil
	}

	apiPort := strconv.Itoa(int(in.APIPort))
	proxyPort := strconv.Itoa(int(in.ProxyPort))
	dataplaneInfo := proto.DataplaneInformation{
		Address:   ipAddress,
		ApiPort:   apiPort,
		ProxyPort: proxyPort,
	}

	err := c.PersistenceLayer.StoreDataPlaneInformation(ctx, &dataplaneInfo)
	if err != nil {
		logrus.Errorf("Failed to store information to persistence layer (error : %s)", err.Error())
		return &proto.ActionStatus{Success: false}, err
	}

	c.connectToRegisteredDataplane(&dataplaneInfo)

	return &proto.ActionStatus{Success: true}, nil
}

func (c *CpApiServer) connectToRegisteredDataplane(information *proto.DataplaneInformation) {
	c.registerDataplane(
		grpc_helpers.InitializeDataPlaneConnection(information.Address, information.ApiPort),
		information.Address,
		information.ApiPort,
		information.ProxyPort,
	)
}

func (c *CpApiServer) registerDataplane(iface proto.DpiInterfaceClient, ip, apiPort, proxyPort string) {
	c.DataPlaneConnections[ip] = &function_metadata.DataPlaneConnectionInfo{
		Iface:     iface,
		IP:        ip,
		APIPort:   apiPort,
		ProxyPort: proxyPort,
	}
}

func (c *CpApiServer) DeregisterDataplane(ctx context.Context, in *proto.DataplaneInfo) (*proto.ActionStatus, error) {
	logrus.Trace("Recieved a data plane deregistration")

	ipAddress, ok := grpc_helpers.GetIPAddressFromGRPCCall(ctx)
	if !ok {
		logrus.Debug("Failed to extract IP address from data plane registration request")
		return &proto.ActionStatus{Success: false}, nil
	}

	apiPort := strconv.Itoa(int(in.APIPort))
	proxyPort := strconv.Itoa(int(in.ProxyPort))
	dataplaneInfo := proto.DataplaneInformation{
		Address:   ipAddress,
		ApiPort:   apiPort,
		ProxyPort: proxyPort,
	}

	err := c.PersistenceLayer.DeleteDataPlaneInformation(ctx, &dataplaneInfo)
	if err != nil {
		logrus.Errorf("Failed to store information to persistence layer (error : %s)", err.Error())
		return &proto.ActionStatus{Success: false}, err
	}

	c.deregisterDataplane(&dataplaneInfo)

	return &proto.ActionStatus{Success: true}, nil
}

func (c *CpApiServer) deregisterDataplane(information *proto.DataplaneInformation) {
	delete(c.DataPlaneConnections, information.Address)
}

func (c *CpApiServer) ReconstructState(ctx context.Context) error {
	go c.reconstructDataplaneState(ctx)

	err := c.reconstructWorkersState(ctx)
	if err != nil {
		return err
	}

	// TODO: Fix issue francois costa
	//err = c.reconstructServiceState(ctx)
	//if err != nil {
	//return err
	//}

	return nil

	return c.reconstructEndpointsState(ctx)
}

func (c *CpApiServer) reconstructDataplaneState(ctx context.Context) error {
	dataplanesValues, err := c.PersistenceLayer.GetDataPlaneInformation(ctx)
	if err != nil {
		return err
	}

	for _, dataplane := range dataplanesValues {
		c.connectToRegisteredDataplane(dataplane)
	}

	return nil
}

func (c *CpApiServer) reconstructWorkersState(ctx context.Context) error {
	workers, err := c.PersistenceLayer.GetWorkerNodeInformation(ctx)
	if err != nil {
		return err
	}

	for _, worker := range workers {
		c.connectToRegisteredWorker(&control_plane.WorkerNode{
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

func (c *CpApiServer) reconstructServiceState(ctx context.Context) error {
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

func (c *CpApiServer) reconstructEndpointsState(ctx context.Context) error {
	endpoints, services, err := c.PersistenceLayer.GetEndpoints(ctx)
	if err != nil {
		return err
	}

	for i, endpoint := range endpoints {
		controlPlaneEndpoint := &control_plane.Endpoint{
			SandboxID: endpoint.SandboxID,
			URL:       endpoint.URL,
			Node:      c.NIStorage.NodeInfo[endpoint.NodeName],
			HostPort:  endpoint.HostPort,
		}
		c.SIStorage[services[i]].ReconstructEndpointsFromDatabase(controlPlaneEndpoint)
	}

	return nil
}

func (c *CpApiServer) SerializeCpApiServer(ctx context.Context) {
	serialized, err := json.Marshal(*c)
	if err != nil {
		logrus.Errorf("Failed to serialize control plane on failure : %s", err.Error())
		return
	}

	err = c.PersistenceLayer.StoreSerialized(ctx, serialized)
	if err != nil {
		logrus.Errorf("Failed to save control plane on failure : %s", err.Error())
		return
	}

	logrus.Info("Stored the control plane in the persistence layer with success")
}

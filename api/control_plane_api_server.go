package api

import (
	"cluster_manager/api/proto"
	"cluster_manager/internal/common"
	"cluster_manager/internal/control_plane"
	_map "cluster_manager/pkg/map"
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/emptypb"
)

type CpApiServer struct {
	proto.UnimplementedCpiInterfaceServer

	DataPlaneConnections []*common.DataPlaneConnectionInfo

	NIStorage control_plane.NodeInfoStorage
	SIStorage map[string]*control_plane.ServiceInfoStorage

	ColdStartTracing *common.TracingService[common.ColdStartLogEntry]
	PlacementPolicy  control_plane.PlacementPolicy
	PersistenceLayer control_plane.RedisClient
}

func CreateNewCpApiServer(client control_plane.RedisClient, outputFile string, placementPolicy control_plane.PlacementPolicy) *CpApiServer {
	return &CpApiServer{
		NIStorage: control_plane.NodeInfoStorage{
			NodeInfo: make(map[string]*control_plane.WorkerNode),
		},
		SIStorage:        make(map[string]*control_plane.ServiceInfoStorage),
		ColdStartTracing: common.NewColdStartTracingService(outputFile),
		PlacementPolicy:  placementPolicy,
		PersistenceLayer: client,
	}
}

func (c *CpApiServer) GetDpiConnections() []*common.DataPlaneConnectionInfo {
	return c.DataPlaneConnections
}

func (c *CpApiServer) appendDpiConnection(iface proto.DpiInterfaceClient, ip, apiPort, proxyPort string) {
	conn := &common.DataPlaneConnectionInfo{
		Iface:     iface,
		IP:        ip,
		APIPort:   apiPort,
		ProxyPort: proxyPort,
	}

	c.DataPlaneConnections = append(c.DataPlaneConnections, conn)
}

func (c *CpApiServer) OnMetricsReceive(_ context.Context, metric *proto.AutoscalingMetric) (*proto.ActionStatus, error) {
	service, ok := c.SIStorage[metric.ServiceName]
	if !ok {
		logrus.Warn("Autoscaling controller does not exist for function ", metric.ServiceName)
		return &proto.ActionStatus{Success: false}, nil
	}

	storage, ok := c.SIStorage[metric.ServiceName]
	if !ok {
		logrus.Warn("SIStorage does not exist for '", metric.ServiceName, "'")
		return &proto.ActionStatus{Success: false}, nil
	}

	storage.Controller.ScalingMetadata.SetCachedScalingMetric(float64(metric.Metric))
	logrus.Debug("Scaling metric for '", service.ServiceInfo.Name, "' is ", metric.Metric)

	service.Controller.Start()

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

	ipAddress, ok := common.GetIPAddressFromGRPCCall(ctx)
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

	err := c.PersistenceLayer.StoreWorkerNodeInformation(ctx, fmt.Sprintf("worker:%s", wn.Name), control_plane.WorkerNodeInformation{
		Name:     wn.Name,
		Ip:       wn.IP,
		Port:     wn.Port,
		CpuCores: strconv.Itoa(wn.CpuCores),
		Memory:   strconv.Itoa(wn.Memory),
	})
	if err != nil {
		logrus.Error("Failed to store information to persistence layer")
		return &proto.ActionStatus{Success: false}, err
	}

	c.NIStorage.NodeInfo[in.NodeID] = wn
	go wn.GetAPI()

	logrus.Info("Node '", in.NodeID, "' has been successfully register with the control plane")

	return &proto.ActionStatus{Success: true}, nil
}

func (c *CpApiServer) NodeHeartbeat(_ context.Context, in *proto.NodeHeartbeatMessage) (*proto.ActionStatus, error) {
	c.NIStorage.Lock()
	defer c.NIStorage.Unlock()

	n, ok := c.NIStorage.NodeInfo[in.NodeID]
	if !ok {
		logrus.Debug("Received a heartbeat for non-registered node")

		return &proto.ActionStatus{Success: false}, nil
	}

	updateWorkerNode(n, in)

	logrus.Debugf("Heartbeat received from %s with %d percent cpu usage and %d percent memory usage", in.NodeID, in.CpuUsage, in.MemoryUsage)

	return &proto.ActionStatus{Success: true}, nil
}

func updateWorkerNode(workerNode *control_plane.WorkerNode, in *proto.NodeHeartbeatMessage) {
	workerNode.LastHeartbeat = time.Now()
	workerNode.CpuUsage = int(in.CpuUsage)
	workerNode.MemoryUsage = int(in.MemoryUsage)
}

func (c *CpApiServer) RegisterService(ctx context.Context, serviceInfo *proto.ServiceInfo) (*proto.ActionStatus, error) {
	for _, conn := range c.DataPlaneConnections {
		resp, err := conn.Iface.AddDeployment(ctx, serviceInfo)
		if err != nil || !resp.Success {
			logrus.Warn("Failed to propagate service registration to the data plane")
			return &proto.ActionStatus{Success: false}, nil
		}
	}

	scalingChannel := make(chan int)

	service := &control_plane.ServiceInfoStorage{
		ServiceInfo: serviceInfo,
		Controller: &control_plane.PFStateController{
			DesiredStateChannel: &scalingChannel,
			NotifyChannel:       &scalingChannel,
			Period:              2 * time.Second, // TODO: hardcoded autoscaling period for now
			ScalingMetadata: control_plane.AutoscalingMetadata{
				AutoscalingConfig: serviceInfo.AutoscalingConfig,
			},
		},
		ColdStartTracingChannel: &c.ColdStartTracing.InputChannel,
		PlacementPolicy:         c.PlacementPolicy,
		PertistenceLayer:        c.PersistenceLayer,
	}

	err := c.PersistenceLayer.StoreServiceInformation(ctx, fmt.Sprintf("service:%s", serviceInfo.Name), serviceInfo)
	if err != nil {
		logrus.Error("Failed to store information to persistence layer")
		return &proto.ActionStatus{Success: false}, err
	}

	c.SIStorage[serviceInfo.Name] = service

	go service.ScalingControllerLoop(&c.NIStorage, c.GetDpiConnections())

	return &proto.ActionStatus{Success: true}, nil
}

func (c *CpApiServer) RegisterDataplane(ctx context.Context, in *proto.DataplaneInfo) (*proto.ActionStatus, error) {
	ipAddress, ok := common.GetIPAddressFromGRPCCall(ctx)
	apiPort := strconv.Itoa(int(in.APIPort))
	proxyPort := strconv.Itoa(int(in.ProxyPort))

	if !ok {
		logrus.Debug("Failed to extract IP address from data plane registration request")
		return &proto.ActionStatus{Success: false}, nil
	}

	err := c.PersistenceLayer.StoreDataPlaneInformation(ctx, fmt.Sprintf("dataplane:%s", ipAddress), control_plane.DataPlaneInformation{
		Address:   ipAddress,
		ApiPort:   apiPort,
		ProxyPort: proxyPort,
	})
	if err != nil {
		logrus.Error("Failed to store information to persistence layer")
		return &proto.ActionStatus{Success: false}, err
	}

	c.appendDpiConnection(common.InitializeDataPlaneConnection(ipAddress, apiPort), ipAddress, apiPort, proxyPort)

	return &proto.ActionStatus{Success: true}, nil
}

package api

import (
	"cluster_manager/api/proto"
	"cluster_manager/internal/algorithms/placement"
	common "cluster_manager/internal/common"
	"cluster_manager/internal/control_plane"
	"context"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/emptypb"
)

type CpApiServer struct {
	proto.UnimplementedCpiInterfaceServer

	dpiInterface proto.DpiInterfaceClient
	dpiIP        string
	dpiAPIPort   string
	dpiProxyPort string

	NIStorage control_plane.NodeInfoStorage
	SIStorage map[string]*control_plane.ServiceInfoStorage

	ColdStartTracing *common.TracingService[common.ColdStartLogEntry]
}

func CreateNewCpApiServer(outputFile string) *CpApiServer {
	return &CpApiServer{
		NIStorage: control_plane.NodeInfoStorage{
			NodeInfo: make(map[string]*control_plane.WorkerNode),
		},
		SIStorage:        make(map[string]*control_plane.ServiceInfoStorage),
		ColdStartTracing: common.NewColdStartTracingService(outputFile),
	}
}

func (c *CpApiServer) DpiInterface() proto.DpiInterfaceClient {
	if c.dpiInterface == nil {
		logrus.Fatal("Connection with the data plane has not been established.")
	}

	return c.dpiInterface
}

func (c *CpApiServer) setDpiInterface(iface proto.DpiInterfaceClient, ip, apiPort, proxyPort string) {
	c.dpiInterface = iface

	c.dpiIP = ip
	c.dpiAPIPort = apiPort
	c.dpiProxyPort = proxyPort
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
	return &proto.ServiceList{Service: common.Keys(c.SIStorage)}, nil
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
	resp, err := c.DpiInterface().AddDeployment(ctx, serviceInfo)
	if err != nil || !resp.Success {
		logrus.Warn("Failed to propagate service registration to the data plane")
		return &proto.ActionStatus{Success: false}, nil
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
		PlacementPolicy:         placement.KUBERNETES,
	}
	c.SIStorage[serviceInfo.Name] = service

	go service.ScalingControllerLoop(&c.NIStorage, c.DpiInterface())

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

	c.setDpiInterface(common.InitializeDataPlaneConnection(ipAddress, apiPort), ipAddress, apiPort, proxyPort)

	return &proto.ActionStatus{Success: true}, nil
}

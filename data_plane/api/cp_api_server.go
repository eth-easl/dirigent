package api

import (
	"cluster_manager/api/proto"
	"cluster_manager/common"
	"context"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/emptypb"
	"strconv"
	"time"
)

type CpApiServer struct {
	proto.UnimplementedCpiInterfaceServer

	DpiInterface proto.DpiInterfaceClient
	NIStorage    NodeInfoStorage
	SIStorage    map[string]*ServiceInfoStorage
}

func CreateNewCpApiServer() *CpApiServer {
	return &CpApiServer{
		NIStorage: NodeInfoStorage{
			NodeInfo: make(map[string]*WorkerNode),
		},
		SIStorage: make(map[string]*ServiceInfoStorage),
	}
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

	storage.Controller.ScalingMetadata.setCachedScalingMetric(float64(metric.Metric))
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

	wn := &WorkerNode{
		Name: in.NodeID,
		IP:   ipAddress,
		Port: strconv.Itoa(int(in.Port)),
	}

	c.NIStorage.NodeInfo[in.NodeID] = wn
	go wn.GetAPI()

	logrus.Info("Node '", in.NodeID, "' has been successfully register with the control plane")
	return &proto.ActionStatus{Success: true}, nil
}

func (c *CpApiServer) NodeHeartbeat(_ context.Context, in *proto.NodeInfo) (*proto.ActionStatus, error) {
	c.NIStorage.Lock()
	defer c.NIStorage.Unlock()

	n, ok := c.NIStorage.NodeInfo[in.NodeID]
	if !ok {
		logrus.Debug("Received a heartbeat for non-registered node")

		return &proto.ActionStatus{Success: false}, nil
	}

	n.LastHeartbeat = time.Now()

	//logrus.Debug("Heartbeat received for '", in.NodeID, "'")
	return &proto.ActionStatus{Success: true}, nil
}

func (c *CpApiServer) RegisterService(ctx context.Context, serviceInfo *proto.ServiceInfo) (*proto.ActionStatus, error) {
	resp, err := c.DpiInterface.AddDeployment(ctx, serviceInfo)
	if err != nil || !resp.Success {
		logrus.Warn("Failed to propagate service registration to the data plane")
		return &proto.ActionStatus{Success: false}, nil
	}

	scalingChannel := make(chan int)

	service := &ServiceInfoStorage{
		ServiceInfo: serviceInfo,
		Controller: &PFStateController{
			DesiredStateChannel: &scalingChannel,
			NotifyChannel:       &scalingChannel,
			Period:              2 * time.Second, // TODO: hardcoded autoscaling period for now
			ScalingMetadata:     ConvertProtoToAutoscalingStruct(serviceInfo.AutoscalingConfig),
		},
	}
	c.SIStorage[serviceInfo.Name] = service
	go service.ScalingControllerLoop(&c.NIStorage, c.DpiInterface)

	return &proto.ActionStatus{Success: true}, nil
}

func (c *CpApiServer) RegisterDataplane(ctx context.Context, in *proto.DataplaneInfo) (*proto.ActionStatus, error) {
	ipAddress, ok := common.GetIPAddressFromGRPCCall(ctx)
	if !ok {
		logrus.Debug("Failed to extract IP address from data plane registration request")
		return &proto.ActionStatus{Success: false}, nil
	}

	c.DpiInterface = common.InitializeDataPlaneConnection(ipAddress, strconv.Itoa(int(in.Port)))
	return &proto.ActionStatus{Success: true}, nil
}

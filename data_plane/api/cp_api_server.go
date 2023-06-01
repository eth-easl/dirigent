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

func CreateNewCpApiServer(dpiInterface proto.DpiInterfaceClient) *CpApiServer {
	return &CpApiServer{
		DpiInterface: dpiInterface,
		NIStorage: NodeInfoStorage{
			NodeInfo: make(map[string]*WorkerNode),
		},
		SIStorage: make(map[string]*ServiceInfoStorage),
	}
}

func (c *CpApiServer) ScaleFromZero(_ context.Context, info *proto.ServiceInfo) (*proto.ActionStatus, error) {
	service, ok := c.SIStorage[info.Name]
	if !ok {
		logrus.Warn("Autoscaling controller does not exist for function ", info.Name)
		return &proto.ActionStatus{Success: false}, nil
	}

	service.Scaling.Start()

	return &proto.ActionStatus{Success: true}, nil
}

func (c *CpApiServer) ListServices(_ context.Context, _ *emptypb.Empty) (*proto.ServiceList, error) {
	return &proto.ServiceList{Service: common.Keys(c.SIStorage)}, nil
}

func (c *CpApiServer) RegisterNode(_ context.Context, in *proto.NodeInfo) (*proto.ActionStatus, error) {
	c.NIStorage.Lock()
	defer c.NIStorage.Unlock()

	_, ok := c.NIStorage.NodeInfo[in.NodeID]
	if ok {
		return &proto.ActionStatus{
			Success: false,
			Message: "Node registration failed. Node with the same name already exists.",
		}, nil
	}

	wn := &WorkerNode{
		Name: in.NodeID,
		IP:   in.IP,
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

	logrus.Debug("Heartbeat received for '", in.NodeID, "'")
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
		Scaling: &Autoscaler{
			NotifyChannel: &scalingChannel,
			Period:        2 * time.Second, // TODO: hardcoded for now
		},
		Controller: &PFStateController{
			DesiredStateChannel: &scalingChannel,
		},
	}
	c.SIStorage[serviceInfo.Name] = service
	go service.ScalingControllerLoop(&c.NIStorage, c.DpiInterface)

	return &proto.ActionStatus{Success: true}, nil
}

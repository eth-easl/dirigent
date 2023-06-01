package api

import (
	"cluster_manager/api/proto"
	"cluster_manager/common"
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/emptypb"
	"strconv"
	"sync"
	"time"
)

func InitializeControlPlaneConnection() proto.CpiInterfaceClient {
	conn := common.EstablishGRPCConnectionPoll(common.ControlPlaneHost, common.ControlPlanePort)
	if conn == nil {
		logrus.Fatal("Failed to establish connection with the control plane")
	}

	logrus.Info("Successfully established connection with the control plane")

	return proto.NewCpiInterfaceClient(conn)
}

func InitializeWorkerNodeConnection(host, port string) proto.WorkerNodeInterfaceClient {
	conn := common.EstablishGRPCConnectionPoll(host, port)
	if conn == nil {
		logrus.Fatal("Failed to establish connection with the worker node")
	}

	logrus.Info("Successfully established connection with the worker node")

	return proto.NewWorkerNodeInterfaceClient(conn)
}

type NodeInfoStorage struct {
	sync.Mutex

	NodeInfo map[string]*WorkerNode
}

type ServiceInfoStorage struct {
	ServiceInfo *proto.ServiceInfo
	Scaling     *Autoscaler

	Controller *PFStateController
	endpoints  []string
}

func (ss *ServiceInfoStorage) ScalingControllerLoop(nodeList *NodeInfoStorage, dpiClient proto.DpiInterfaceClient) {
	// TODO: add locking
	for {
		select {
		case desiredCount := <-*ss.Controller.DesiredStateChannel:
			if ss.Controller.ActualScale < desiredCount {
				diff := desiredCount - ss.Controller.ActualScale

				for i := 0; i < diff; i++ {
					node := placementPolicy(nodeList)
					resp, err := node.GetAPI().CreateSandbox(context.Background(), ss.ServiceInfo)
					if err != nil || !resp.Success {
						logrus.Warn("Failed to start a sandbox on worker node ", node.Name)
					}

					ss.Controller.ActualScale++
					nodePort := resp.Message // TODO: temp only
					ss.endpoints = append(ss.endpoints, fmt.Sprintf("localhost:%s", nodePort))

					if resp.Success {
						resp, err := dpiClient.UpdateEndpointList(context.Background(), &proto.DeploymentEndpointPatch{
							Service:   ss.ServiceInfo,
							Endpoints: ss.endpoints,
						})
						if err != nil || !resp.Success {
							logrus.Warn("Failed to update endpoint list in the data plane")
						}
					}
				}
			} else if ss.Controller.ActualScale > desiredCount {
				// TODO: implement downscaling
			}
		}
	}
}

type WorkerNode struct {
	Name string
	IP   string
	Port string

	LastHeartbeat time.Time
	api           proto.WorkerNodeInterfaceClient
}

func (w *WorkerNode) GetAPI() proto.WorkerNodeInterfaceClient {
	if w.api == nil {
		// TODO: remove hardcoded IP address
		w.api = InitializeWorkerNodeConnection("localhost", w.Port)
	}

	return w.api
}

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
			Period:        2 * time.Second,
		},
		Controller: &PFStateController{
			DesiredStateChannel: &scalingChannel,
		},
	}
	c.SIStorage[serviceInfo.Name] = service
	go service.ScalingControllerLoop(&c.NIStorage, c.DpiInterface)

	return &proto.ActionStatus{Success: true}, nil
}

package api

import (
	"cluster_manager/api/proto"
	"cluster_manager/common"
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"k8s.io/apimachinery/pkg/util/wait"
	"net"
	"strconv"
	"sync"
	"time"
)

func InitializeControlPlaneConnection() proto.CpiInterfaceClient {
	var conn *grpc.ClientConn

	pollContext, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pollErr := wait.PollUntilContextCancel(pollContext, 5*time.Second, false,
		func(ctx context.Context) (done bool, err error) {
			c, err := common.EstablishConnection(
				ctx,
				net.JoinHostPort(common.ControlPlaneHost, common.ControlPlanePort),
				common.GetLongLivingConnectionDialOptions()...,
			)
			if err != nil {
				logrus.Warn("Retrying to connect to the control plane in 5 seconds")
			}

			conn = c
			return c != nil, nil
		},
	)

	if pollErr != nil {
		logrus.Fatal("Failed to establish connection with the data plane")
	}

	logrus.Info("Successfully established connection with the control plane")

	return proto.NewCpiInterfaceClient(conn)
}

func InitializeWorkerNodeConnection(host, port string) proto.WorkerNodeInterfaceClient {
	var conn *grpc.ClientConn

	pollContext, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pollErr := wait.PollUntilContextCancel(pollContext, 5*time.Second, false,
		func(ctx context.Context) (done bool, err error) {
			c, err := common.EstablishConnection(
				ctx,
				net.JoinHostPort(host, port),
				common.GetLongLivingConnectionDialOptions()...,
			)
			if err != nil {
				logrus.Warn("Retrying to connect to the worker node in 5 seconds")
			}

			conn = c
			return c != nil, nil
		},
	)

	if pollErr != nil {
		logrus.Fatal("Failed to establish connection with the worker node")
	}

	logrus.Info("Successfully established connection with the worker node")

	return proto.NewWorkerNodeInterfaceClient(conn)
}

type NodeInfoStorage struct {
	sync.Mutex

	NodeInfo map[string]*WorkerNode
}

type Autoscaler struct {
	sync.Mutex

	Running       bool
	StopChannel   chan struct{}
	NotifyChannel chan struct{}
}

/*func (a *Autoscaler) Start() {
	a.Lock()
	defer a.Unlock()

	if !a.Running {
		go a.ScalingLoop()
	}
}*/

func ScalingLoop(info *proto.ServiceInfo, api proto.WorkerNodeInterfaceClient) string {
	resp, err := api.CreateSandbox(context.Background(), info)
	if err != nil || !resp.Success {
		logrus.Warn("Failed to upscale.")
	}

	return resp.Message
}

type ServiceInfoStorage struct {
	ServiceInfo map[string]*proto.ServiceInfo
	Scaling     map[string]*Autoscaler
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
		// TODO: unhardcode IP address
		w.api = InitializeWorkerNodeConnection("localhost", w.Port)
	}

	return w.api
}

type CpApiServer struct {
	proto.UnimplementedCpiInterfaceServer

	DpiInterface proto.DpiInterfaceClient
	NIStorage    NodeInfoStorage
	SIStorage    ServiceInfoStorage
}

func (c *CpApiServer) ScaleFromZero(ctx context.Context, in *proto.ServiceInfo) (*proto.ActionStatus, error) {
	// TODO: needs locking
	/*autoscaler, ok := c.SIStorage.Scaling[serviceName]
	if !ok {
		logrus.Warn("Could not find an autoscaler for the requested service type.")
		return &proto.ActionStatus{Success: false}, nil
	}*/

	node := c.NIStorage.NodeInfo["node-0"]
	serviceInfo := c.SIStorage.ServiceInfo[in.Name]
	ip := ScalingLoop(serviceInfo, node.GetAPI())

	resp, err := c.DpiInterface.UpdateEndpointList(ctx, &proto.DeploymentEndpointPatch{
		Service: &proto.ServiceInfo{
			Name: in.Name,
		},
		Endpoints: []string{
			fmt.Sprintf("%s:%s", node.IP, ip),
		},
	})

	return &proto.ActionStatus{
		Success: resp.Success,
		Message: resp.Message,
	}, err
}

func (c *CpApiServer) ListServices(_ context.Context, _ *emptypb.Empty) (*proto.ServiceList, error) {
	return &proto.ServiceList{Service: []string{"/faas.Executor/Execute"}}, nil
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

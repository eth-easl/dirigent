package main

import (
	"cluster_manager/api"
	"cluster_manager/api/proto"
	"cluster_manager/common"
	"cluster_manager/sandbox"
	"context"
	"flag"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"k8s.io/apimachinery/pkg/util/wait"
	"math/rand"
	"strconv"
	"time"
)

var (
	nodeName  = flag.String("nodeName", "node-0", "Node name to register with the control plane")
	ipAddress = flag.String("ipAddress", "0.0.0.0", "IP address to create worker node gRPC server on")
	port      = flag.Int("port", common.WorkerNodePort, "Port worker node gRPC server listens on")
	verbosity = flag.String("verbosity", "info", "Logging verbosity - choose from [info, debug, trace]")
)

func main() {
	flag.Parse()
	rand.Seed(time.Now().UnixNano())

	stopChannel := make(chan struct{})

	logrus.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
	switch *verbosity {
	case "debug":
		logrus.SetLevel(logrus.DebugLevel)
	case "trace":
		logrus.SetLevel(logrus.TraceLevel)
	default:
		logrus.SetLevel(logrus.InfoLevel)
	}

	cli := sandbox.GetDockerClient()
	defer cli.Close()

	cpApi := common.InitializeControlPlaneConnection()

	registerNodeWithControlPlane(&cpApi)
	go setupHeartbeatLoop(&cpApi)

	logrus.Info("Starting API handlers")
	go common.CreateGRPCServer(*ipAddress, strconv.Itoa(common.WorkerNodePort), func(sr grpc.ServiceRegistrar) {
		proto.RegisterWorkerNodeInterfaceServer(sr, &api.WnApiServer{
			DockerClient: cli,
		})
	})

	<-stopChannel
}

func registerNodeWithControlPlane(cpApi *proto.CpiInterfaceClient) {
	pollContext, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pollErr := wait.PollUntilContextCancel(pollContext, 5*time.Second, false,
		func(ctx context.Context) (done bool, err error) {
			resp, err := (*cpApi).RegisterNode(context.Background(), &proto.NodeInfo{
				NodeID: *nodeName,
				IP:     *ipAddress,
				Port:   int32(*port),
			})

			if err != nil || resp == nil {
				logrus.Warn("Retrying to register the node with the control plane in 5 seconds")
				return false, nil
			}

			return resp.Success, nil
		},
	)
	if pollErr != nil {
		logrus.Fatal("Failed to register the node with the control plane")
	}

	logrus.Info("Successfully registered the node with the control plane")
}

func setupHeartbeatLoop(cpApi *proto.CpiInterfaceClient) {
	HeartbeatInterval := 10 * time.Second

	nodeInfo := &proto.NodeInfo{
		NodeID: *nodeName,
		IP:     *ipAddress,
		Port:   int32(*port),
	}

	for {
		resp, err := (*cpApi).NodeHeartbeat(context.Background(), nodeInfo)
		if err != nil || resp == nil || !resp.Success {
			logrus.Warn("Failed to send a heartbeat to the control plane")
		} else {
			logrus.Debug("Successfully sent a heartbeat")
		}

		time.Sleep(HeartbeatInterval)
	}
}

package main

import (
	"cluster_manager/api"
	"cluster_manager/api/proto"
	"cluster_manager/common"
	"cluster_manager/sandbox"
	"context"
	"flag"
	"fmt"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"k8s.io/apimachinery/pkg/util/wait"
	"os"
	"strconv"
	"time"
)

var (
	controlPlaneIP   = flag.String("controlPlaneIP", "localhost", "Control plane IP address")
	controlPlanePort = flag.String("controlPlanePort", common.DefaultControlPlanePort, "Control plane port")
	port             = flag.Int("port", common.DefaultWorkerNodePort, "Worker daemon incoming traffic port")
	verbosity        = flag.String("verbosity", "trace", "Logging verbosity - choose from [info, debug, trace]")

	criPath       = flag.String("criPath", "/run/containerd/containerd.sock", "Path to containerd socket")
	cniConfigPath = flag.String("cniConfigPath", "/home/francois/Documents/cluster_manager/configs/cni.conf", "Path to CNI config")
)

func main() {
	flag.Parse()
	common.InitLibraries(*verbosity)

	stopChannel := make(chan struct{})

	containerdClient := sandbox.GetContainerdClient(*criPath)
	defer containerdClient.Close()

	cniClient := sandbox.GetCNIClient(*cniConfigPath)
	ipt, err := sandbox.NewIptablesUtil()
	if err != nil {
		logrus.Fatal("Error while accessing iptables - ", err)
	}

	cpApi := common.InitializeControlPlaneConnection(*controlPlaneIP, *controlPlanePort, -1, -1)

	registerNodeWithControlPlane(&cpApi)
	go setupHeartbeatLoop(&cpApi)

	logrus.Info("Starting API handlers")
	go common.CreateGRPCServer("0.0.0.0", strconv.Itoa(common.DefaultWorkerNodePort), func(sr grpc.ServiceRegistrar) {
		proto.RegisterWorkerNodeInterfaceServer(sr, &api.WnApiServer{
			ContainerdClient: containerdClient,
			CNIClient:        cniClient,
			IPT:              ipt,

			ImageManager:   sandbox.NewImageManager(),
			SandboxManager: sandbox.NewSandboxManager(),
		})
	})

	<-stopChannel
}

func registerNodeWithControlPlane(cpApi *proto.CpiInterfaceClient) {
	hostName, err := os.Hostname()
	if err != nil {
		logrus.Warn("Error fetching host name.")
	}

	pollContext, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pollErr := wait.PollUntilContextCancel(pollContext, 5*time.Second, false,
		func(ctx context.Context) (done bool, err error) {
			resp, err := (*cpApi).RegisterNode(context.Background(), &proto.NodeInfo{
				NodeID: hostName,
				// IP fetched from server-side context
				Port: int32(*port),
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
	const HeartbeatInterval = 10 * time.Second

	for {
		// Send
		sendHeartbeatLoop(cpApi)
		// Wait
		time.Sleep(HeartbeatInterval)
	}
}

func getWorkerStatistics() (*proto.NodeHeartbeatMessage, error) {
	harwareUsage := common.GetHardwareUsage()

	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}

	return &proto.NodeHeartbeatMessage{
		NodeID:      hostname,
		CpuUsage:    harwareUsage.CpuUsage,
		MemoryUsage: harwareUsage.MemoryUsage,
	}, nil
}

func sendHeartbeatLoop(cpApi *proto.CpiInterfaceClient) {

	pollContext, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pollErr := wait.PollUntilContextCancel(pollContext, 5*time.Second, false,
		func(ctx context.Context) (done bool, err error) {
			workerStatistics, err := getWorkerStatistics()
			if err != nil {
				return false, err
			}

			resp, err := (*cpApi).NodeHeartbeat(ctx, workerStatistics)

			// In case we don't manage to connect, we give up
			if err != nil || resp == nil || !resp.Success {
				return false, nil
			}

			return true, nil
		},
	)
	if pollErr != nil {
		logrus.Warn(fmt.Sprintf("Failed to send a heartbeat to the control plane : %s", pollErr))
	} else {
		logrus.Debug("Sent heartbeat to the control plane")
	}

}

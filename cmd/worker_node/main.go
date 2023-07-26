package main

import (
	"cluster_manager/api"
	"cluster_manager/api/proto"
	"cluster_manager/internal/common"
	"cluster_manager/internal/sandbox"
	config2 "cluster_manager/pkg/config"
	"cluster_manager/pkg/hardware"
	"cluster_manager/pkg/logger"
	"cluster_manager/pkg/utils"
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"k8s.io/apimachinery/pkg/util/wait"
)

func main() {
	config, err := config2.ReadWorkedNodeConfiguration("cmd/worker_node/config.yaml")
	if err != nil {
		logrus.Fatal("Failed to read configuration file (error : %s)", err.Error())
	}

	logger.SetupLogger(config.Verbosity)

	stopChannel := make(chan struct{})

	containerdClient := sandbox.GetContainerdClient(config.CRIPath)
	defer containerdClient.Close()

	cniClient := sandbox.GetCNIClient(config.CNIConfigPath)
	ipt, err := sandbox.NewIptablesUtil()

	if err != nil {
		logrus.Fatal("Error while accessing iptables - ", err)
	}

	cpApi := common.InitializeControlPlaneConnection(config.ControlPlaneIp, config.ControlPlanePort, -1, -1)

	registerNodeWithControlPlane(config, &cpApi)

	go setupHeartbeatLoop(&cpApi)

	logrus.Info("Starting API handlers")

	go common.CreateGRPCServer(utils.DockerLocalhost, strconv.Itoa(config.Port), func(sr grpc.ServiceRegistrar) {
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

func registerNodeWithControlPlane(config config2.WorkerNodeConfig, cpApi *proto.CpiInterfaceClient) {
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
				Port:       int32(config.Port),
				CpuCores:   hardware.GetNumberCpus(),
				MemorySize: hardware.GetMemory(),
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
	harwareUsage := hardware.GetHardwareUsage()

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
			if err != nil || resp == nil {
				return false, err
			}

			return resp.Success, nil
		},
	)
	if pollErr != nil {
		logrus.Warn(fmt.Sprintf("Failed to send a heartbeat to the control plane : %s", pollErr))
	} else {
		logrus.Debug("Sent heartbeat to the control plane")
	}
}

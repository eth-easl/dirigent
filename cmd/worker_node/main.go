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
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	REGISTER_ACTION = iota
	DEREGISTER_ACTION
)

func main() {
	config, err := config2.ReadWorkedNodeConfiguration("cmd/worker_node/config.yaml")
	if err != nil {
		logrus.Fatal("Failed to read configuration file (error : %s)", err.Error())
	}

	logger.SetupLogger(config.Verbosity)

	containerdClient := sandbox.GetContainerdClient(config.CRIPath)
	defer containerdClient.Close()

	cniClient := sandbox.GetCNIClient(config.CNIConfigPath)
	ipt, err := sandbox.NewIptablesUtil()

	if err != nil {
		logrus.Fatal("Error while accessing iptables - ", err)
	}

	cpApi := common.InitializeControlPlaneConnection(config.ControlPlaneIp, config.ControlPlanePort, -1, -1)

	quitChannel := make(chan bool)

	registerNodeWithControlPlane(config, &cpApi)
	defer deregisterNodeFromControlPlane(config, &cpApi, quitChannel)

	go setupHeartbeatLoop(&cpApi, quitChannel)

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

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	select {
	case <-ctx.Done():
		logrus.Info("Received interruption signal, try to gracefully stop")
	}
}

func registerNodeWithControlPlane(config config2.WorkerNodeConfig, cpApi *proto.CpiInterfaceClient) {
	logrus.Info("Trying to register the node with the control plane")

	err := sendInstructionToControlPlane(config, cpApi, REGISTER_ACTION)
	if err != nil {
		logrus.Fatal("Failed to register from the control plane")
	}

	logrus.Info("Successfully registered the node with the control plane")
}

func deregisterNodeFromControlPlane(config config2.WorkerNodeConfig, cpApi *proto.CpiInterfaceClient, quitChannel chan bool) {
	logrus.Info("Trying to deregister the node with the control plane")

	quitChannel <- true

	err := sendInstructionToControlPlane(config, cpApi, DEREGISTER_ACTION)
	if err != nil {
		logrus.Fatal("Failed to deregister from the control plane")
	}

	logrus.Info("Successfully registered the node with the control plane")
}

func sendInstructionToControlPlane(config config2.WorkerNodeConfig, cpi *proto.CpiInterfaceClient, action int) error {
	hostName, err := os.Hostname()
	if err != nil {
		logrus.Warn("Error fetching host name.")
	}

	pollContext, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pollErr := wait.PollUntilContextCancel(pollContext, 5*time.Second, false,
		func(ctx context.Context) (done bool, err error) {
			nodeInfo := &proto.NodeInfo{
				NodeID: hostName,
				// IP fetched from server-side context
				Port:       int32(config.Port),
				CpuCores:   hardware.GetNumberCpus(),
				MemorySize: hardware.GetMemory(),
			}

			var resp *proto.ActionStatus

			if action == REGISTER_ACTION {
				resp, err = (*cpi).RegisterNode(context.Background(), nodeInfo)
			} else if action == DEREGISTER_ACTION {
				resp, err = (*cpi).DeregisterNode(context.Background(), nodeInfo)
			}

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

	return nil
}

func setupHeartbeatLoop(cpApi *proto.CpiInterfaceClient, quitChannel chan bool) {

	for {
		// Quit (if required) or Send
		select {
		case <-quitChannel:
			return
		default:
			sendHeartbeatLoop(cpApi)
		}

		// Wait
		time.Sleep(utils.HeartbeatInterval)
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

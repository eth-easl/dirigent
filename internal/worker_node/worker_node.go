package worker_node

import (
	"cluster_manager/api/proto"
	"cluster_manager/internal/worker_node/managers"
	"cluster_manager/internal/worker_node/sandbox"
	"cluster_manager/internal/worker_node/sandbox/containerd"
	"cluster_manager/internal/worker_node/sandbox/firecracker"
	"cluster_manager/pkg/config"
	"cluster_manager/pkg/hardware"
	"cluster_manager/pkg/utils"
	"context"
	"fmt"
	"os"
	"time"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	RegisterAction = iota
	DeregisterAction
)

type WorkerNode struct {
	cpApi          proto.CpiInterfaceClient
	SandboxRuntime sandbox.RuntimeInterface

	ImageManager   *containerd.ImageManager
	SandboxManager *managers.SandboxManager
	ProcessMonitor *managers.ProcessMonitor

	quitChannel chan bool
}

func isUserRoot() (int, bool) {
	uid := os.Getuid()

	return uid, uid == 0
}

func NewWorkerNode(cpApi proto.CpiInterfaceClient, config config.WorkerNodeConfig) *WorkerNode {
	hostName, err := os.Hostname()
	if err != nil {
		logrus.Warn("Error fetching host name.")
	}

	imageManager := containerd.NewImageManager()
	sandboxManager := managers.NewSandboxManager(hostName)
	processMonitor := managers.NewProcessMonitor()

	if _, isRoot := isUserRoot(); !isRoot {
		logrus.Fatal("Cannot create a worker daemon without sudo.")
	}

	var runtimeInterface sandbox.RuntimeInterface
	switch config.CRIType {
	case "containerd":
		runtimeInterface = containerd.NewContainerdRuntime(cpApi, config, imageManager, sandboxManager, processMonitor)
	case "firecracker":
		runtimeInterface = firecracker.NewFirecrackerRuntime()
	default:
		logrus.Fatal("Unsupported sandbox type.")
	}

	workerNode := &WorkerNode{
		cpApi:          cpApi,
		SandboxRuntime: runtimeInterface,

		ImageManager:   imageManager,
		SandboxManager: sandboxManager,
		ProcessMonitor: processMonitor,

		quitChannel: make(chan bool),
	}

	return workerNode
}

func (w *WorkerNode) RegisterNodeWithControlPlane(config config.WorkerNodeConfig, cpApi *proto.CpiInterfaceClient) {
	logrus.Info("Trying to register the node with the control plane")

	err := w.sendInstructionToControlPlane(config, cpApi, RegisterAction)
	if err != nil {
		logrus.Fatal("Failed to register from the control plane")
	}

	logrus.Info("Successfully registered the node with the control plane")
}

func (w *WorkerNode) StopWorkerNode(config config.WorkerNodeConfig, cpApi *proto.CpiInterfaceClient) {
	w.CleanResources()
	w.DeregisterNodeFromControlPlane(config, cpApi)
}

func (w *WorkerNode) CleanResources() {
	keys := make([]string, 0)
	for _, key := range w.SandboxManager.Metadata.Keys() {
		keys = append(keys, key)
	}

	for _, key := range keys {
		_, err := w.SandboxRuntime.DeleteSandbox(context.Background(), &proto.SandboxID{
			ID: key,
		})
		if err != nil {
			logrus.Warn("Failed to clean resource (sandbox) - ", key)
		}
	}
}

func (w *WorkerNode) DeregisterNodeFromControlPlane(config config.WorkerNodeConfig, cpApi *proto.CpiInterfaceClient) {
	logrus.Info("Trying to deregister the node with the control plane")

	w.quitChannel <- true

	err := w.sendInstructionToControlPlane(config, cpApi, DeregisterAction)
	if err != nil {
		logrus.Fatal("Failed to deregister from the control plane")
	}

	logrus.Info("Successfully registered the node with the control plane")
}

func (w *WorkerNode) sendInstructionToControlPlane(config config.WorkerNodeConfig, cpi *proto.CpiInterfaceClient, action int) error {
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

			if action == RegisterAction {
				resp, err = (*cpi).RegisterNode(context.Background(), nodeInfo)
			} else if action == DeregisterAction {
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

func (w *WorkerNode) SetupHeartbeatLoop(cpApi *proto.CpiInterfaceClient) {
	for {
		// Quit (if required) or Send
		select {
		case <-w.quitChannel:
			return
		default:
			w.sendHeartbeatLoop(cpApi)
		}

		// Wait
		time.Sleep(utils.HeartbeatInterval)
	}
}

func (w *WorkerNode) getWorkerStatistics() (*proto.NodeHeartbeatMessage, error) {
	hardwareUsage := hardware.GetHardwareUsage()

	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}

	return &proto.NodeHeartbeatMessage{
		NodeID:      hostname,
		CpuUsage:    hardwareUsage.CpuUsage,
		MemoryUsage: hardwareUsage.MemoryUsage,
	}, nil
}

func (w *WorkerNode) sendHeartbeatLoop(cpApi *proto.CpiInterfaceClient) {
	pollContext, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pollErr := wait.PollUntilContextCancel(pollContext, 5*time.Second, false,
		func(ctx context.Context) (done bool, err error) {
			workerStatistics, err := w.getWorkerStatistics()
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

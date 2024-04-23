package worker_node

import (
	"cluster_manager/internal/worker_node/managers"
	"cluster_manager/internal/worker_node/sandbox"
	"cluster_manager/internal/worker_node/sandbox/containerd"
	"cluster_manager/internal/worker_node/sandbox/fake_snapshot"
	"cluster_manager/internal/worker_node/sandbox/firecracker"
	"cluster_manager/pkg/config"
	"cluster_manager/pkg/grpc_helpers"
	"cluster_manager/pkg/hardware"
	"cluster_manager/pkg/utils"
	"cluster_manager/proto"
	"context"
	"fmt"
	"google.golang.org/protobuf/types/known/emptypb"
	"math/rand"
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
	proto.UnimplementedWorkerNodeInterfaceServer

	cpApi          proto.CpiInterfaceClient
	SandboxRuntime sandbox.RuntimeInterface

	ImageManager   *containerd.ImageManager
	SandboxManager *managers.SandboxManager
	ProcessMonitor *managers.ProcessMonitor

	Name string
}

func isUserRoot() (int, bool) {
	uid := os.Getuid()

	return uid, uid == 0
}

func NewWorkerNode(cpApi proto.CpiInterfaceClient, config config.WorkerNodeConfig, name ...string) *WorkerNode {
	hostName, err := os.Hostname()
	if err != nil {
		logrus.Warn("Error fetching host name.")
	}

	if len(name) > 0 {
		hostName = name[0]
	}

	nodeName := fmt.Sprintf("%s-%d", hostName, rand.Int())

	sandboxManager := managers.NewSandboxManager(nodeName)

	if _, isRoot := isUserRoot(); !isRoot {
		logrus.Fatal("Cannot create a worker daemon without sudo.")
	}

	var runtimeInterface sandbox.RuntimeInterface
	switch config.CRIType {
	case "containerd":
		runtimeInterface = containerd.NewContainerdRuntime(
			cpApi,
			config,
			sandboxManager,
		)
	case "firecracker":
		runtimeInterface = firecracker.NewFirecrackerRuntime(
			cpApi,
			sandboxManager,
			config.FirecrackerKernel,
			config.FirecrackerFileSystem,
			config.FirecrackerInternalIPPrefix,
			config.FirecrackerExposedIPPrefix,
			config.FirecrackerVMDebugMode,
			config.FirecrackerUseSnapshots,
			config.FirecrackerNetworkPoolSize,
		)
	case "scalability_test":
		runtimeInterface = fake_snapshot.NewFakeSnapshotRuntime()
	default:
		logrus.Fatal("Unsupported sandbox type.")
	}

	if !runtimeInterface.ValidateHostConfig() {
		logrus.Fatal("The host machine configuration is invalid or it does not support required features. Terminating worker daemon.")
	}

	workerNode := &WorkerNode{
		cpApi:          cpApi,
		SandboxRuntime: runtimeInterface,

		SandboxManager: sandboxManager,

		Name: nodeName,
	}

	return workerNode
}

func (w *WorkerNode) CreateSandbox(grpcCtx context.Context, in *proto.ServiceInfo) (*proto.SandboxCreationStatus, error) {
	return w.SandboxRuntime.CreateSandbox(grpcCtx, in)
}

func (w *WorkerNode) DeleteSandbox(grpcCtx context.Context, in *proto.SandboxID) (*proto.ActionStatus, error) {
	return w.SandboxRuntime.DeleteSandbox(grpcCtx, in)
}

func (w *WorkerNode) ListEndpoints(grpcCtx context.Context, in *emptypb.Empty) (*proto.EndpointsList, error) {
	return w.SandboxRuntime.ListEndpoints(grpcCtx, in)
}

func (w *WorkerNode) RegisterNodeWithControlPlane(config config.WorkerNodeConfig, cpApi *proto.CpiInterfaceClient) {
	logrus.Info("Trying to register the node with the control plane")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := w.sendInstructionToControlPlane(ctx, config, cpApi, RegisterAction)
	if err != nil {
		logrus.Fatal("Failed to register from the control plane")
	}

	logrus.Info("Successfully registered the node with the control plane")
}

func (w *WorkerNode) DeregisterNodeFromControlPlane(config config.WorkerNodeConfig, cpApi *proto.CpiInterfaceClient) {
	logrus.Info("Trying to deregister the node with the control plane")

	pollContext, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := w.sendInstructionToControlPlane(pollContext, config, cpApi, DeregisterAction)
	if err != nil {
		logrus.Fatal("Failed to deregister from the control plane")
	}

	logrus.Info("Successfully registered the node with the control plane, cleaning resources")

	w.cleanResources()

	logrus.Info("Successfully clean-up resources")
}

func (w *WorkerNode) SetupHeartbeatLoop(cfg *config.WorkerNodeConfig) {
	for {
		// Send
		w.sendHeartbeatLoop(cfg)

		// Wait
		time.Sleep(utils.HeartbeatInterval)
	}
}

func (w *WorkerNode) sendInstructionToControlPlane(ctx context.Context, config config.WorkerNodeConfig, cpi *proto.CpiInterfaceClient, action int) error {
	pollErr := wait.PollUntilContextCancel(ctx, 5*time.Second, true,
		func(ctx context.Context) (done bool, err error) {
			nodeInfo := &proto.NodeInfo{
				NodeID:     w.Name,
				IP:         config.WorkerNodeIP,
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

func (w *WorkerNode) cleanResources() {
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

func (w *WorkerNode) getWorkerStatistics() (*proto.NodeHeartbeatMessage, error) {
	hardwareUsage := hardware.GetHardwareUsage()

	return &proto.NodeHeartbeatMessage{
		NodeID:      w.Name,
		CpuUsage:    hardwareUsage.CpuUsage,
		MemoryUsage: hardwareUsage.MemoryUsage,
	}, nil
}

func (w *WorkerNode) sendHeartbeatLoop(cfg *config.WorkerNodeConfig) {
	pollContext, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pollErr := wait.PollUntilContextCancel(pollContext, 2500*time.Millisecond, true,
		func(ctx context.Context) (done bool, err error) {
			workerStatistics, err := w.getWorkerStatistics()
			if err != nil {
				return false, err
			}

			resp, err := w.cpApi.NodeHeartbeat(ctx, workerStatistics)

			// In case we don't manage to connect, we give up
			if err != nil || resp == nil {
				return false, err
			}

			return resp.Success, nil
		},
	)
	if pollErr != nil {
		logrus.Warnf("Failed to send a heartbeat to the control plane : %s", pollErr)
		logrus.Warnf("Trying to establish connection with some other control plane replica.")
		cpApi, err := grpc_helpers.NewControlPlaneConnection(cfg.ControlPlaneAddress)
		if err != nil {
			logrus.Fatalf("Cannot establish connection with any of the specified control plane(s) (error : %s)", err.Error())
		} else {
			w.cpApi = cpApi
			logrus.Infof("Control plance changed successfully.")
		}
	} else {
		logrus.Debug("Sent heartbeat to the control plane")
	}
}

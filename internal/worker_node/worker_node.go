package worker_node

import (
	"cluster_manager/api/proto"
	"cluster_manager/internal/worker_node/sandbox"
	"cluster_manager/pkg/config"
	"cluster_manager/pkg/hardware"
	"cluster_manager/pkg/utils"
	"context"
	"fmt"
	"github.com/containerd/containerd"
	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/go-cni"
	"github.com/coreos/go-iptables/iptables"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/durationpb"
	"k8s.io/apimachinery/pkg/util/wait"
	"os"
	"time"
)

const (
	REGISTER_ACTION = iota
	DEREGISTER_ACTION
)

type WorkerNode struct {
	proto.UnimplementedWorkerNodeInterfaceServer

	ContainerdClient *containerd.Client
	CNIClient        cni.CNI
	IPT              *iptables.IPTables

	ImageManager   *sandbox.ImageManager
	SandboxManager *sandbox.Manager

	quitChannel chan bool
}

func NewWorkerNode(config config.WorkerNodeConfig, containerdClient *containerd.Client) *WorkerNode {
	cniClient := sandbox.GetCNIClient(config.CNIConfigPath)
	ipt, err := sandbox.NewIptablesUtil()
	if err != nil {
		logrus.Fatal("Error while accessing iptables - ", err)
	}

	return &WorkerNode{
		ContainerdClient: containerdClient,
		CNIClient:        cniClient,
		IPT:              ipt,

		ImageManager:   sandbox.NewImageManager(),
		SandboxManager: sandbox.NewSandboxManager(),

		quitChannel: make(chan bool),
	}
}

func (w *WorkerNode) CreateSandbox(grpcCtx context.Context, in *proto.ServiceInfo) (*proto.SandboxCreationStatus, error) {
	logrus.Debug("Create sandbox for service = '", in.Name, "'")

	start := time.Now()

	ctx := namespaces.WithNamespace(grpcCtx, "cm")
	image, err, durationFetch := w.ImageManager.GetImage(ctx, w.ContainerdClient, in.Image)

	if err != nil {
		logrus.Warn("Failed fetching image - ", err)
		return &proto.SandboxCreationStatus{Success: false}, err
	}

	container, err, durationContainerCreation := sandbox.CreateContainer(ctx, w.ContainerdClient, image)
	if err != nil {
		logrus.Warn("Failed creating a container - ", err)
		return &proto.SandboxCreationStatus{Success: false}, err
	}

	task, exitChannel, ip, netNs, err, durationContainerStart, durationCNI := sandbox.StartContainer(ctx, container, w.CNIClient)
	if err != nil {
		logrus.Warn("Failed starting a container - ", err)
		return &proto.SandboxCreationStatus{Success: false}, err
	}

	metadata := &sandbox.Metadata{
		Task:        task,
		Container:   container,
		ExitChannel: exitChannel,
		HostPort:    sandbox.AssignRandomPort(),
		IP:          ip,
		GuestPort:   int(in.PortForwarding.GuestPort),
		NetNs:       netNs,
	}
	w.SandboxManager.AddSandbox(container.ID(), metadata)

	logrus.Debug("Sandbox creation took ", time.Since(start).Microseconds(), " μs (", container.ID(), ")")

	startIptables := time.Now()

	sandbox.AddRules(w.IPT, metadata.HostPort, metadata.IP, metadata.GuestPort)

	durationIptables := time.Since(startIptables)

	logrus.Debug("IP tables configuration (add rule(s)) took ", durationIptables.Microseconds(), " μs")

	in.PortForwarding.HostPort = int32(metadata.HostPort)

	return &proto.SandboxCreationStatus{
		Success:      true,
		ID:           container.ID(),
		PortMappings: in.PortForwarding,
		LatencyBreakdown: &proto.SandboxCreationBreakdown{
			Total:           durationpb.New(time.Since(start)),
			ImageFetch:      durationpb.New(durationFetch),
			ContainerCreate: durationpb.New(durationContainerCreation),
			CNI:             durationpb.New(durationCNI),
			ContainerStart:  durationpb.New(durationContainerStart),
			Iptables:        durationpb.New(durationIptables),
		},
	}, nil
}

func (w *WorkerNode) DeleteSandbox(grpcCtx context.Context, in *proto.SandboxID) (*proto.ActionStatus, error) {
	logrus.Debug("Delete sandbox with ID = '", in.ID, "'")

	ctx := namespaces.WithNamespace(grpcCtx, "cm")
	metadata := w.SandboxManager.DeleteSandbox(in.ID)

	if metadata == nil {
		logrus.Warn("Tried to delete non-existing sandbox ", in.ID)
		return &proto.ActionStatus{Success: false}, nil
	}

	start := time.Now()

	sandbox.DeleteRules(w.IPT, metadata.HostPort, metadata.IP, metadata.GuestPort)
	sandbox.UnassignPort(metadata.HostPort)
	logrus.Debug("IP tables configuration (remove rule(s)) took ", time.Since(start).Microseconds(), " μs")

	start = time.Now()
	err := sandbox.DeleteContainer(
		ctx,
		w.CNIClient,
		metadata,
	)

	if err != nil {
		logrus.Warn(err)
		return &proto.ActionStatus{Success: false}, err
	}

	logrus.Debug("Sandbox deletion took ", time.Since(start).Microseconds(), " μs")

	return &proto.ActionStatus{Success: true}, nil
}

func (w *WorkerNode) RegisterNodeWithControlPlane(config config.WorkerNodeConfig, cpApi *proto.CpiInterfaceClient) {
	logrus.Info("Trying to register the node with the control plane")

	err := w.sendInstructionToControlPlane(config, cpApi, REGISTER_ACTION)
	if err != nil {
		logrus.Fatal("Failed to register from the control plane")
	}

	logrus.Info("Successfully registered the node with the control plane")
}

func (w *WorkerNode) DeregisterNodeFromControlPlane(config config.WorkerNodeConfig, cpApi *proto.CpiInterfaceClient) {
	logrus.Info("Trying to deregister the node with the control plane")

	w.quitChannel <- true

	err := w.sendInstructionToControlPlane(config, cpApi, DEREGISTER_ACTION)
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

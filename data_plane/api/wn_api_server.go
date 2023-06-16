package api

import (
	"cluster_manager/api/proto"
	"cluster_manager/sandbox"
	"context"
	"github.com/containerd/containerd"
	"github.com/containerd/go-cni"
	"github.com/sirupsen/logrus"
	"time"
)

type WnApiServer struct {
	proto.UnimplementedWorkerNodeInterfaceServer

	ContainerdClient *containerd.Client
	CNIClient        cni.CNI

	ImageManager   *sandbox.ImageManager
	SandboxManager *sandbox.Manager
}

func (w *WnApiServer) CreateSandbox(ctx context.Context, in *proto.ServiceInfo) (*proto.SandboxCreationStatus, error) {
	logrus.Debug("Create sandbox for service = '", in.Name, "'")

	start := time.Now()

	image, err := w.ImageManager.GetImage(ctx, w.ContainerdClient, in.Image)
	if err != nil {
		logrus.Warn("Failed fetching image - ", err)
		return &proto.SandboxCreationStatus{Success: false}, err
	}

	container, err := sandbox.CreateContainer(ctx, w.ContainerdClient, in.Name, image)
	if err != nil {
		logrus.Warn("Failed creating a container - ", err)
		return &proto.SandboxCreationStatus{Success: false}, err
	}

	task, exitChannel, ip, netNs, err := sandbox.StartContainer(ctx, container, w.CNIClient)
	if err != nil {
		logrus.Warn("Failed starting a container - ", err)
		return &proto.SandboxCreationStatus{Success: false}, err
	}

	metadata := &sandbox.Metadata{
		Task:        task,
		Container:   container,
		ExitChannel: exitChannel,
		IP:          ip,
		NetNs:       netNs,
	}
	w.SandboxManager.AddSandbox(container.ID(), metadata)

	timeTook := time.Since(start).Microseconds()
	logrus.Debug("Sandbox creation took ", timeTook, " μs (", container.ID(), ")")

	return &proto.SandboxCreationStatus{
		Success:      true,
		ID:           container.ID(),
		PortMappings: in.PortForwarding,
		TimeTookMs:   timeTook / 1000,
	}, nil
}

func (w *WnApiServer) DeleteSandbox(ctx context.Context, in *proto.SandboxID) (*proto.ActionStatus, error) {
	logrus.Debug("Delete sandbox with ID = '", in.ID, "'")

	start := time.Now()

	metadata := w.SandboxManager.DeleteSandbox(in.ID)
	if metadata == nil {
		logrus.Warn("Tried to delete non-existing sandbox ", in.ID)
		return &proto.ActionStatus{Success: false}, nil
	}

	err := sandbox.DeleteContainer(
		ctx,
		w.CNIClient,
		metadata,
	)
	if err != nil {
		logrus.Debug(err)
		return &proto.ActionStatus{Success: false}, err
	}

	logrus.Debug("Sandbox deletion took ", time.Since(start).Microseconds(), " μs")
	return &proto.ActionStatus{Success: true}, nil
}

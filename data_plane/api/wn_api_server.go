package api

import (
	"cluster_manager/api/proto"
	"cluster_manager/sandbox"
	"context"
	"github.com/containerd/containerd"
	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/go-cni"
	"github.com/coreos/go-iptables/iptables"
	"github.com/sirupsen/logrus"
	"time"
)

type WnApiServer struct {
	proto.UnimplementedWorkerNodeInterfaceServer

	ContainerdClient *containerd.Client
	CNIClient        cni.CNI
	IPT              *iptables.IPTables

	ImageManager   *sandbox.ImageManager
	SandboxManager *sandbox.Manager
}

func (w *WnApiServer) CreateSandbox(grpcCtx context.Context, in *proto.ServiceInfo) (*proto.SandboxCreationStatus, error) {
	logrus.Debug("Create sandbox for service = '", in.Name, "'")

	start := time.Now()

	ctx := namespaces.WithNamespace(grpcCtx, "cm")
	image, err := w.ImageManager.GetImage(ctx, w.ContainerdClient, in.Image)
	if err != nil {
		logrus.Warn("Failed fetching image - ", err)
		return &proto.SandboxCreationStatus{Success: false}, err
	}

	container, err := sandbox.CreateContainer(ctx, w.ContainerdClient, image)
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
		HostPort:    sandbox.AssignRandomPort(),
		IP:          ip,
		GuestPort:   int(in.PortForwarding.GuestPort),
		NetNs:       netNs,
	}
	w.SandboxManager.AddSandbox(container.ID(), metadata)

	timeTook := time.Since(start).Microseconds()
	logrus.Debug("Sandbox creation took ", timeTook, " μs (", container.ID(), ")")

	start = time.Now()
	sandbox.AddRules(w.IPT, metadata.HostPort, metadata.IP, metadata.GuestPort)
	logrus.Debug("IP tables configuration (add rule(s)) took ", time.Since(start).Microseconds(), " μs")

	return &proto.SandboxCreationStatus{
		Success:      true,
		ID:           container.ID(),
		PortMappings: in.PortForwarding,
		TimeTookMs:   timeTook / 1000,
	}, nil
}

func (w *WnApiServer) DeleteSandbox(grpcCtx context.Context, in *proto.SandboxID) (*proto.ActionStatus, error) {
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

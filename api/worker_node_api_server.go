package api

import (
	"cluster_manager/api/proto"
	sandbox2 "cluster_manager/internal/sandbox"
	"context"
	"time"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/go-cni"
	"github.com/coreos/go-iptables/iptables"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/durationpb"
)

type WnApiServer struct {
	proto.UnimplementedWorkerNodeInterfaceServer

	ContainerdClient *containerd.Client
	CNIClient        cni.CNI
	IPT              *iptables.IPTables

	ImageManager   *sandbox2.ImageManager
	SandboxManager *sandbox2.Manager
}

func (w *WnApiServer) CreateSandbox(grpcCtx context.Context, in *proto.ServiceInfo) (*proto.SandboxCreationStatus, error) {
	logrus.Debug("Create sandbox for service = '", in.Name, "'")

	start := time.Now()

	ctx := namespaces.WithNamespace(grpcCtx, "cm")
	image, err, durationFetch := w.ImageManager.GetImage(ctx, w.ContainerdClient, in.Image)

	if err != nil {
		logrus.Warn("Failed fetching image - ", err)
		return &proto.SandboxCreationStatus{Success: false}, err
	}

	container, err, durationContainerCreation := sandbox2.CreateContainer(ctx, w.ContainerdClient, image)
	if err != nil {
		logrus.Warn("Failed creating a container - ", err)
		return &proto.SandboxCreationStatus{Success: false}, err
	}

	task, exitChannel, ip, netNs, err, durationContainerStart, durationCNI := sandbox2.StartContainer(ctx, container, w.CNIClient)
	if err != nil {
		logrus.Warn("Failed starting a container - ", err)
		return &proto.SandboxCreationStatus{Success: false}, err
	}

	metadata := &sandbox2.Metadata{
		Task:        task,
		Container:   container,
		ExitChannel: exitChannel,
		HostPort:    sandbox2.AssignRandomPort(),
		IP:          ip,
		GuestPort:   int(in.PortForwarding.GuestPort),
		NetNs:       netNs,
	}
	w.SandboxManager.AddSandbox(container.ID(), metadata)

	logrus.Debug("Sandbox creation took ", time.Since(start).Microseconds(), " μs (", container.ID(), ")")

	startIptables := time.Now()

	sandbox2.AddRules(w.IPT, metadata.HostPort, metadata.IP, metadata.GuestPort)

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

func (w *WnApiServer) DeleteSandbox(grpcCtx context.Context, in *proto.SandboxID) (*proto.ActionStatus, error) {
	logrus.Debug("Delete sandbox with ID = '", in.ID, "'")

	ctx := namespaces.WithNamespace(grpcCtx, "cm")
	metadata := w.SandboxManager.DeleteSandbox(in.ID)

	if metadata == nil {
		logrus.Warn("Tried to delete non-existing sandbox ", in.ID)
		return &proto.ActionStatus{Success: false}, nil
	}

	start := time.Now()

	sandbox2.DeleteRules(w.IPT, metadata.HostPort, metadata.IP, metadata.GuestPort)
	sandbox2.UnassignPort(metadata.HostPort)
	logrus.Debug("IP tables configuration (remove rule(s)) took ", time.Since(start).Microseconds(), " μs")

	start = time.Now()
	err := sandbox2.DeleteContainer(
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

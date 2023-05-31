package api

import (
	"cluster_manager/api/proto"
	"cluster_manager/sandbox"
	"context"
	"github.com/docker/docker/client"
	"github.com/sirupsen/logrus"
)

type WnApiServer struct {
	proto.UnimplementedWorkerNodeInterfaceServer
	DockerClient *client.Client

	containerList []string
}

func (w *WnApiServer) CreateSandbox(_ context.Context, in *proto.ServiceInfo) (*proto.ActionStatus, error) {
	host, guest, err := sandbox.CreateSandboxConfig(in.Image, in.PortForwarding)
	if err != nil {
		logrus.Warn(err)
		return &proto.ActionStatus{Success: false, Message: err.Error()}, nil
	}

	sandboxID, err := sandbox.CreateSandbox(w.DockerClient, in.Name, host, guest)
	if err != nil {
		logrus.Warn(err)
		return &proto.ActionStatus{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	w.containerList = append(w.containerList, sandboxID)
	tempPort := host.PortBindings["80/tcp"][0].HostPort

	return &proto.ActionStatus{
		Success: true,
		Message: tempPort,
	}, nil
}

func (w *WnApiServer) DeleteSandbox(_ context.Context, in *proto.ServiceInfo) (*proto.ActionStatus, error) {
	panic("implement me")

	// https://github.com/containerd/containerd/blob/main/docs/getting-started.md
	// defer container.Delete(ctx, containerd.WithSnapshotCleanup)
	// defer task.Delete(ctx)
	// status := <-exitStatusC
	// code, _, err := status.Result()
}

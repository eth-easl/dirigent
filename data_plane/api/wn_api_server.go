package api

import (
	"cluster_manager/api/proto"
	"cluster_manager/sandbox"
	"context"
	"github.com/docker/docker/client"
	"github.com/sirupsen/logrus"
	"time"
)

type WnApiServer struct {
	proto.UnimplementedWorkerNodeInterfaceServer
	DockerClient *client.Client
}

func (w *WnApiServer) CreateSandbox(_ context.Context, in *proto.ServiceInfo) (*proto.SandboxCreationStatus, error) {
	logrus.Debug("Create sandbox for service = '", in.Name, "'")

	start := time.Now()

	host, guest, err := sandbox.CreateSandboxConfig(in.Image, in.PortForwarding)
	if err != nil {
		logrus.Warn(err)
		return &proto.SandboxCreationStatus{Success: false}, err
	}

	sandboxID, err := sandbox.CreateSandbox(w.DockerClient, host, guest)
	if err != nil {
		logrus.Warn(err)
		return &proto.SandboxCreationStatus{Success: false}, err
	}

	logrus.Debug("Sandbox creation took ", time.Since(start).Microseconds(), " μs (", sandboxID, ")")
	return &proto.SandboxCreationStatus{Success: true, ID: sandboxID, PortMappings: in.PortForwarding}, nil
}

func (w *WnApiServer) DeleteSandbox(_ context.Context, in *proto.SandboxID) (*proto.ActionStatus, error) {
	logrus.Debug("Delete sandbox with ID = '", in.ID, "'")

	start := time.Now()

	sandbox.UnassignPort(int(in.HostPort))
	err := sandbox.DeleteSandbox(w.DockerClient, in.ID)
	if err != nil {
		logrus.Warn("Failed to delete sandbox with ID = '", in.ID, "' - ", err)
		return &proto.ActionStatus{Success: false}, nil
	}

	logrus.Debug("Sandbox creation took ", time.Since(start).Microseconds(), " μs")
	return &proto.ActionStatus{Success: true}, nil
}

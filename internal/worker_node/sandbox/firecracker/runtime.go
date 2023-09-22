package firecracker

import (
	"cluster_manager/api/proto"
	"cluster_manager/internal/worker_node/sandbox"
	"context"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
	"google.golang.org/protobuf/types/known/emptypb"
)

type Runtime struct {
	sandbox.RuntimeInterface

	ipManager IPManager
}

func NewFirecrackerRuntime() *Runtime {
	return &Runtime{}
}

func (fcr *Runtime) CreateSandbox(ctx context.Context, in *proto.ServiceInfo) (*proto.SandboxCreationStatus, error) {
	panic("Not yet implemented")
}

func (fcr *Runtime) DeleteSandbox(ctx context.Context, in *proto.SandboxID) (*proto.ActionStatus, error) {
	panic("Not yet implemented")
}

func (fcr *Runtime) ListEndpoints(ctx context.Context, _ *emptypb.Empty) (*proto.EndpointsList, error) {
	panic("Not yet implemented")
}

func (fcr *Runtime) ValidateHostConfig() bool {
	// Check for KVM access
	err := unix.Access("/dev/kvm", unix.W_OK)
	if err != nil {
		logrus.Error("KVM access denied - ", err)
		return false
	}

	return true
}

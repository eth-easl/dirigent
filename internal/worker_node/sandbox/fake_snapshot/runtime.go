package fake_snapshot

import (
	"cluster_manager/internal/worker_node/sandbox"
	"cluster_manager/proto"
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/emptypb"
	"math/rand"
	"time"
)

type Runtime struct {
	sandbox.RuntimeInterface
}

func NewFakeSnapshotRuntime() *Runtime {
	return &Runtime{}
}

func (fsr *Runtime) CreateSandbox(_ context.Context, in *proto.ServiceInfo) (*proto.SandboxCreationStatus, error) {
	time.Sleep(40 * time.Millisecond)
	logrus.Debugf("Fake sandbox created successfully.")

	zeroDuration := durationpb.New(0)

	return &proto.SandboxCreationStatus{
		Success: true,
		ID:      fmt.Sprintf("dummy-sandbox-%d", rand.Int()),
		PortMappings: &proto.PortMapping{
			HostPort:  80,
			GuestPort: 80,
			Protocol:  proto.L4Protocol_TCP,
		},
		LatencyBreakdown: &proto.SandboxCreationBreakdown{
			Total:                zeroDuration,
			ImageFetch:           zeroDuration,
			SandboxCreate:        zeroDuration,
			NetworkSetup:         zeroDuration,
			SandboxStart:         zeroDuration,
			Iptables:             zeroDuration,
			ReadinessProbing:     zeroDuration,
			DataplanePropagation: zeroDuration,
			SnapshotCreation:     zeroDuration,
			ConfigureMonitoring:  zeroDuration,
			FindSnapshot:         zeroDuration,
		},
	}, nil
}

func (fsr *Runtime) DeleteSandbox(_ context.Context, _ *proto.SandboxID) (*proto.ActionStatus, error) {
	return &proto.ActionStatus{Success: true}, nil
}

func (fsr *Runtime) ListEndpoints(_ context.Context, _ *emptypb.Empty) (*proto.EndpointsList, error) {
	return &proto.EndpointsList{Endpoint: nil}, nil
}

func (fsr *Runtime) ValidateHostConfig() bool {
	return true
}

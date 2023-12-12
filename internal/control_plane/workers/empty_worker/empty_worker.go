package empty_worker

import (
	"cluster_manager/api/proto"
	"cluster_manager/internal/control_plane/core"
	"cluster_manager/pkg/synchronization"
	"context"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	durationpb "google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/emptypb"
	"time"
)

type emptyWorker struct {
	name              string
	workerEndPointMap synchronization.SyncStructure[*core.Endpoint, string]
}

func (e *emptyWorker) SetSchedulability(b bool) {
}

func (e *emptyWorker) GetSchedulability() bool {
	return true
}

func NewEmptyWorkerNode(workerNodeConfiguration core.WorkerNodeConfiguration) core.WorkerNodeInterface {
	return &emptyWorker{
		name:              workerNodeConfiguration.Name,
		workerEndPointMap: synchronization.NewControlPlaneSyncStructure[*core.Endpoint, string](),
	}
}

func (e *emptyWorker) ConnectToWorker() proto.WorkerNodeInterfaceClient {
	return NewEmptyInterfaceClient()
}

func (e *emptyWorker) CreateSandbox(ctx context.Context, info *proto.ServiceInfo, option ...grpc.CallOption) (*proto.SandboxCreationStatus, error) {
	logrus.Debug("Creation sandbox")
	return &proto.SandboxCreationStatus{
		Success: true,
		ID:      uuid.New().String(),
		PortMappings: &proto.PortMapping{
			HostPort:  0,
			GuestPort: 0,
			Protocol:  0,
		},
		LatencyBreakdown: &proto.SandboxCreationBreakdown{
			Total:                &durationpb.Duration{},
			ImageFetch:           &durationpb.Duration{},
			SandboxCreate:        &durationpb.Duration{},
			NetworkSetup:         &durationpb.Duration{},
			SandboxStart:         &durationpb.Duration{},
			Iptables:             &durationpb.Duration{},
			DataplanePropagation: &durationpb.Duration{},
		},
	}, nil
}

func (e *emptyWorker) DeleteSandbox(ctx context.Context, id *proto.SandboxID, option ...grpc.CallOption) (*proto.ActionStatus, error) {
	logrus.Debug("Deletion sandbox")
	return &proto.ActionStatus{
		Success: true,
	}, nil
}

func (e *emptyWorker) ListEndpoints(ctx context.Context, empty *emptypb.Empty, option ...grpc.CallOption) (*proto.EndpointsList, error) {
	return &proto.EndpointsList{}, nil
}

func (e *emptyWorker) GetName() string {
	return e.name
}

func (e *emptyWorker) GetLastHeartBeat() time.Time {
	return time.Now()
}

func (e *emptyWorker) GetWorkerNodeConfiguration() core.WorkerNodeConfiguration {
	return core.WorkerNodeConfiguration{}
}

func (e *emptyWorker) UpdateLastHearBeat() {
}

func (e *emptyWorker) SetCpuUsage(u uint64) {
}

func (e *emptyWorker) SetMemoryUsage(u uint64) {
}

func (e *emptyWorker) GetMemory() uint64 {
	return 0
}

func (e *emptyWorker) GetCpuCores() uint64 {
	return 0
}

func (e *emptyWorker) GetCpuUsage() uint64 {
	return 0
}

func (e *emptyWorker) GetMemoryUsage() uint64 {
	return 0
}

func (e *emptyWorker) GetIP() string {
	return ""
}

func (e *emptyWorker) GetPort() string {
	return ""
}

func (e *emptyWorker) GetEndpointMap() synchronization.SyncStructure[*core.Endpoint, string] {
	return e.workerEndPointMap
}

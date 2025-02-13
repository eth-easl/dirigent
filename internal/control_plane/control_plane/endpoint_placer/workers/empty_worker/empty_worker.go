package empty_worker

import (
	"cluster_manager/internal/control_plane/control_plane/core"
	"cluster_manager/pkg/synchronization"
	"cluster_manager/proto"
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	durationpb "google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/emptypb"
)

type emptyWorker struct {
	name              string
	schedulability    bool
	workerEndPointMap synchronization.SyncStructure[*core.Endpoint, string]
}

func (e *emptyWorker) SetSchedulability(b bool) {
	e.schedulability = b
}

func (e *emptyWorker) GetSchedulability() bool {
	return e.schedulability
}

func NewEmptyWorkerNode(workerNodeConfiguration core.WorkerNodeConfiguration) core.WorkerNodeInterface {
	return &emptyWorker{
		name:              workerNodeConfiguration.Name,
		schedulability:    true,
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
		URL:     "",
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
	return &proto.ActionStatus{Success: true}, nil
}

func (e *emptyWorker) CreateTaskSandbox(_ context.Context, _ *proto.WorkflowTaskInfo, _ ...grpc.CallOption) (*proto.SandboxCreationStatus, error) {
	logrus.Debug("Creation task sandbox")
	return &proto.SandboxCreationStatus{
		Success: true,
		ID:      uuid.New().String(),
		URL:     "",
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

func (e *emptyWorker) ReceiveRouteUpdate(ctx context.Context, update *proto.RouteUpdate, option ...grpc.CallOption) (*proto.ActionStatus, error) {
	return &proto.ActionStatus{Success: true}, nil
}

func (e *emptyWorker) ListEndpoints(ctx context.Context, empty *emptypb.Empty, option ...grpc.CallOption) (*proto.EndpointsList, error) {
	return &proto.EndpointsList{}, nil
}

func (e *emptyWorker) PrepullImage(ctx context.Context, id *proto.ImageInfo, option ...grpc.CallOption) (*proto.ActionStatus, error) {
	return &proto.ActionStatus{Success: true}, nil
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

func (e *emptyWorker) SetCpuUsed(u uint64) {
}

func (e *emptyWorker) SetMemoryUsed(u uint64) {
}

func (e *emptyWorker) AddUsage(c, m uint64) {
}

func (e *emptyWorker) GetCpuAvailable() uint64 {
	return 0
}

func (e *emptyWorker) GetCpuUsed() uint64 {
	return 0
}

func (e *emptyWorker) GetMemoryAvailable() uint64 {
	return 0
}

func (e *emptyWorker) GetMemoryUsed() uint64 {
	return 0
}

func (e *emptyWorker) GetGpuAvailable() uint64 {
	return 0
}

func (e *emptyWorker) GetIP() string {
	return ""
}

func (e *emptyWorker) GetPort() string {
	return ""
}

func (e *emptyWorker) GetCIDR() string {
	return ""
}

func (e *emptyWorker) GetEndpointMap() synchronization.SyncStructure[*core.Endpoint, string] {
	return e.workerEndPointMap
}

func (e *emptyWorker) AddImage(string) bool {
	return false
}

func (e *emptyWorker) RemoveImage(string) bool {
	return false
}

func (e *emptyWorker) HasImage(string) bool {
	return false
}

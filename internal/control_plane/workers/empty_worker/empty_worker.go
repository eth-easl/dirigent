package empty_worker

import (
	"cluster_manager/api/proto"
	"cluster_manager/internal/control_plane/core"
	"cluster_manager/pkg/synchronization"
	"context"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"time"
)

type emptyWorker struct{}

func NewEmptyWorkerNode(workerNodeConfiguration core.WorkerNodeConfiguration) core.WorkerNodeInterface {
	return &emptyWorker{}
}

func (e *emptyWorker) GetAPI() proto.WorkerNodeInterfaceClient {
	return NewEmptyInterfaceClient()
}

func (e *emptyWorker) CreateSandbox(ctx context.Context, info *proto.ServiceInfo, option ...grpc.CallOption) (*proto.SandboxCreationStatus, error) {
	return &proto.SandboxCreationStatus{}, nil
}

func (e *emptyWorker) DeleteSandbox(ctx context.Context, id *proto.SandboxID, option ...grpc.CallOption) (*proto.ActionStatus, error) {
	return &proto.ActionStatus{}, nil
}

func (e *emptyWorker) ListEndpoints(ctx context.Context, empty *emptypb.Empty, option ...grpc.CallOption) (*proto.EndpointsList, error) {
	return &proto.EndpointsList{}, nil
}

func (e *emptyWorker) GetName() string {
	return ""
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
	return synchronization.NewControlPlaneSyncStructure[*core.Endpoint, string]()
}

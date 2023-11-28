package workers

import (
	"cluster_manager/api/proto"
	"cluster_manager/internal/control_plane/core"
	"cluster_manager/pkg/grpc_helpers"
	"cluster_manager/pkg/synchronization"
	"context"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"time"
)

type WorkerNode struct {
	Name string
	IP   string
	Port string

	CpuUsage    uint64
	MemoryUsage uint64

	CpuCores uint64
	Memory   uint64

	LastHeartbeat time.Time
	api           proto.WorkerNodeInterfaceClient

	endpointMap synchronization.SyncStructure[*core.Endpoint, string]
	Schedulable bool
}

func (w *WorkerNode) SetSchedulability(val bool) {
	w.Schedulable = val
}

func (w *WorkerNode) GetSchedulability() bool {
	return w.Schedulable
}

func NewWorkerNode(workerNodeConfiguration core.WorkerNodeConfiguration) core.WorkerNodeInterface {
	return &WorkerNode{
		Name:          workerNodeConfiguration.Name,
		IP:            workerNodeConfiguration.IP,
		Port:          workerNodeConfiguration.Port,
		CpuCores:      workerNodeConfiguration.CpuCores,
		Memory:        workerNodeConfiguration.Memory,
		LastHeartbeat: time.Now(),
		endpointMap:   synchronization.NewControlPlaneSyncStructure[*core.Endpoint, string](),
	}
}

func (w *WorkerNode) GetWorkerNodeConfiguration() core.WorkerNodeConfiguration {
	return core.WorkerNodeConfiguration{
		Name:     w.Name,
		IP:       w.IP,
		Port:     w.Port,
		CpuCores: w.CpuCores,
		Memory:   w.Memory,
	}
}

func (w *WorkerNode) GetName() string {
	return w.Name
}

func (w *WorkerNode) GetLastHeartBeat() time.Time {
	return w.LastHeartbeat
}

func (w *WorkerNode) CreateSandbox(ctx context.Context, info *proto.ServiceInfo, option ...grpc.CallOption) (*proto.SandboxCreationStatus, error) {
	return w.GetAPI().CreateSandbox(ctx, info, option...)
}

func (w *WorkerNode) DeleteSandbox(ctx context.Context, id *proto.SandboxID, option ...grpc.CallOption) (*proto.ActionStatus, error) {
	return w.GetAPI().DeleteSandbox(ctx, id, option...)
}

func (w *WorkerNode) ListEndpoints(ctx context.Context, empty *emptypb.Empty, option ...grpc.CallOption) (*proto.EndpointsList, error) {
	return w.ListEndpoints(ctx, empty, option...)
}

func (w *WorkerNode) GetAPI() proto.WorkerNodeInterfaceClient {
	if w.api == nil {
		w.api, _ = grpc_helpers.InitializeWorkerNodeConnection(w.IP, w.Port)
	}

	return w.api
}

func (w *WorkerNode) UpdateLastHearBeat() {
	w.LastHeartbeat = time.Now()
}

func (w *WorkerNode) SetCpuUsage(usage uint64) {
	w.CpuUsage = usage
}

func (w *WorkerNode) SetMemoryUsage(usage uint64) {
	w.MemoryUsage = usage
}

func (w *WorkerNode) GetIP() string {
	return w.IP
}

func (w *WorkerNode) GetPort() string {
	return w.Port
}

func (w *WorkerNode) GetMemory() uint64 {
	return w.Memory
}

func (w *WorkerNode) GetCpuCores() uint64 {
	return w.CpuCores
}

func (w *WorkerNode) GetCpuUsage() uint64 {
	return w.CpuUsage
}

func (w *WorkerNode) GetMemoryUsage() uint64 {
	return w.MemoryUsage
}

func (w *WorkerNode) GetEndpointMap() synchronization.SyncStructure[*core.Endpoint, string] {
	return w.endpointMap
}

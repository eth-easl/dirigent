package control_plane

import (
	"cluster_manager/api/proto"
	"cluster_manager/internal/control_plane/core"
	"cluster_manager/pkg/grpc_helpers"
	"context"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"time"
)

type WorkerNode struct {
	Name string
	IP   string
	Port string

	CpuUsage    int
	MemoryUsage int

	CpuCores int
	Memory   int

	LastHeartbeat time.Time
	api           proto.WorkerNodeInterfaceClient
}

func NewWorkerNode(workerNodeConfiguration core.WorkerNodeConfiguration) core.WorkerNodeInterface {
	return &WorkerNode{
		Name:          workerNodeConfiguration.Name,
		IP:            workerNodeConfiguration.IP,
		Port:          workerNodeConfiguration.Port,
		CpuCores:      workerNodeConfiguration.CpuCores,
		Memory:        workerNodeConfiguration.Memory,
		LastHeartbeat: time.Now(),
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

func (w *WorkerNode) GetIP() string {
	return w.IP
}

func (w *WorkerNode) GetPort() string {
	return w.Port
}

func (w *WorkerNode) UpdateLastHearBeat() {
	w.LastHeartbeat = time.Now()
}

func (w *WorkerNode) SetCpuUsage(usage int) {
	w.CpuUsage = usage
}

func (w *WorkerNode) SetMemoryUsage(usage int) {
	w.MemoryUsage = usage
}

func (w *WorkerNode) GetMemory() int {
	return w.Memory
}

func (w *WorkerNode) GetCpuCores() int {
	return w.CpuCores
}

func (w *WorkerNode) GetCpuUsage() int {
	return w.CpuUsage
}

func (w *WorkerNode) GetMemoryUsage() int {
	return w.MemoryUsage
}

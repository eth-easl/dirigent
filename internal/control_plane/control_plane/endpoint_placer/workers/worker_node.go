package workers

import (
	"cluster_manager/internal/control_plane/control_plane/core"
	"cluster_manager/pkg/grpc_helpers"
	"cluster_manager/pkg/synchronization"
	"cluster_manager/proto"
	"context"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type WorkerNode struct {
	Name string
	IP   string
	Port string
	CIDR string

	// In milli-CPU
	CpuUsed      uint64
	CpuAvailable uint64
	// In MiB
	MemoryUsed      uint64
	MemoryAvailable uint64

	LastHeartbeat time.Time
	wnConnection  proto.WorkerNodeInterfaceClient

	endpointMap synchronization.SyncStructure[*core.Endpoint, string]
	imageSet    synchronization.SyncStructure[string, struct{}]
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
		Name:            workerNodeConfiguration.Name,
		IP:              workerNodeConfiguration.IP,
		Port:            workerNodeConfiguration.Port,
		CpuAvailable:    workerNodeConfiguration.Cpu,
		MemoryAvailable: workerNodeConfiguration.Memory,
		LastHeartbeat:   time.Now(),
		endpointMap:     synchronization.NewControlPlaneSyncStructure[*core.Endpoint, string](),
		imageSet:        synchronization.NewControlPlaneSyncStructure[string, struct{}](),
	}
}

func (w *WorkerNode) GetWorkerNodeConfiguration() core.WorkerNodeConfiguration {
	return core.WorkerNodeConfiguration{
		Name:   w.Name,
		IP:     w.IP,
		Port:   w.Port,
		Cpu:    w.CpuAvailable,
		Memory: w.MemoryAvailable,
		CIDR:   w.CIDR,
	}
}

func (w *WorkerNode) GetName() string {
	return w.Name
}

func (w *WorkerNode) GetLastHeartBeat() time.Time {
	return w.LastHeartbeat
}

func (w *WorkerNode) CreateSandbox(ctx context.Context, info *proto.ServiceInfo, option ...grpc.CallOption) (*proto.SandboxCreationStatus, error) {
	return w.wnConnection.CreateSandbox(ctx, info, option...)
}

func (w *WorkerNode) DeleteSandbox(ctx context.Context, id *proto.SandboxID, option ...grpc.CallOption) (*proto.ActionStatus, error) {
	return w.wnConnection.DeleteSandbox(ctx, id, option...)
}

func (w *WorkerNode) CreateTaskSandbox(ctx context.Context, task *proto.WorkflowTaskInfo, option ...grpc.CallOption) (*proto.SandboxCreationStatus, error) {
	return w.wnConnection.CreateTaskSandbox(ctx, task, option...)
}

func (w *WorkerNode) ListEndpoints(ctx context.Context, empty *emptypb.Empty, option ...grpc.CallOption) (*proto.EndpointsList, error) {
	if w.wnConnection != nil {
		return w.wnConnection.ListEndpoints(ctx, empty, option...)
	} else {
		return &proto.EndpointsList{}, nil
	}
}

func (w *WorkerNode) PrepullImage(ctx context.Context, id *proto.ImageInfo, option ...grpc.CallOption) (*proto.ActionStatus, error) {
	return w.wnConnection.PrepullImage(ctx, id, option...)
}

func (w *WorkerNode) ConnectToWorker() proto.WorkerNodeInterfaceClient {
	if w.wnConnection == nil {
		var err error

		w.wnConnection, err = grpc_helpers.InitializeWorkerNodeConnection(w.IP, w.Port)
		if err != nil {
			logrus.Errorf("Failed to establish connection with the worker node.")
		}
	}

	return w.wnConnection
}

func (w *WorkerNode) UpdateLastHearBeat() {
	w.LastHeartbeat = time.Now()
}

func (w *WorkerNode) SetCpuUsed(usage uint64) {
	w.CpuUsed = usage
}

func (w *WorkerNode) SetMemoryUsed(usage uint64) {
	w.MemoryUsed = usage
}

func (w *WorkerNode) AddUsage(cpu, memory uint64) {
	atomic.AddUint64(&w.CpuUsed, cpu)
	atomic.AddUint64(&w.MemoryUsed, memory)
}

func (w *WorkerNode) GetIP() string {
	return w.IP
}

func (w *WorkerNode) GetPort() string {
	return w.Port
}

func (w *WorkerNode) GetCpuAvailable() uint64 {
	return w.CpuAvailable
}

func (w *WorkerNode) GetCpuUsed() uint64 {
	return w.CpuUsed
}

func (w *WorkerNode) GetMemoryAvailable() uint64 {
	return w.MemoryAvailable
}

func (w *WorkerNode) GetMemoryUsed() uint64 {
	return w.MemoryUsed
}

func (w *WorkerNode) GetEndpointMap() synchronization.SyncStructure[*core.Endpoint, string] {
	return w.endpointMap
}

func (w *WorkerNode) GetCIDR() string {
	return w.CIDR
}

func (w *WorkerNode) AddImage(image string) bool {
	w.imageSet.Lock()
	defer w.imageSet.Unlock()
	if w.imageSet.Present(image) {
		return false
	}
	w.imageSet.Set(image, struct{}{})
	return true
}

func (w *WorkerNode) RemoveImage(image string) bool {
	w.imageSet.Lock()
	defer w.imageSet.Unlock()
	if !w.imageSet.Present(image) {
		return false
	}
	w.imageSet.Remove(image)
	return true
}

func (w *WorkerNode) HasImage(image string) bool {
	w.imageSet.RLock()
	defer w.imageSet.RUnlock()
	return w.imageSet.Present(image)
}

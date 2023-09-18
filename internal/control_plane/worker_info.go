package control_plane

import (
	"cluster_manager/api/proto"
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

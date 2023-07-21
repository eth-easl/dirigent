package api

import (
	"cluster_manager/api/proto"
	"cluster_manager/internal/worker_node"
	"context"
)

type WnApiServer struct {
	proto.UnimplementedWorkerNodeInterfaceServer

	workerNode *worker_node.WorkerNode
}

func NewWorkerNodeApi(workerNode *worker_node.WorkerNode) *WnApiServer {
	return &WnApiServer{
		workerNode: workerNode,
	}
}

func (w *WnApiServer) CreateSandbox(grpcCtx context.Context, in *proto.ServiceInfo) (*proto.SandboxCreationStatus, error) {
	return w.workerNode.CreateSandbox(grpcCtx, in)
}

func (w *WnApiServer) DeleteSandbox(grpcCtx context.Context, in *proto.SandboxID) (*proto.ActionStatus, error) {
	return w.workerNode.DeleteSandbox(grpcCtx, in)
}

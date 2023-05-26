package api

import (
	"cluster_manager/api/proto"
	"context"
)

type WnApiServer struct {
	proto.UnimplementedWorkerNodeInterfaceServer
}

func (w *WnApiServer) CreateSandbox(ctx context.Context, in *proto.SandboxInfo) (*proto.ActionStatus, error) {
	//TODO implement me
	panic("implement me")
}

func (w *WnApiServer) DeleteSandbox(ctx context.Context, in *proto.SandboxInfo) (*proto.ActionStatus, error) {
	//TODO implement me
	panic("implement me")
}

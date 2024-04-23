package sandbox

import (
	"cluster_manager/proto"
	"context"
	"google.golang.org/protobuf/types/known/emptypb"
)

type RuntimeInterface interface {
	CreateSandbox(ctx context.Context, serviceInfo *proto.ServiceInfo) (*proto.SandboxCreationStatus, error)
	DeleteSandbox(ctx context.Context, in *proto.SandboxID) (*proto.ActionStatus, error)
	ListEndpoints(ctx context.Context, _ *emptypb.Empty) (*proto.EndpointsList, error)
	ValidateHostConfig() bool
}

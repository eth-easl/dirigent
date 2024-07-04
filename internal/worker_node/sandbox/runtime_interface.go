package sandbox

import (
	"cluster_manager/proto"
	"context"

	"google.golang.org/protobuf/types/known/emptypb"
)

type RuntimeInterface interface {
	CreateSandbox(ctx context.Context, serviceInfo *proto.ServiceInfo) (*proto.SandboxCreationStatus, error)
	DeleteSandbox(ctx context.Context, in *proto.SandboxID) (*proto.ActionStatus, error)
	CreateTaskSandbox(ctx context.Context, task *proto.WorkflowTaskInfo) (*proto.SandboxCreationStatus, error)
	ListEndpoints(ctx context.Context, _ *emptypb.Empty) (*proto.EndpointsList, error)
	PrepullImage(ctx context.Context, image *proto.ImageInfo) (*proto.ActionStatus, error)
	GetImages(ctx context.Context) ([]*proto.ImageInfo, error)
	ValidateHostConfig() bool
}

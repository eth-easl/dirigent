package empty_worker

import (
	"cluster_manager/proto"
	"context"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type emptyInterfaceClient struct {
}

func NewEmptyInterfaceClient() proto.WorkerNodeInterfaceClient {
	return &emptyInterfaceClient{}
}

func (e emptyInterfaceClient) CreateSandbox(ctx context.Context, in *proto.ServiceInfo, opts ...grpc.CallOption) (*proto.SandboxCreationStatus, error) {
	return &proto.SandboxCreationStatus{Success: true}, nil
}

func (e emptyInterfaceClient) DeleteSandbox(ctx context.Context, in *proto.SandboxID, opts ...grpc.CallOption) (*proto.ActionStatus, error) {
	return &proto.ActionStatus{Success: true}, nil
}

func (e emptyInterfaceClient) CreateTaskSandbox(ctx context.Context, in *proto.WorkflowTaskInfo, opts ...grpc.CallOption) (*proto.SandboxCreationStatus, error) {
	return &proto.SandboxCreationStatus{
		Success: true,
	}, nil
}

func (e emptyInterfaceClient) ReceiveRouteUpdate(ctx context.Context, in *proto.RouteUpdate, opts ...grpc.CallOption) (*proto.ActionStatus, error) {
	return &proto.ActionStatus{Success: true}, nil
}

func (e emptyInterfaceClient) ListEndpoints(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*proto.EndpointsList, error) {
	return &proto.EndpointsList{}, nil
}

func (e emptyInterfaceClient) PrepullImage(ctx context.Context, in *proto.ImageInfo, opts ...grpc.CallOption) (*proto.ActionStatus, error) {
	return &proto.ActionStatus{Success: true}, nil
}

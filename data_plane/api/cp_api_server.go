package api

import (
	"cluster_manager/api/proto"
	"context"
	"google.golang.org/protobuf/types/known/emptypb"
)

type CpApiServer struct {
	proto.UnimplementedCpiInterfaceServer

	DpiInterface proto.DpiInterfaceClient
}

func (c *CpApiServer) ScaleFromZero(ctx context.Context, in *proto.DeploymentName) (*proto.ActionStatus, error) {
	resp, err := c.DpiInterface.UpdateEndpointList(ctx, &proto.DeploymentEndpointPatch{
		Deployment: &proto.DeploymentName{
			Name: in.Name,
		},
		Endpoints: []string{
			"localhost:10000",
		},
	})

	return &proto.ActionStatus{
		Success: resp.Success,
		Message: resp.Message,
	}, err
}

func (c *CpApiServer) PushDataPlaneMetrics(ctx context.Context, in *proto.DeploymentName) (*proto.ActionStatus, error) {
	//TODO implement me
	panic("implement me")
}

func (c *CpApiServer) ListDeployments(_ context.Context, _ *emptypb.Empty) (*proto.DeploymentList, error) {
	return &proto.DeploymentList{
		Service: []string{"/faas.Executor/Execute"},
	}, nil
}

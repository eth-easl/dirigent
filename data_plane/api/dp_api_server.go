package api

import (
	"cluster_manager/api/proto"
	"cluster_manager/common"
	"context"
	"google.golang.org/grpc"
)

type dpApiServer struct {
	proto.UnimplementedDpiInterfaceServer

	deployments *common.Deployments
}

func (api *dpApiServer) AddDeployment(_ context.Context, in *proto.DeploymentName) (*proto.DeploymentUpdateSuccess, error) {
	return &proto.DeploymentUpdateSuccess{
		Success: api.deployments.AddDeployment(in.GetName()),
	}, nil
}

func (api *dpApiServer) UpdateEndpointList(_ context.Context, patch *proto.DeploymentEndpointPatch) (*proto.DeploymentUpdateSuccess, error) {
	deployment := api.deployments.GetDeployment(patch.GetDeployment().GetName())
	if deployment == nil {
		return &proto.DeploymentUpdateSuccess{
			Success: false,
		}, nil
	}

	deployment.SetUpstreamURLs(patch.Endpoints)

	return &proto.DeploymentUpdateSuccess{
		Success: true,
	}, nil
}

func (api *dpApiServer) DeleteDeployment(_ context.Context, name *proto.DeploymentName) (*proto.DeploymentUpdateSuccess, error) {
	return &proto.DeploymentUpdateSuccess{
		Success: api.deployments.DeleteDeployment(name.GetName()),
	}, nil
}

func CreateDataPlaneAPIServer(host string, port string, cache *common.Deployments) {
	common.CreateGRPCServer(host, port, func(sr grpc.ServiceRegistrar) {
		proto.RegisterDpiInterfaceServer(sr, &dpApiServer{
			deployments: cache,
		})
	})
}

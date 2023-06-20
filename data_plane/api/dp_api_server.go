package api

import (
	"cluster_manager/api/proto"
	"cluster_manager/common"
	"context"
)

type DpApiServer struct {
	proto.UnimplementedDpiInterfaceServer

	Deployments *common.Deployments
}

func (api *DpApiServer) AddDeployment(_ context.Context, in *proto.ServiceInfo) (*proto.DeploymentUpdateSuccess, error) {
	return &proto.DeploymentUpdateSuccess{Success: api.Deployments.AddDeployment(in.GetName())}, nil
}

func (api *DpApiServer) UpdateEndpointList(_ context.Context, patch *proto.DeploymentEndpointPatch) (*proto.DeploymentUpdateSuccess, error) {
	deployment, _ := api.Deployments.GetDeployment(patch.GetService().GetName())
	if deployment == nil {
		return &proto.DeploymentUpdateSuccess{Success: false}, nil
	}

	deployment.SetUpstreamURLs(patch.Endpoints)

	return &proto.DeploymentUpdateSuccess{Success: true}, nil
}

func (api *DpApiServer) DeleteDeployment(_ context.Context, name *proto.ServiceInfo) (*proto.DeploymentUpdateSuccess, error) {
	return &proto.DeploymentUpdateSuccess{Success: api.Deployments.DeleteDeployment(name.GetName())}, nil
}

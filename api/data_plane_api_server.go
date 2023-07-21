package api

import (
	"cluster_manager/api/proto"
	"cluster_manager/internal/data_plane"
	"context"
)

type DpApiServer struct {
	proto.UnimplementedDpiInterfaceServer

	dataplane *data_plane.Dataplane
}

func NewDpApiServer(dataplane *data_plane.Dataplane) *DpApiServer {
	return &DpApiServer{
		dataplane: dataplane,
	}
}

func (api *DpApiServer) AddDeployment(_ context.Context, in *proto.ServiceInfo) (*proto.DeploymentUpdateSuccess, error) {
	return api.dataplane.AddDeployment(in)
}

func (api *DpApiServer) UpdateEndpointList(_ context.Context, patch *proto.DeploymentEndpointPatch) (*proto.DeploymentUpdateSuccess, error) {
	return api.dataplane.UpdateEndpointList(patch)
}

func (api *DpApiServer) DeleteDeployment(_ context.Context, name *proto.ServiceInfo) (*proto.DeploymentUpdateSuccess, error) {
	return api.dataplane.DeleteDeployment(name)
}

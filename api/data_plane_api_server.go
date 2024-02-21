package api

import (
	"cluster_manager/api/proto"
	"cluster_manager/internal/data_plane"
	"cluster_manager/internal/data_plane/proxy"
	"context"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/emptypb"
)

type DpApiServer struct {
	proto.UnimplementedDpiInterfaceServer

	dataplane *data_plane.Dataplane
	Proxy     *proxy.ProxyingService
}

func NewDpApiServer(dataPlane *data_plane.Dataplane) *DpApiServer {
	return &DpApiServer{
		dataplane: dataPlane,
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

func (api *DpApiServer) DrainSandbox(_ context.Context, endpoint *proto.DeploymentEndpointPatch) (*proto.DeploymentUpdateSuccess, error) {
	return api.dataplane.DrainSandbox(endpoint)
}

// TODO: Remove this function
func (api *DpApiServer) ResetMeasurements(_ context.Context, in *emptypb.Empty) (*proto.ActionStatus, error) {
	logrus.Warn("This function does nothing")
	return &proto.ActionStatus{Success: true}, nil
}

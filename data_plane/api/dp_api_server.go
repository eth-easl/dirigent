package api

import (
	"cluster_manager/api/proto"
	"cluster_manager/common"
	"context"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"net"
)

type apiServer struct {
	proto.UnimplementedDpiInterfaceServer

	deployments *common.Deployments
}

func (api *apiServer) AddDeployment(_ context.Context, in *proto.DeploymentName) (*proto.DeploymentUpdateSuccess, error) {
	err := api.deployments.AddDeployment(in.GetName())

	return &proto.DeploymentUpdateSuccess{
		Success: err == nil,
	}, nil
}

func (api *apiServer) UpdateEndpointList(_ context.Context, patch *proto.DeploymentEndpointPatch) (*proto.DeploymentUpdateSuccess, error) {
	deployment := api.deployments.GetDeployment(patch.GetDeployment().GetName())
	if deployment == nil {
		return &proto.DeploymentUpdateSuccess{
			Success: false,
		}, nil
	}

	newURLs := patch.Endpoints
	deployment.SetUpstreamURLs(newURLs)

	return &proto.DeploymentUpdateSuccess{
		Success: true,
	}, nil
}

func (api *apiServer) DeleteDeployment(_ context.Context, name *proto.DeploymentName) (*proto.DeploymentUpdateSuccess, error) {
	deploymentName := name.GetName()

	// TODO: add drain here

	return &proto.DeploymentUpdateSuccess{
		Success: api.deployments.DeleteDeployment(deploymentName),
	}, nil
}

func apiServerStartInternal(servingUrl string, deploymentCache *common.Deployments) {
	lis, err := net.Listen("tcp", servingUrl)
	if err != nil {
		logrus.Fatalf("Failed to create data plane API server socket - %s\n", err)
	}

	grpcServer := grpc.NewServer()
	reflection.Register(grpcServer)

	proto.RegisterDpiInterfaceServer(grpcServer, &apiServer{
		deployments: deploymentCache,
	})

	err = grpcServer.Serve(lis)
	if err != nil {
		logrus.Fatalf("Failed to bind the API Server gRPC handler - %s\n", err)
	}
}

func CreateDataPlaneAPIServer(host string, port string, stop chan struct{}, cache *common.Deployments) {
	dpAPIAddress := net.JoinHostPort(host, port)

	apiServerStartInternal(dpAPIAddress, cache)
	<-stop
}

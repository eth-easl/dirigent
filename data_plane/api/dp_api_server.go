package api

import (
	"cluster_manager/api/proto"
	"cluster_manager/common"
	"context"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"k8s.io/apimachinery/pkg/util/wait"
	"net"
	"time"
)

func InitializeDataPlaneConnection() proto.DpiInterfaceClient {
	var conn *grpc.ClientConn

	pollContext, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pollErr := wait.PollUntilContextCancel(pollContext, 5*time.Second, false,
		func(ctx context.Context) (done bool, err error) {
			c, err := common.EstablishConnection(
				ctx,
				net.JoinHostPort(common.DataPlaneHost, common.DataPlaneApiPort),
				common.GetLongLivingConnectionDialOptions()...,
			)
			if err != nil {
				logrus.Warn("Retrying to connect to the data plane in 5 seconds")
			}

			conn = c
			return c != nil, nil
		},
	)
	if pollErr != nil {
		logrus.Fatal("Failed to establish connection with the data plane")
	}

	logrus.Info("Successfully established connection with the data plane")
	return proto.NewDpiInterfaceClient(conn)
}

type dpApiServer struct {
	proto.UnimplementedDpiInterfaceServer

	deployments *common.Deployments
}

func (api *dpApiServer) AddDeployment(_ context.Context, in *proto.ServiceInfo) (*proto.DeploymentUpdateSuccess, error) {
	return &proto.DeploymentUpdateSuccess{Success: api.deployments.AddDeployment(in.GetName())}, nil
}

func (api *dpApiServer) UpdateEndpointList(_ context.Context, patch *proto.DeploymentEndpointPatch) (*proto.DeploymentUpdateSuccess, error) {
	deployment := api.deployments.GetDeployment(patch.GetService().GetName())
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

func (api *dpApiServer) DeleteDeployment(_ context.Context, name *proto.ServiceInfo) (*proto.DeploymentUpdateSuccess, error) {
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

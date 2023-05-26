package main

import (
	"cluster_manager/api"
	"cluster_manager/api/proto"
	"cluster_manager/common"
	"cluster_manager/proxy"
	"context"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"k8s.io/apimachinery/pkg/util/wait"
	"net"
	"time"
)

func syncDeploymentCache(cpApi *proto.CpiInterfaceClient, deployments *common.Deployments) {
	resp, err := (*cpApi).ListDeployments(context.Background(), &emptypb.Empty{})
	if err != nil {
		logrus.Fatal("Initial deployment cache synchronization failed.")
	}

	for i := 0; i < len(resp.Service); i++ {
		deployments.AddDeployment(resp.Service[i])
	}
}

func main() {
	logrus.SetLevel(logrus.DebugLevel)

	cache := common.NewDeploymentList()
	dpCreated := make(chan struct{})

	go api.CreateDataPlaneAPIServer(common.DataPlaneHost, common.DataPlaneApiPort, cache)

	var dpConnection proto.CpiInterfaceClient
	go func() {
		dpConnection = InitializeControlPlaneConnection()
		syncDeploymentCache(&dpConnection, cache)

		dpCreated <- struct{}{}
	}()

	<-dpCreated
	proxy.CreateProxyServer(common.DataPlaneHost, common.DataPlaneProxyPort, cache, &dpConnection)
}

func InitializeControlPlaneConnection() proto.CpiInterfaceClient {
	var conn *grpc.ClientConn

	pollContext, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pollErr := wait.PollUntilContextCancel(pollContext, 5*time.Second, false,
		func(ctx context.Context) (done bool, err error) {
			c, err := common.EstablishConnection(
				ctx,
				net.JoinHostPort(common.ControlPlaneHost, common.ControlPlanePort),
				common.GetLongLivingConnectionDialOptions()...,
			)
			if err != nil {
				logrus.Warn("Retrying to connect to the control plane in 5 seconds")
			}

			conn = c
			return c != nil, nil
		},
	)

	if pollErr != nil {
		logrus.Fatal("Failed to establish connection with the data plane")
	}

	logrus.Info("Successfully established connection with the control plane")

	return proto.NewCpiInterfaceClient(conn)
}

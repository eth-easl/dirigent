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
	"time"
)

func syncDeploymentCache(cpApi *proto.CpiInterfaceClient, deployments *common.Deployments) {
	resp, err := (*cpApi).ListServices(context.Background(), &emptypb.Empty{})
	if err != nil {
		logrus.Fatal("Initial deployment cache synchronization failed.")
	}

	for i := 0; i < len(resp.Service); i++ {
		deployments.AddDeployment(resp.Service[i])
	}
}

func main() {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.TextFormatter{TimestampFormat: time.StampMilli, FullTimestamp: true})

	cache := common.NewDeploymentList()
	dpCreated := make(chan struct{})

	go common.CreateGRPCServer(common.DataPlaneHost, common.DataPlaneApiPort, func(sr grpc.ServiceRegistrar) {
		proto.RegisterDpiInterfaceServer(sr, &api.DpApiServer{
			Deployments: cache,
		})
	})

	var dpConnection proto.CpiInterfaceClient
	go func() {
		dpConnection = common.InitializeControlPlaneConnection()
		syncDeploymentCache(&dpConnection, cache)

		dpCreated <- struct{}{}
	}()

	<-dpCreated
	proxy.CreateProxyServer(common.DataPlaneHost, common.DataPlaneProxyPort, cache, &dpConnection)
}

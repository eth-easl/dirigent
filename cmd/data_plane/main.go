package main

import (
	"cluster_manager/api"
	"cluster_manager/api/proto"
	common "cluster_manager/internal/common"
	"cluster_manager/internal/proxy"
	"context"
	"flag"
	"path"
	"strconv"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

var (
	controlPlaneIP    = flag.String("controlPlaneIP", "localhost", "Control plane IP address")
	controlPlanePort  = flag.String("controlPlanePort", common.DefaultControlPlanePort, "Control plane port")
	portProxy         = flag.String("portProxy", common.DefaultDataPlaneProxyPort, "Data plane incoming traffic port")
	portGRPC          = flag.String("portGRPC", common.DefaultDataPlaneApiPort, "Data plane incoming traffic port")
	verbosity         = flag.String("verbosity", "info", "Logging verbosity - choose from [info, debug, trace]")
	traceOutputFolder = flag.String("traceOutputFolder", common.DefaultTraceOutputFolder, "Folder where to write all logs")
)

func main() {
	flag.Parse()
	common.InitLibraries(*verbosity)

	cache := common.NewDeploymentList()
	dpCreated := make(chan struct{})

	go common.CreateGRPCServer("0.0.0.0", *portGRPC, func(sr grpc.ServiceRegistrar) {
		proto.RegisterDpiInterfaceServer(sr, &api.DpApiServer{
			Deployments: cache,
		})
	})

	var dpConnection proto.CpiInterfaceClient
	go func() {
		grpcPort, _ := strconv.Atoi(*portGRPC)
		proxyPort, _ := strconv.Atoi(*portProxy)

		dpConnection = common.InitializeControlPlaneConnection(*controlPlaneIP, *controlPlanePort, int32(grpcPort), int32(proxyPort))
		syncDeploymentCache(&dpConnection, cache)

		dpCreated <- struct{}{}
	}()

	<-dpCreated

	proxyServer := proxy.NewProxyingService("0.0.0.0", *portProxy, cache, &dpConnection, path.Join(*traceOutputFolder, "proxy_trace.csv"))

	go proxyServer.Tracing.StartTracingService()
	defer close(proxyServer.Tracing.InputChannel)

	proxyServer.StartProxyServer()
}

func syncDeploymentCache(cpApi *proto.CpiInterfaceClient, deployments *common.Deployments) {
	resp, err := (*cpApi).ListServices(context.Background(), &emptypb.Empty{})
	if err != nil {
		logrus.Fatal("Initial deployment cache synchronization failed.")
	}

	for i := 0; i < len(resp.Service); i++ {
		deployments.AddDeployment(resp.Service[i])
	}
}

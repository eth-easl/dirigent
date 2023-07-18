package main

import (
	"cluster_manager/api"
	"cluster_manager/api/proto"
	common "cluster_manager/internal/common"
	"cluster_manager/internal/proxy"
	"cluster_manager/internal/proxy/load_balancing"
	config2 "cluster_manager/pkg/config"
	"cluster_manager/pkg/logger"
	"context"
	"os/signal"
	"path"
	"strconv"
	"syscall"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

func parseLoadBalancingPolicy(dataPlaneConfig config2.DataPlaneConfig) load_balancing.LoadBalancingPolicy {
	switch dataPlaneConfig.LoadBalancingPolicy {
	case "random":
		return load_balancing.LOAD_BALANCING_RANDOM
	case "round-robin":
		return load_balancing.LOAD_BALANCING_ROUND_ROBIN
	case "least-processed":
		return load_balancing.LOAD_BALANCING_LEAST_PROCESSED
	case "knative":
		return load_balancing.LOAD_BALANCING_KNATIVE
	default:
		logrus.Error("Failed to parse policy, default policy is random")
		return load_balancing.LOAD_BALANCING_RANDOM
	}
}

func main() {
	config, err := config2.ReadDataPlaneConfiguration("cmd/data_plane/config.yaml")
	if err != nil {
		logrus.Fatal("Failed to read configuration file (error : %s)", err.Error())
	}

	logger.SetupLogger(config.Verbosity)

	cache := common.NewDeploymentList()
	dpCreated := make(chan struct{})

	go common.CreateGRPCServer("0.0.0.0", config.PortGRPC, func(sr grpc.ServiceRegistrar) {
		proto.RegisterDpiInterfaceServer(sr, &api.DpApiServer{
			Deployments: cache,
		})
	})

	var dpConnection proto.CpiInterfaceClient

	grpcPort, _ := strconv.Atoi(config.PortGRPC)
	proxyPort, _ := strconv.Atoi(config.PortProxy)

	go func() {
		dpConnection = common.InitializeControlPlaneConnection(config.ControlPlaneIp, config.ControlPlanePort, int32(grpcPort), int32(proxyPort))
		syncDeploymentCache(&dpConnection, cache)

		dpCreated <- struct{}{}
	}()

	<-dpCreated

	var loadBalancingPolicy load_balancing.LoadBalancingPolicy = parseLoadBalancingPolicy(config)

	proxyServer := proxy.NewProxyingService("0.0.0.0", config.PortProxy, cache, &dpConnection, path.Join(config.TraceOutputFolder, "proxy_trace.csv"), loadBalancingPolicy)

	go proxyServer.Tracing.StartTracingService()
	defer close(proxyServer.Tracing.InputChannel)
	go proxyServer.StartProxyServer()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	defer common.DeregisterControlPlaneConnection(config.ControlPlaneIp, config.ControlPlanePort, int32(grpcPort), int32(proxyPort))

	select {
	case <-ctx.Done():
		logrus.Info("Received interruption signal, try to gracefully stop")
	}
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

func deregisterNode() {

}

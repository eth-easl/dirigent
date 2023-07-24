package main

import (
	"cluster_manager/api"
	"cluster_manager/api/proto"
	"cluster_manager/internal/data_plane"
	"cluster_manager/internal/data_plane/function_metadata"
	"cluster_manager/pkg/config"
	"cluster_manager/pkg/grpc_helpers"
	"cluster_manager/pkg/logger"
	"context"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"os/signal"
	"syscall"
)

func main() {
	config, err := config.ReadDataPlaneConfiguration("cmd/data_plane/config.yaml")
	if err != nil {
		logrus.Fatalf("Failed to read configuration file (error : %s)", err.Error())
	}

	logger.SetupLogger(config.Verbosity)

	cache := function_metadata.NewDeploymentList()
	dataPlane := data_plane.NewDataplane(config, cache)

	go grpc_helpers.CreateGRPCServer("0.0.0.0", config.PortGRPC, func(sr grpc.ServiceRegistrar) {
		proto.RegisterDpiInterfaceServer(sr, api.NewDpApiServer(dataPlane))
	})

	proxyServer := dataPlane.GetProxyServer()

	go proxyServer.Tracing.StartTracingService()
	defer close(proxyServer.Tracing.InputChannel)

	go proxyServer.StartProxyServer()

	defer dataPlane.DeregisterControlPlaneConnection()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	select {
	case <-ctx.Done():
		logrus.Info("Received interruption signal, try to gracefully stop")
	}
}

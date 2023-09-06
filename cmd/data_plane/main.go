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
	"flag"
	"os/signal"
	"syscall"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

var (
	configPath = flag.String("configPath", "cmd/data_plane/config.yaml", "Path to the configuration file")
)

func main() {
	flag.Parse()

	logrus.Debugf("Configuration path is : %s", *configPath)

	cfg, err := config.ReadDataPlaneConfiguration(*configPath)
	if err != nil {
		logrus.Fatalf("Failed to read configuration file (error : %s)", err.Error())
	}

	logger.SetupLogger(cfg.Verbosity)

	cache := function_metadata.NewDeploymentList()
	dataPlane := data_plane.NewDataplane(cfg, cache)

	apiServer := api.NewDpApiServer(dataPlane)

	go grpc_helpers.CreateGRPCServer("0.0.0.0", cfg.PortGRPC, func(sr grpc.ServiceRegistrar) {
		proto.RegisterDpiInterfaceServer(sr, apiServer)
	})

	proxyServer, err := dataPlane.GetProxyServer()
	if err != nil {
		logrus.Fatalf("Failed to start proxy server (error : %s)", err.Error())
	}

	apiServer.Proxy = proxyServer

	go proxyServer.Tracing.StartTracingService()
	defer close(proxyServer.Tracing.InputChannel)

	go dataPlane.SetupHeartbeatLoop()
	go proxyServer.StartProxyServer()

	defer dataPlane.DeregisterControlPlaneConnection()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	select {
	case <-ctx.Done():
		logrus.Info("Received interruption signal, try to gracefully stop")
	}
}

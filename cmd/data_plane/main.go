package main

import (
	"cluster_manager/api"
	"cluster_manager/api/proto"
	"cluster_manager/internal/data_plane"
	"cluster_manager/internal/data_plane/function_metadata"
	"cluster_manager/pkg/config"
	"cluster_manager/pkg/grpc_helpers"
	"cluster_manager/pkg/logger"
	"cluster_manager/pkg/network"
	"context"
	"flag"
	"os/signal"
	"syscall"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

var (
	configPath = flag.String("config", "cmd/data_plane/config.yaml", "Path to the configuration file")
)

func main() {
	flag.Parse()

	logrus.Debugf("Configuration path is : %s", *configPath)

	cfg, err := config.ReadDataPlaneConfiguration(*configPath)
	if err != nil {
		logrus.Fatalf("Failed to read configuration file (error : %s)", err.Error())
	}

	logger.SetupLogger(cfg.Verbosity)

	if cfg.DataPlaneIp == "dynamic" {
		cfg.DataPlaneIp = network.GetLocalIP()
	}

	cache := function_metadata.NewDeploymentList()
	dataPlane := data_plane.NewDataplane(cfg, cache)

	apiServer := api.NewDpApiServer(dataPlane)

	go grpc_helpers.CreateGRPCServer(cfg.PortGRPC, func(sr grpc.ServiceRegistrar) {
		proto.RegisterDpiInterfaceServer(sr, apiServer)
	})

	proxyServer, err := dataPlane.GetProxyServer(cfg.Async)
	if err != nil {
		logrus.Fatalf("Failed to start proxy server (error : %s)", err.Error())
	}

	go proxyServer.StartTracingService()
	go proxyServer.StartProxyServer()

	go dataPlane.SetupHeartbeatLoop(proxyServer)
	defer dataPlane.DeregisterControlPlaneConnection()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	select {
	case <-ctx.Done():
		logrus.Info("Received interruption signal, try to gracefully stop")
	}
}

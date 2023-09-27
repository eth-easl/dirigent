package main

import (
	"cluster_manager/api"
	"cluster_manager/api/proto"
	"cluster_manager/internal/worker_node"
	"cluster_manager/pkg/config"
	"cluster_manager/pkg/grpc_helpers"
	"cluster_manager/pkg/logger"
	"cluster_manager/pkg/utils"
	"context"
	"flag"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

var (
	configPath = flag.String("configPath", "cmd/worker_node/config.yaml", "Path to the configuration file")
)

func main() {
	flag.Parse()

	logrus.Debugf("Configuration path is : %s", *configPath)

	cfg, err := config.ReadWorkedNodeConfiguration(*configPath)
	if err != nil {
		logrus.Fatalf("Failed to read configuration file (error : %s)", err.Error())
	}

	logger.SetupLogger(cfg.Verbosity)

	cpApi, err := grpc_helpers.InitializeControlPlaneConnection(cfg.ControlPlaneIp, cfg.ControlPlanePort, -1, -1)
	if err != nil {
		logrus.Fatalf("Failed to initialize control plane connection (error : %s)", err.Error())
	}

	workerNode := worker_node.NewWorkerNode(cpApi, cfg)

	workerNode.RegisterNodeWithControlPlane(cfg, &cpApi)
	defer workerNode.DeregisterNodeFromControlPlane(cfg, &cpApi)

	go workerNode.SetupHeartbeatLoop(&cpApi)

	logrus.Info("Starting API handlers")

	go grpc_helpers.CreateGRPCServer(utils.DockerLocalhost, strconv.Itoa(cfg.Port), func(sr grpc.ServiceRegistrar) {
		proto.RegisterWorkerNodeInterfaceServer(sr, api.NewWorkerNodeApi(workerNode))
	})

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	select {
	case <-ctx.Done():
		logrus.Info("Received interruption signal, try to gracefully stop")
	}
}

package main

import (
	"cluster_manager/api"
	"cluster_manager/api/proto"
	"cluster_manager/internal/worker_node"
	"cluster_manager/internal/worker_node/sandbox"
	"cluster_manager/pkg/config"
	"cluster_manager/pkg/grpc_helpers"
	"cluster_manager/pkg/logger"
	"cluster_manager/pkg/utils"
	"context"
	"flag"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"os/signal"
	"strconv"
	"syscall"
)

var (
	configPath = flag.String("configPath", "config.yaml", "Path to the configuration file")
)

func main() {
	config, err := config.ReadWorkedNodeConfiguration("cmd/worker_node/config.yaml")
	if err != nil {
		logrus.Fatalf("Failed to read configuration file (error : %s)", err.Error())
	}

	logger.SetupLogger(config.Verbosity)

	containerdClient := sandbox.GetContainerdClient(config.CRIPath)
	defer containerdClient.Close()

	cpApi := grpc_helpers.InitializeControlPlaneConnection(config.ControlPlaneIp, config.ControlPlanePort, -1, -1)

	workerNode := worker_node.NewWorkerNode(config, containerdClient)

	workerNode.RegisterNodeWithControlPlane(config, &cpApi)
	defer workerNode.DeregisterNodeFromControlPlane(config, &cpApi)

	go workerNode.SetupHeartbeatLoop(&cpApi)

	logrus.Info("Starting API handlers")

	go grpc_helpers.CreateGRPCServer(utils.DockerLocalhost, strconv.Itoa(config.Port), func(sr grpc.ServiceRegistrar) {
		proto.RegisterWorkerNodeInterfaceServer(sr, api.NewWorkerNodeApi(workerNode))
	})

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	select {
	case <-ctx.Done():
		logrus.Info("Received interruption signal, try to gracefully stop")
	}
}

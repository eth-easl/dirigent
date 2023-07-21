package main

import (
	"cluster_manager/api"
	"cluster_manager/api/proto"
	"cluster_manager/internal/common"
	"cluster_manager/internal/worker_node"
	"cluster_manager/internal/worker_node/sandbox"
	"cluster_manager/pkg/config"
	"cluster_manager/pkg/logger"
	"cluster_manager/pkg/utils"
	"context"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"os/signal"
	"strconv"
	"syscall"
)

func main() {
	config, err := config.ReadWorkedNodeConfiguration("cmd/worker_node/config.yaml")
	if err != nil {
		logrus.Fatal("Failed to read configuration file (error : %s)", err.Error())
	}

	logger.SetupLogger(config.Verbosity)

	containerdClient := sandbox.GetContainerdClient(config.CRIPath)
	defer containerdClient.Close()

	cpApi := common.InitializeControlPlaneConnection(config.ControlPlaneIp, config.ControlPlanePort, -1, -1)

	workerNode := worker_node.NewWorkerNode(config, containerdClient)

	workerNode.RegisterNodeWithControlPlane(config, &cpApi)
	defer workerNode.DeregisterNodeFromControlPlane(config, &cpApi)

	go workerNode.SetupHeartbeatLoop(&cpApi)

	logrus.Info("Starting API handlers")

	go common.CreateGRPCServer(utils.DockerLocalhost, strconv.Itoa(config.Port), func(sr grpc.ServiceRegistrar) {
		proto.RegisterWorkerNodeInterfaceServer(sr, api.NewWorkerNodeApi(workerNode))
	})

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	select {
	case <-ctx.Done():
		logrus.Info("Received interruption signal, try to gracefully stop")
	}
}

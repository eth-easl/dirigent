package main

import (
	"cluster_manager/api"
	"cluster_manager/api/proto"
	"cluster_manager/internal/worker_node"
	"cluster_manager/internal/worker_node/sandbox/firecracker"
	"cluster_manager/pkg/config"
	"cluster_manager/pkg/grpc_helpers"
	"cluster_manager/pkg/logger"
	"cluster_manager/pkg/network"
	"context"
	"flag"
	"os/exec"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

var (
	configPath = flag.String("config", "cmd/worker_node/config.yaml", "Path to the configuration file")
)

func main() {
	flag.Parse()

	logrus.Debugf("Configuration path is : %s", *configPath)

	cfg, err := config.ReadWorkedNodeConfiguration(*configPath)
	if err != nil {
		logrus.Fatalf("Failed to read configuration file (error : %s)", err.Error())
	}

	logger.SetupLogger(cfg.Verbosity)

	if cfg.WorkerNodeIP == "dynamic" {
		cfg.WorkerNodeIP = network.GetLocalIP()
	}

	cpApi, err := grpc_helpers.NewControlPlaneConnection(cfg.ControlPlaneAddress)
	if err != nil {
		logrus.Fatalf("Failed to initialize control plane connection (error : %s)", err.Error())
	}

	workerNode := worker_node.NewWorkerNode(cpApi, cfg)

	workerNode.RegisterNodeWithControlPlane(cfg, &cpApi)
	defer workerNode.DeregisterNodeFromControlPlane(cfg, &cpApi)

	go workerNode.SetupHeartbeatLoop(&cfg)
	defer resetIPTables()

	logrus.Info("Starting API handlers")

	go grpc_helpers.CreateGRPCServer(strconv.Itoa(cfg.Port), func(sr grpc.ServiceRegistrar) {
		proto.RegisterWorkerNodeInterfaceServer(sr, api.NewWorkerNodeApi(workerNode))
	})

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	select {
	case <-ctx.Done():
		logrus.Info("Received interruption signal, try to gracefully stop")

		firecracker.DeleteAllSnapshots()
		err = firecracker.DeleteUnusedNetworkDevices()
		if err != nil {
			logrus.Warn("Interruption received, but failed to delete leftover network devices.")
		}
	}
}

func resetIPTables() {
	logrus.Warn(exec.Command("sudo", "iptables", "-t", "nat", "-F").Err.Error())
}

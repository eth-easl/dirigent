package main

import (
	"cluster_manager/internal/worker_node"
	"cluster_manager/internal/worker_node/sandbox/firecracker"
	"cluster_manager/pkg/config"
	"cluster_manager/pkg/grpc_helpers"
	"cluster_manager/pkg/logger"
	"cluster_manager/pkg/network"
	"cluster_manager/pkg/utils"
	"cluster_manager/proto"
	"flag"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"os/exec"
	"strconv"
)

var (
	configPath = flag.String("config", "cmd/worker_node/config.yaml", "Path to the configuration file")
)

func main() {
	flag.Parse()

	if !utils.IsRoot() {
		logrus.Fatalf("Worker node must be started with sudo")
	}

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

	go grpc_helpers.CreateGRPCServer(strconv.Itoa(cfg.Port), func(sr grpc.ServiceRegistrar) {
		proto.RegisterWorkerNodeInterfaceServer(sr, workerNode)
	})

	workerNode.RegisterNodeWithControlPlane(cfg, &cpApi)
	defer workerNode.DeregisterNodeFromControlPlane(cfg, &cpApi)

	go workerNode.SetupHeartbeatLoop(&cfg)
	defer resetIPTables()

	logrus.Info("Starting API handlers")

	utils.WaitTerminationSignal(func() {
		firecracker.DeleteAllSnapshots()
		err := firecracker.DeleteUnusedNetworkDevices()
		if err != nil {
			logrus.Warn("Interruption received, but failed to delete leftover network devices.")
		}
	})
}

func resetIPTables() {
	err := exec.Command("sudo", "iptables", "-t", "nat", "-FileSystem").Run()
	if err != nil {
		logrus.Errorf("Error reseting IP tables - %v", err.Error())
	}
}

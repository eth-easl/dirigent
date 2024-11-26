/*
 * MIT License
 *
 * Copyright (c) 2024 EASL
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */

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
	"cluster_manager/pkg/utils"
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

	workerNode.RegisterNodeWithControlPlane(cfg, &cpApi)
	defer workerNode.DeregisterNodeFromControlPlane(cfg, &cpApi)

	go workerNode.SetupHeartbeatLoop(&cfg)
	defer resetIPTables()

	logrus.Info("Starting API handlers")

	go grpc_helpers.CreateGRPCServer(strconv.Itoa(cfg.Port), func(sr grpc.ServiceRegistrar) {
		proto.RegisterWorkerNodeInterfaceServer(sr, api.NewWorkerNodeApi(workerNode))
	})

	utils.WaitTerminationSignal(func() {
		firecracker.DeleteAllSnapshots()
		err := firecracker.DeleteUnusedNetworkDevices()
		if err != nil {
			logrus.Warn("Interruption received, but failed to delete leftover network devices.")
		}
	})
}

func resetIPTables() {
	err := exec.Command("sudo", "iptables", "-t", "nat", "-F").Run()
	if err != nil {
		logrus.Errorf("Error reseting IP tables - %v", err.Error())
	}
}

package main

import (
	"cluster_manager/internal/data_plane"
	"cluster_manager/internal/data_plane/function_metadata"
	"cluster_manager/internal/worker_node"
	"cluster_manager/pkg/config"
	"cluster_manager/pkg/grpc_helpers"
	"cluster_manager/pkg/logger"
	"context"
	"flag"
	"github.com/sirupsen/logrus"
	"math/rand"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"
)

var (
	nbWorkers  = flag.Int("workers", 25, "Number of fake workers for the simulation")
	dataPlanes = flag.Int("dataplanes", 25, "Number of fake dataplanes for the simulation")
)

func main() {
	flag.Parse()

	// TODO: Finde better way for the simulation
	// Simulate dataplanes
	{
		cfg := config.DataPlaneConfig{
			DataPlaneIp:         "127.0.0.1",
			ControlPlaneIp:      "localhost",
			ControlPlanePort:    "9090",
			PortProxy:           "8080",
			PortGRPC:            "8081",
			Verbosity:           "trace",
			TraceOutputFolder:   "data",
			LoadBalancingPolicy: "random",
		}

		wg := sync.WaitGroup{}
		wg.Add(*nbWorkers)

		dataplanes := make([]*data_plane.Dataplane, 0)

		for i := 0; i < *nbWorkers; i++ {
			cache := function_metadata.NewDeploymentList()
			dataPlane := data_plane.NewDataplane(cfg, cache)
			dataplanes = append(dataplanes, dataPlane)
		}

		for i := 0; i < *nbWorkers; i++ {
			go func(idx int) {
				time.Sleep(time.Duration(rand.Intn(1000)) * time.Millisecond)
				go dataplanes[idx].SetupHeartbeatLoop()
				wg.Done()
			}(i)
		}

		for i := 0; i < *nbWorkers; i++ {
			defer dataplanes[i].DeregisterControlPlaneConnection()
		}

		wg.Wait()
	}

	// Simulate nodes
	{
		cfg := config.WorkerNodeConfig{
			WorkerNodeIP:           "127.0.0.1",
			ControlPlaneIp:         "localhost",
			ControlPlanePort:       "9090",
			Port:                   10010,
			Verbosity:              "trace",
			CRIType:                "containerd",
			CRIPath:                "/run/containerd/containerd.sock",
			CNIConfigPath:          "configs/cni.conf",
			PrefetchImage:          false,
			FirecrackerKernel:      "",
			FirecrackerFileSystem:  "",
			FirecrackerIPPrefix:    "",
			FirecrackerVMDebugMode: false,
		}

		logger.SetupLogger(cfg.Verbosity)

		cpApi, err := grpc_helpers.InitializeControlPlaneConnection(cfg.ControlPlaneIp, cfg.ControlPlanePort, "", -1, -1)
		if err != nil {
			logrus.Fatalf("Failed to initialize control plane connection (error : %s)", err.Error())
		}

		wg := sync.WaitGroup{}
		wg.Add(*nbWorkers)

		workers := make([]*worker_node.WorkerNode, 0)

		for i := 0; i < *nbWorkers; i++ {
			workers = append(workers, worker_node.NewWorkerNode(cpApi, cfg, "mockWorker:"+strconv.Itoa(i)))
		}

		for i := 0; i < *nbWorkers; i++ {
			go func(idx int) {
				workers[idx].RegisterNodeWithControlPlane(cfg, &cpApi)

				time.Sleep(time.Duration(rand.Intn(1000)) * time.Millisecond * 5)

				go workers[idx].SetupHeartbeatLoop(&cpApi)
				wg.Done()
			}(i)
		}

		for i := 0; i < *nbWorkers; i++ {
			defer workers[i].DeregisterNodeFromControlPlane(cfg, &cpApi)
		}

		wg.Wait()
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	select {
	case <-ctx.Done():
		logrus.Info("Received interruption signal, try to gracefully stop")
	}
}

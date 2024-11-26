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
	"cluster_manager/internal/data_plane"
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
	nbWorkers  = flag.Int("workers", 100, "Number of fake workers for the simulation")
	dataPlanes = flag.Int("dataplanes", 25, "Number of fake dataplanes for the simulation")
)

func main() {
	flag.Parse()

	// TODO: Find a better way for the simulation
	// Simulate data planes
	{
		cfg := config.DataPlaneConfig{
			DataPlaneIp:         "127.0.0.1",
			ControlPlaneAddress: []string{"localhost:9090"},
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
			dataPlane := data_plane.NewDataplane(cfg)
			dataplanes = append(dataplanes, dataPlane)
		}

		for i := 0; i < *nbWorkers; i++ {
			go func(idx int) {
				time.Sleep(time.Duration(rand.Intn(1000)) * time.Millisecond)
				go dataplanes[idx].SetupHeartbeatLoop(nil)
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
			ControlPlaneAddress:    []string{"localhost:9090"},
			Port:                   10010,
			Verbosity:              "trace",
			CRIType:                "containerd",
			CRIPath:                "/run/containerd/containerd.sock",
			CNIConfigPath:          "configs/cni.conf",
			PrefetchImage:          false,
			FirecrackerKernel:      "",
			FirecrackerFileSystem:  "",
			FirecrackerVMDebugMode: false,
		}

		logger.SetupLogger(cfg.Verbosity)

		cpApi, err := grpc_helpers.NewControlPlaneConnection(cfg.ControlPlaneAddress)
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

				go workers[idx].SetupHeartbeatLoop(&cfg)
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

package main

import (
	proto2 "cluster_manager/api/proto"
	common "cluster_manager/pkg/grpc_helpers"
	"cluster_manager/pkg/utils"
	"cluster_manager/tests/shared"
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"sync"
	"time"
)

func main() {
	nbServices := 1
	DeployWorkers(5, 0)
	shared.DeployServiceMultiThread(nbServices, 0)

	for j := 0; j < 3; j++ {
		shared.FireColdstartMultiThread(nbServices, 0, 1)
		time.Sleep(1 * time.Second)
		shared.FireColdstartMultiThread(nbServices, 0, 0)
		time.Sleep(time.Second)
	}

}

// TODO: Remove duplicate code
func DeployWorkers(nbDeploys, offset int) {
	logrus.SetLevel(logrus.ErrorLevel)
	logrus.SetFormatter(&logrus.TextFormatter{TimestampFormat: time.StampMilli, FullTimestamp: true})

	cpApi, err := common.InitializeControlPlaneConnection("localhost", utils.DefaultControlPlanePort, "", -1, -1)
	if err != nil {
		logrus.Fatalf("Failed to start control plane connection (error %s)", err.Error())
	}

	ctx, cancel := context.WithTimeout(context.Background(), utils.GRPCFunctionTimeout)
	defer cancel()

	wg := sync.WaitGroup{}
	wg.Add(nbDeploys)

	for i := 0; i < nbDeploys; i++ {
		go func(i int, offset int) {
			id := fmt.Sprintf("%s %d %d", "mockIp", i, offset)
			resp, err := cpApi.RegisterNode(ctx, &proto2.NodeInfo{
				NodeID:     id, // Unique id while registering
				IP:         id,
				Port:       0,
				CpuCores:   0,
				MemorySize: 0,
			})

			if err != nil || !resp.Success {
				errText := "response is not successful"
				if err != nil {
					errText = err.Error()
				}
				logrus.Errorf("Failed to deploy worker : (error %s)", errText)
			}

			wg.Done()
		}(i, offset)
	}

	wg.Wait()
}

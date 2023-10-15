package main

import (
	proto2 "cluster_manager/api/proto"
	common "cluster_manager/pkg/grpc_helpers"
	"cluster_manager/pkg/utils"
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"sync"
	"time"
)

const (
	ControlPlaneAddress string = "localhost"
	MockIp              string = "mockIP"
)

func main() {

	times := 0

	for times < 1 {
		counter := 0
		nbWorkers := 1
		// Measures
		for nbWorkers <= 10000 {
			start := time.Now()

			DeployDataplanes(nbWorkers, counter)

			delta := time.Since(start)
			fmt.Printf("%d,", delta)

			counter += nbWorkers
			nbWorkers *= 10
		}
		fmt.Println()

		times++
	}

}

func DeployDataplanes(nbDeploys, offset int) {
	logrus.SetLevel(logrus.ErrorLevel)
	logrus.SetFormatter(&logrus.TextFormatter{TimestampFormat: time.StampMilli, FullTimestamp: true})

	cpApi, err := common.InitializeControlPlaneConnection(ControlPlaneAddress, utils.DefaultControlPlanePort, "", -1, -1)
	if err != nil {
		logrus.Fatalf("Failed to start control plane connection (error %s)", err.Error())
	}

	ctx, cancel := context.WithTimeout(context.Background(), utils.GRPCFunctionTimeout)
	defer cancel()

	wg := sync.WaitGroup{}
	wg.Add(nbDeploys)

	for i := 0; i < nbDeploys; i++ {
		go func(i int, offset int) {
			id := fmt.Sprintf("%s %d %d", MockIp, i, offset)
			resp, err := cpApi.RegisterDataplane(ctx, &proto2.DataplaneInfo{
				IP:        id,
				APIPort:   0,
				ProxyPort: 0,
			})

			if err != nil || !resp.Success {
				errText := ""
				if err != nil {
					errText = err.Error()
				}
				logrus.Errorf("Failed to deploy dataplane : (error %s)", errText)
			}

			wg.Done()
		}(i, offset)
	}

	wg.Wait()
}

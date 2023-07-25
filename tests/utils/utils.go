package utils

import (
	proto2 "cluster_manager/api/proto"
	"cluster_manager/internal/control_plane/autoscaling"
	common "cluster_manager/pkg/grpc_helpers"
	"cluster_manager/pkg/utils"
	"cluster_manager/tests/proto"
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"sync"
	"testing"
	"time"
)

const (
	deployedFunctionName string = "/faas.Executor/Execute"
	controlPlaneAddress  string = "10.0.1.2"
	dataPlaneAddress     string = "10.0.1.2"
)

func DeployService(t *testing.T, nbDeploys, offset int) {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.TextFormatter{TimestampFormat: time.StampMilli, FullTimestamp: true})

	cpApi := common.InitializeControlPlaneConnection(controlPlaneAddress, utils.DefaultControlPlanePort, -1, -1)

	ctx, cancel := context.WithTimeout(context.Background(), common.GRPCFunctionTimeout)
	defer cancel()

	autoscalingConfig := autoscaling.NewDefaultAutoscalingMetadata()
	autoscalingConfig.ScalingUpperBound = 1
	//autoscalingConfig.ScalingLowerBound = 1

	for i := 0; i < nbDeploys; i++ {
		resp, err := cpApi.RegisterService(ctx, &proto2.ServiceInfo{
			Name:  fmt.Sprintf("%s&%d", deployedFunctionName, i+offset),
			Image: "docker.io/cvetkovic/empty_function:latest",
			PortForwarding: &proto2.PortMapping{
				GuestPort: 80,
				Protocol:  proto2.L4Protocol_TCP,
			},
			AutoscalingConfig: autoscalingConfig,
		})

		if err != nil || !resp.Success {
			t.Error("Failed to deploy service")
			panic("Failed to deploy service")
		}
	}
}

func PerformXInvocations(t *testing.T, nbInvocations, offset int) {
	wg := sync.WaitGroup{}
	wg.Add(nbInvocations)

	for i := 0; i < nbInvocations; i++ {
		go func(i int) {
			conn := common.EstablishGRPCConnectionPoll(dataPlaneAddress, "8080", grpc.WithAuthority(fmt.Sprintf("%s&%d", deployedFunctionName, i+offset)))
			if conn == nil {
				logrus.Fatal("Failed to establish gRPC connection with the data plane")
			}

			logrus.Info("Connection with the gRPC server has been established")

			executorClient := proto.NewExecutorClient(conn)
			err := FireInvocation(executorClient)

			if err != nil {
				t.Error("Invocation failed - ", err)
			}

			wg.Done()
		}(i)
	}

	wg.Wait()
}

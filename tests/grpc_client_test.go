package tests

import (
	proto2 "cluster_manager/api/proto"
	common "cluster_manager/internal/common"
	"cluster_manager/internal/control_plane"
	"cluster_manager/pkg/utils"
	"cluster_manager/tests/proto"
	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

const (
	deployedFunctionName = "/faas.Executor/Execute"
)

func TestDeployService(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.TextFormatter{TimestampFormat: time.StampMilli, FullTimestamp: true})

	cpApi := common.InitializeControlPlaneConnection("localhost", utils.DefaultControlPlanePort, -1, -1)

	ctx, cancel := context.WithTimeout(context.Background(), common.GRPCFunctionTimeout)
	defer cancel()

	autoscalingConfig := control_plane.NewDefaultAutoscalingMetadata()
	autoscalingConfig.ScalingUpperBound = 1
	//autoscalingConfig.ScalingLowerBound = 1

	resp, err := cpApi.RegisterService(ctx, &proto2.ServiceInfo{
		Name:  deployedFunctionName,
		Image: "docker.io/cvetkovic/empty_function:latest",
		PortForwarding: &proto2.PortMapping{
			GuestPort: 80,
			Protocol:  proto2.L4Protocol_TCP,
		},
		AutoscalingConfig: autoscalingConfig,
	})

	if err != nil || !resp.Success {
		t.Error("Failed to deploy service")
	}
}

func TestInvocationProxying(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.TextFormatter{TimestampFormat: time.StampMilli, FullTimestamp: true})

	conn := common.EstablishGRPCConnectionPoll("localhost", "8080", grpc.WithAuthority(deployedFunctionName))
	if conn == nil {
		logrus.Fatal("Failed to establish gRPC connection with the data plane")
	}

	logrus.Info("Connection with the gRPC server has been established")

	executorClient := proto.NewExecutorClient(conn)
	err := FireInvocation(executorClient)

	if err != nil {
		t.Error("Invocation failed - ", err)
	}
}

func Test_100Invocations(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.TextFormatter{TimestampFormat: time.StampMilli, FullTimestamp: true})

	conn := common.EstablishGRPCConnectionPoll("localhost", "8080", grpc.WithAuthority(deployedFunctionName))
	if conn == nil {
		logrus.Fatal("Failed to establish gRPC connection with the data plane")
	}

	logrus.Info("Connection with the gRPC server has been established")

	invocationCount := 100
	ch := make(chan struct{})

	executorClient := proto.NewExecutorClient(conn)

	for i := 0; i < invocationCount; i++ {
		go func() {
			err := FireInvocation(executorClient)
			if err != nil {
				t.Error("Invocation failed - ", err)
			}

			ch <- struct{}{}
		}()
	}

	for i := 0; i < invocationCount; i++ {
		<-ch
	}
}

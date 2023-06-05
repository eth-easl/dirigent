package tests

import (
	"cluster_manager/api"
	proto2 "cluster_manager/api/proto"
	"cluster_manager/common"
	"context"
	"github.com/sirupsen/logrus"
	"testing"
)

func TestDeployService(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)

	cpApi := common.InitializeControlPlaneConnection()

	ctx, cancel := context.WithTimeout(context.Background(), common.GRPCFunctionTimeout)
	defer cancel()

	resp, err := cpApi.RegisterService(ctx, &proto2.ServiceInfo{
		Name:  "/faas.Executor/Execute",
		Image: "docker.io/cvetkovic/empty_function:latest",
		PortForwarding: &proto2.PortMapping{
			GuestPort: 80,
			Protocol:  proto2.L4Protocol_TCP,
		},
		AutoscalingConfig: api.NewDefaultAutoscalingMetadata(),
	})

	if err != nil || !resp.Success {
		t.Error("Failed to deploy service")
	}
}

func TestInvocationProxying(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)

	conn := common.EstablishGRPCConnectionPoll("localhost", "8080")
	if conn == nil {
		logrus.Fatal("Failed to establish gRPC connection with the data plane")
	}

	logrus.Info("Connection with the gRPC server has been established")

	err := FireInvocation(conn)
	if err != nil {
		t.Error("Invocation failed.")
	}
}

func Test_100Invocations(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)

	conn := common.EstablishGRPCConnectionPoll("localhost", "8080")
	if conn == nil {
		logrus.Fatal("Failed to establish gRPC connection with the data plane")
	}

	logrus.Info("Connection with the gRPC server has been established")

	invocationCount := 100
	ch := make(chan struct{})

	for i := 0; i < invocationCount; i++ {
		go func() {
			err := FireInvocation(conn)
			if err != nil {
				t.Error("Invocation failed.")
			}

			ch <- struct{}{}
		}()
	}

	for i := 0; i < invocationCount; i++ {
		<-ch
	}
}

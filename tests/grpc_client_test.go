package tests

import (
	proto2 "cluster_manager/api/proto"
	"cluster_manager/internal/common"
	"cluster_manager/internal/control_plane"
	"cluster_manager/pkg/utils"
	"cluster_manager/tests/proto"
	"context"
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

const (
	deployedFunctionName = "/faas.Executor/Execute"
)

func deploySingleService(t *testing.T, cpApi proto2.CpiInterfaceClient, autoscalingConfig *proto2.AutoscalingConfiguration, name string) {
	ctx, cancel := context.WithTimeout(context.Background(), common.GRPCFunctionTimeout)
	defer cancel()

	logrus.Info("Starting deploying a service in the database")

	resp, err := cpApi.RegisterService(ctx, &proto2.ServiceInfo{
		Name:  name,
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

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randSeq(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	fmt.Println(string(b))
	return string(b)
}

func deployXservices(t *testing.T, x int) {
	cpApi := common.InitializeControlPlaneConnection("localhost", utils.DefaultControlPlanePort, -1, -1)

	autoscalingConfig := control_plane.NewDefaultAutoscalingMetadata()
	autoscalingConfig.ScalingUpperBound = 1
	//autoscalingConfig.ScalingLowerBound = 1

	wg := sync.WaitGroup{}
	wg.Add(x)

	for i := 0; i < x; i++ {
		go func() {
			deploySingleService(t, cpApi, autoscalingConfig, randSeq(25))
			wg.Done()
		}()
	}

	wg.Wait()
}

func TestDeployService(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.TextFormatter{TimestampFormat: time.StampMilli, FullTimestamp: true})

	deployXservices(t, 1)
}

func TestDeployRandomService(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.TextFormatter{TimestampFormat: time.StampMilli, FullTimestamp: true})

	deployXservices(t, 1)
}

func TestDeploy10Services(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.TextFormatter{TimestampFormat: time.StampMilli, FullTimestamp: true})

	deployXservices(t, 10)
}

func TestDeploy100Services(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.TextFormatter{TimestampFormat: time.StampMilli, FullTimestamp: true})

	deployXservices(t, 100)
}

func TestDeploy1000Services(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.TextFormatter{TimestampFormat: time.StampMilli, FullTimestamp: true})

	deployXservices(t, 1000)
}

func TestDeploy10000Services(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.TextFormatter{TimestampFormat: time.StampMilli, FullTimestamp: true})

	deployXservices(t, 10000)
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

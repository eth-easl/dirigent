package tests

import (
	proto2 "cluster_manager/api/proto"
	"cluster_manager/internal/control_plane/autoscaling"
	"cluster_manager/pkg/grpc_helpers"
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
	ctx, cancel := context.WithTimeout(context.Background(), grpc_helpers.GRPCFunctionTimeout)
	defer cancel()

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
	cpApi := grpc_helpers.InitializeControlPlaneConnection("localhost", utils.DefaultControlPlanePort, -1, -1)

	autoscalingConfig := autoscaling.NewDefaultAutoscalingMetadata()
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

func deployService(t *testing.T, nbDeploys int) {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.TextFormatter{TimestampFormat: time.StampMilli, FullTimestamp: true})

	cpApi := grpc_helpers.InitializeControlPlaneConnection("localhost", utils.DefaultControlPlanePort, -1, -1)

	ctx, cancel := context.WithTimeout(context.Background(), grpc_helpers.GRPCFunctionTimeout)
	defer cancel()

	autoscalingConfig := autoscaling.NewDefaultAutoscalingMetadata()
	autoscalingConfig.ScalingUpperBound = 1
	//autoscalingConfig.ScalingLowerBound = 1

	for i := 0; i < nbDeploys; i++ {
		resp, err := cpApi.RegisterService(ctx, &proto2.ServiceInfo{
			Name:  fmt.Sprintf("%s&%d", deployedFunctionName, i),
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
}

func TestDeploy100Services(t *testing.T) {
	deployService(t, 100)
}

func TestDeploy1000Services(t *testing.T) {
	deployService(t, 1000)
}

func TestDeploy10000Services(t *testing.T) {
	deployService(t, 10000)
}

func TestDeploy100000Services(t *testing.T) {
	deployService(t, 100000)
}

func testXInvocations(t *testing.T, nbInvocations int) {
	wg := sync.WaitGroup{}
	wg.Add(nbInvocations)

	for i := 0; i < nbInvocations; i++ {
		go func(i int) {
			conn := grpc_helpers.EstablishGRPCConnectionPoll("localhost", "8080", grpc.WithAuthority(fmt.Sprintf("%s:%d", deployedFunctionName, i)))
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

func TestInvocationProxying(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.TextFormatter{TimestampFormat: time.StampMilli, FullTimestamp: true})

	testXInvocations(t, 1)
}

func TestInvocation25Proxying(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.TextFormatter{TimestampFormat: time.StampMilli, FullTimestamp: true})

	testXInvocations(t, 25)
}

func TestInvocation50Proxying(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.TextFormatter{TimestampFormat: time.StampMilli, FullTimestamp: true})

	testXInvocations(t, 50)
}

func TestInvocation100Proxying(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.TextFormatter{TimestampFormat: time.StampMilli, FullTimestamp: true})

	testXInvocations(t, 100)
}

func TestInvocation200Proxying(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.TextFormatter{TimestampFormat: time.StampMilli, FullTimestamp: true})

	testXInvocations(t, 200)
}

func TestInvocation400Proxying(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.TextFormatter{TimestampFormat: time.StampMilli, FullTimestamp: true})

	testXInvocations(t, 400)
}

func TestInvocation800Proxying(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.TextFormatter{TimestampFormat: time.StampMilli, FullTimestamp: true})

	testXInvocations(t, 800)
}

func Test_100Invocations(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.TextFormatter{TimestampFormat: time.StampMilli, FullTimestamp: true})

	conn := grpc_helpers.EstablishGRPCConnectionPoll("localhost", "8080", grpc.WithAuthority(deployedFunctionName))
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

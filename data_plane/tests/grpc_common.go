package tests

import (
	proto2 "cluster_manager/api/proto"
	"cluster_manager/tests/proto"
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
	"net"
	"testing"
	"time"
)

const (
	GRPCConnectionTimeout = 10 * time.Second
	GRPCFunctionTimeout   = 30 * time.Second
)

func GRPCConnectionClose(conn *grpc.ClientConn) {
	if conn == nil {
		return
	}

	if err := conn.Close(); err != nil {
		logrus.Fatal(fmt.Sprintf("Error while closing gRPC connection - %s\n", err))
	}
}

type testServer struct {
	proto.UnimplementedExecutorServer
}

func (s *testServer) Execute(_ context.Context, _ *proto.FaasRequest) (*proto.FaasReply, error) {
	return &proto.FaasReply{
		Message:            "OK",
		DurationInMicroSec: 123,
		MemoryUsageInKb:    456,
	}, nil
}

func EstablishConnection(endpoint string) (*grpc.ClientConn, error) {
	dialContext, cancelDialing := context.WithTimeout(context.Background(), GRPCConnectionTimeout)
	defer cancelDialing()

	var dialOptions []grpc.DialOption
	dialOptions = append(dialOptions, grpc.WithTransportCredentials(insecure.NewCredentials()))
	dialOptions = append(dialOptions, grpc.WithBlock())

	return grpc.DialContext(dialContext, endpoint, dialOptions...)
}

func StartGRPCServer(serverAddress string, serverPort string) {
	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%s", serverAddress, serverPort))
	if err != nil {
		logrus.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	reflection.Register(grpcServer)
	proto.RegisterExecutorServer(grpcServer, &testServer{})
	err = grpcServer.Serve(lis)
	if err != nil {
		logrus.Fatal("Failed to start a gRPC server.")
	}
}

func FireInvocation(t *testing.T, host string, dpPort string) error {
	conn, err := EstablishConnection(net.JoinHostPort(host, dpPort))
	defer GRPCConnectionClose(conn)
	if err != nil {
		t.Errorf("gRPC connection timeout - %v\n", err)
	}

	executionCxt, cancelExecution := context.WithTimeout(context.Background(), GRPCFunctionTimeout)
	defer cancelExecution()

	_, err = proto.NewExecutorClient(conn).Execute(executionCxt, &proto.FaasRequest{
		Message:           "nothing",
		RuntimeInMilliSec: uint32(1000),
		MemoryInMebiBytes: uint32(2048),
	})

	return err
}

func UpdateEndpointList(t *testing.T, host string, port string, endpoints []string) {
	apiServerURL := net.JoinHostPort(host, port)

	conn, err := EstablishConnection(apiServerURL)
	defer GRPCConnectionClose(conn)
	if err != nil {
		t.Errorf("gRPC connection timeout - %v\n", err)
	}

	executionCxt, cancelExecution := context.WithTimeout(context.Background(), GRPCFunctionTimeout)
	defer cancelExecution()

	resp, err := proto2.NewDpiInterfaceClient(conn).UpdateEndpointList(executionCxt, &proto2.DeploymentEndpointPatch{
		Deployment: &proto2.DeploymentName{
			Name: "/faas.Executor/Execute",
		},
		Endpoints: endpoints,
	})

	if !resp.Success {
		t.Error("Update endpoint list call did not succeed.")
	} else if err != nil {
		t.Error(fmt.Sprintf("Failed to update endpoint list - %s", err))
	}
}

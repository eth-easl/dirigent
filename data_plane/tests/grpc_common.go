package tests

import (
	"cluster_manager/tests/proto"
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
	"net"
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

func StartGRPCServer(serverAddress string, serverPort int) {
	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", serverAddress, serverPort))
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

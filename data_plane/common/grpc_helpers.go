package common

import (
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
	GRPCConnectionTimeout = 5 * time.Second
	GRPCFunctionTimeout   = 30 * time.Second
)

func CreateGRPCServer(host, port string, serverSpecific func(sr grpc.ServiceRegistrar), options ...grpc.ServerOption) {
	lis, err := net.Listen("tcp", net.JoinHostPort(host, port))
	if err != nil {
		logrus.Fatalf("Failed to create data plane API server socket - %s\n", err)
	}

	grpcServer := grpc.NewServer(options...)
	reflection.Register(grpcServer)

	serverSpecific(grpcServer)

	err = grpcServer.Serve(lis)
	if err != nil {
		logrus.Fatalf("Failed to bind the API Server gRPC handler - %s\n", err)
	}
}

func EstablishConnection(ctx context.Context, endpoint string, additionalOptions ...grpc.DialOption) (*grpc.ClientConn, error) {
	dialContext, cancelDialing := context.WithTimeout(ctx, GRPCConnectionTimeout)
	defer cancelDialing()

	var dialOptions []grpc.DialOption
	dialOptions = append(dialOptions, grpc.WithTransportCredentials(insecure.NewCredentials()))
	dialOptions = append(dialOptions, grpc.WithBlock())
	dialOptions = append(dialOptions, additionalOptions...)

	return grpc.DialContext(dialContext, endpoint, dialOptions...)
}

func GRPCConnectionClose(conn *grpc.ClientConn) {
	if conn == nil {
		return
	}

	if err := conn.Close(); err != nil {
		logrus.Fatal(fmt.Sprintf("Error while closing gRPC connection - %s\n", err))
	}
}

func GetLongLivingConnectionDialOptions() []grpc.DialOption {
	var options []grpc.DialOption
	/*options = append(options, grpc.WithKeepaliveParams(keepalive.ClientParameters{
		Time:                10 * time.Second, // default connection timeout on server side is 20 seconds
		Timeout:             30 * time.Minute,
		PermitWithoutStream: true,
	}))*/
	options = append(options, grpc.WithBlock())

	return options
}

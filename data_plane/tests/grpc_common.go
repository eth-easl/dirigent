package tests

import (
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"time"
)

const (
	grpcConnectionTimeout = 10 * time.Second
	grpcFunctionTimeout   = 30 * time.Second
)

func gRPCConnectionClose(conn *grpc.ClientConn) {
	if conn == nil {
		return
	}

	if err := conn.Close(); err != nil {
		logrus.Fatal(fmt.Sprintf("Error while closing gRPC connection - %s\n", err))
	}
}

func establishConnection(endpoint string) (*grpc.ClientConn, error) {
	dialContext, cancelDialing := context.WithTimeout(context.Background(), grpcConnectionTimeout)
	defer cancelDialing()

	var dialOptions []grpc.DialOption
	dialOptions = append(dialOptions, grpc.WithTransportCredentials(insecure.NewCredentials()))
	dialOptions = append(dialOptions, grpc.WithBlock())

	return grpc.DialContext(dialContext, endpoint, dialOptions...)
}

package main

import (
	"cluster_manager/api"
	"cluster_manager/api/proto"
	"cluster_manager/common"
	"context"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"k8s.io/apimachinery/pkg/util/wait"
	"net"
	"time"
)

func main() {
	logrus.SetLevel(logrus.DebugLevel)

	stopCh := make(chan struct{})
	cpApiServer := &api.CpApiServer{}

	cpApiServer.DpiInterface = InitializeDataPlaneConnection()

	common.CreateGRPCServer(common.ControlPlaneHost, common.ControlPlanePort, func(sr grpc.ServiceRegistrar) {
		proto.RegisterCpiInterfaceServer(sr, cpApiServer)
	})

	<-stopCh
}

func InitializeDataPlaneConnection() proto.DpiInterfaceClient {
	var conn *grpc.ClientConn

	pollContext, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pollErr := wait.PollUntilContextCancel(pollContext, 5*time.Second, false,
		func(ctx context.Context) (done bool, err error) {
			c, err := common.EstablishConnection(
				ctx,
				net.JoinHostPort(common.DataPlaneHost, common.DataPlaneApiPort),
				common.GetLongLivingConnectionDialOptions()...,
			)
			if err != nil {
				logrus.Warn("Retrying to connect to the data plane in 5 seconds")
			}

			conn = c
			return c != nil, nil
		},
	)
	if pollErr != nil {
		logrus.Fatal("Failed to establish connection with the data plane")
	}

	logrus.Info("Successfully established connection with the data plane")
	return proto.NewDpiInterfaceClient(conn)
}

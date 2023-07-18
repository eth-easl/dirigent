package common

import (
	"cluster_manager/api/proto"
	"cluster_manager/pkg/utils"
	"context"
	"fmt"
	"net"
	"time"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/reflection"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	GRPCConnectionTimeout = 5 * time.Second
	GRPCFunctionTimeout   = 30 * time.Second
)

func CreateGRPCServer(host, port string, serverSpecific func(sr grpc.ServiceRegistrar), options ...grpc.ServerOption) {
	lis, err := net.Listen(utils.TCP, net.JoinHostPort(host, port))
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

func dialConnection(ctx context.Context, endpoint string, additionalOptions ...grpc.DialOption) (*grpc.ClientConn, error) {
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

func EstablishGRPCConnectionPoll(host, port string, dialOptions ...grpc.DialOption) *grpc.ClientConn {
	var conn *grpc.ClientConn

	var options []grpc.DialOption

	options = append(options, GetLongLivingConnectionDialOptions()...)
	options = append(options, dialOptions...)

	_ = wait.PollUntilContextCancel(context.Background(), 5*time.Second, false,
		func(ctx context.Context) (done bool, err error) {
			establishContext, end := context.WithTimeout(ctx, 3*time.Second)
			defer end()

			c, err := dialConnection(
				establishContext,
				net.JoinHostPort(host, port),
				options...,
			)
			if err != nil {
				logrus.Warn("Retrying to establish gRPC connection in 5 seconds...")
			}

			conn = c

			return c != nil, nil
		},
	)

	return conn
}

func InitializeControlPlaneConnection(host string, port string, dataplanePort, proxyPort int32) proto.CpiInterfaceClient {
	conn := EstablishGRPCConnectionPoll(host, port)
	if conn == nil {
		logrus.Fatal("Failed to establish connection with the control plane")
	}

	logrus.Info("Successfully established connection with the control plane")

	dpiClient := proto.NewCpiInterfaceClient(conn)

	if dataplanePort != -1 {
		dpInfo := &proto.DataplaneInfo{
			APIPort:   dataplanePort,
			ProxyPort: proxyPort,
		}

		resp, err := dpiClient.RegisterDataplane(context.Background(), dpInfo)
		if err != nil || !resp.Success {
			logrus.Fatal("Failed to register data plane with the control plane")
		}
	}

	return dpiClient
}

func DeregisterControlPlaneConnection(host string, port string, dataplanePort, proxyPort int32) proto.CpiInterfaceClient {
	conn := EstablishGRPCConnectionPoll(host, port)
	if conn == nil {
		logrus.Fatal("Failed to establish connection with the control plane")
	}

	logrus.Info("Successfully established connection with the control plane")

	dpiClient := proto.NewCpiInterfaceClient(conn)

	if dataplanePort != -1 {
		dpInfo := &proto.DataplaneInfo{
			APIPort:   dataplanePort,
			ProxyPort: proxyPort,
		}

		resp, err := dpiClient.DeregisterDataplane(context.Background(), dpInfo)
		if err != nil || !resp.Success {
			logrus.Fatal("Failed to register data plane with the control plane")
		}
	}

	return dpiClient
}

func InitializeWorkerNodeConnection(host, port string) proto.WorkerNodeInterfaceClient {
	conn := EstablishGRPCConnectionPoll(host, port)
	if conn == nil {
		logrus.Fatal("Failed to establish connection with the worker node")
	}

	logrus.Info("Successfully established connection with the worker node")

	return proto.NewWorkerNodeInterfaceClient(conn)
}

func InitializeDataPlaneConnection(host string, port string) proto.DpiInterfaceClient {
	conn := EstablishGRPCConnectionPoll(host, port)
	if conn == nil {
		logrus.Fatal("Failed to establish connection with the data plane")
	}

	logrus.Info("Successfully established connection with the data plane")

	return proto.NewDpiInterfaceClient(conn)
}

func GetIPAddressFromGRPCCall(ctx context.Context) (string, bool) {
	peerCtx, ok := peer.FromContext(ctx)
	if !ok {
		return "", false
	}

	switch addr := peerCtx.Addr.(type) {
	case *net.UDPAddr:
		return addr.IP.String(), true
	case *net.TCPAddr:
		return addr.IP.String(), true
	default:
		return "", false
	}
}

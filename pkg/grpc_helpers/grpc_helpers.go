package grpc_helpers

import (
	"cluster_manager/api/proto"
	"cluster_manager/pkg/config"
	"cluster_manager/pkg/utils"
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
	"k8s.io/apimachinery/pkg/util/wait"
)

func CreateGRPCServer(port string, serverSpecific func(sr grpc.ServiceRegistrar), options ...grpc.ServerOption) {
	lis, err := net.Listen(utils.TCP, net.JoinHostPort(utils.Localhost, port))
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
	dialContext, cancelDialing := context.WithTimeout(ctx, utils.GRPCConnectionTimeout)
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

func GetPeerPort(address string) int {
	_, portString, err := net.SplitHostPort(address)
	if err != nil {
		logrus.Fatal("Invalid network address of control plane replica.")
	}

	port, err := strconv.Atoi(portString)
	if err != nil {
		logrus.Fatal("Invalid port of control plane replica.")
	}

	return port
}

func ParseReplicaPorts(cfg *config.ControlPlaneConfig) []int {
	var result []int

	for i := 0; i < len(cfg.Replicas); i++ {
		result = append(result, GetPeerPort(cfg.Replicas[i]))
	}

	return result
}

func EstablishGRPCConnectionPoll(host, port string, dialOptions ...grpc.DialOption) *grpc.ClientConn {
	var conn *grpc.ClientConn

	var options []grpc.DialOption

	options = append(options, GetLongLivingConnectionDialOptions()...)
	options = append(options, dialOptions...)

	_ = wait.PollUntilContextCancel(context.Background(), 5*time.Second, true,
		func(ctx context.Context) (done bool, err error) {
			establishContext, end := context.WithTimeout(ctx, utils.GRPCFunctionTimeout)
			defer end()

			c, err := dialConnection(
				establishContext,
				net.JoinHostPort(host, port),
				options...,
			)
			if err != nil {
				logrus.Warnf("Retrying to establish gRPC connection (%s:%s) in 5 seconds...", host, port)
			}

			conn = c

			return c != nil, nil
		},
	)

	logrus.Infof("Successfully established connection with %s:%s", host, port)

	return conn
}

func InitializeControlPlaneConnection(host string, port string, dataplaneIP string, dataplanePort, proxyPort int32) (proto.CpiInterfaceClient, error) {
	conn := EstablishGRPCConnectionPoll(host, port)
	if conn == nil {
		logrus.Fatal("Failed to establish connection with the control plane")
	}

	logrus.Info("Successfully established connection with the control plane")

	dpiClient := proto.NewCpiInterfaceClient(conn)

	// TODO: this method should be split into two because of parameters
	if dataplanePort != -1 {
		dpInfo := &proto.DataplaneInfo{
			IP:        dataplaneIP,
			APIPort:   dataplanePort,
			ProxyPort: proxyPort,
		}

		resp, err := dpiClient.RegisterDataplane(context.Background(), dpInfo)
		if err != nil || !resp.Success {
			logrus.Fatal("Failed to register data plane with the control plane")
		}
	}

	return dpiClient, nil
}

func DeregisterControlPlaneConnection(cfg *config.DataPlaneConfig) error {
	grpcPort, _ := strconv.Atoi(cfg.PortGRPC)
	proxyPort, _ := strconv.Atoi(cfg.PortProxy)

	conn := EstablishGRPCConnectionPoll(cfg.ControlPlaneIp, cfg.ControlPlanePort)
	cpApi := proto.NewCpiInterfaceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	resp, err := cpApi.DeregisterDataplane(ctx, &proto.DataplaneInfo{
		IP:        cfg.DataPlaneIp,
		APIPort:   int32(grpcPort),
		ProxyPort: int32(proxyPort),
	})
	if err != nil || !resp.Success {
		logrus.Fatal("Failed to register data plane with the control plane")
	}

	return err
}

func InitializeWorkerNodeConnection(host, port string) (proto.WorkerNodeInterfaceClient, error) {
	conn := EstablishGRPCConnectionPoll(host, port)
	if conn == nil {
		return nil, errors.New("Failed to establish connection with the worker node")
	}

	logrus.Info("Successfully established connection with the worker node")

	return proto.NewWorkerNodeInterfaceClient(conn), nil
}

func InitializeDataPlaneConnection(host string, port string) (proto.DpiInterfaceClient, error) {
	conn := EstablishGRPCConnectionPoll(host, port)
	if conn == nil {
		return nil, errors.New("Failed to establish connection with the data plane")
	}

	logrus.Info("Successfully established connection with the data plane")

	return proto.NewDpiInterfaceClient(conn), nil
}

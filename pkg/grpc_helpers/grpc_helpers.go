package grpc_helpers

import (
	"cluster_manager/api/proto"
	"cluster_manager/pkg/config"
	"cluster_manager/pkg/utils"
	"context"
	"errors"
	"fmt"
	"math/rand"
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

func SplitAddress(address string) (string, int) {
	hostString, portString, err := net.SplitHostPort(address)
	if err != nil {
		logrus.Fatalf("Invalid network address of control plane replica - %s", address)
	}

	port, err := strconv.Atoi(portString)
	if err != nil {
		logrus.Fatalf("Invalid port of control plane replica - %s.", portString)
	}

	return hostString, port
}

func ParseReplicaPorts(cfg *config.ControlPlaneConfig) []int {
	var result []int

	for i := 0; i < len(cfg.Replicas); i++ {
		_, port := SplitAddress(cfg.Replicas[i])
		result = append(result, port)
	}

	return result
}

func EstablishGRPCConnectionPoll(addresses []string, dialOptions ...grpc.DialOption) *grpc.ClientConn {
	var conn *grpc.ClientConn
	var options []grpc.DialOption

	options = append(options, GetLongLivingConnectionDialOptions()...)
	options = append(options, dialOptions...)

	attempt := 0

	// Establish a connection with at least one of the addresses.
	for {
		var addr string
		for {
			addr = addresses[rand.Intn(len(addresses))]
			err := wait.PollUntilContextCancel(context.Background(), 5*time.Second, true,
				func(ctx context.Context) (done bool, err error) {
					establishContext, end := context.WithTimeout(ctx, utils.GRPCFunctionTimeout)
					defer end()

					c, err := dialConnection(
						establishContext,
						addr,
						options...,
					)
					if err != nil {
						logrus.Errorf("Retrying to establish gRPC connection with %s...", addr)
						return false, err
					}

					conn = c
					return c != nil, nil
				},
			)

			if err == nil {
				break
			}
		}

		if conn != nil {
			logrus.Infof("Successfully established connection with %s", addr)
			break
		} else {
			attempt++
			logrus.Errorf("Could not establish connection with any of the specified servers (attempt #%d)", attempt)
		}
	}

	return conn
}

func NewControlPlaneConnection(addresses []string) (proto.CpiInterfaceClient, error) {
	conn := EstablishGRPCConnectionPoll(addresses)
	if conn == nil {
		logrus.Fatal("Failed to establish connection with the control plane")
	}

	logrus.Info("Successfully established connection with the control plane")
	return proto.NewCpiInterfaceClient(conn), nil
}

func DeregisterControlPlaneConnection(cfg *config.DataPlaneConfig) error {
	grpcPort, _ := strconv.Atoi(cfg.PortGRPC)
	proxyPort, _ := strconv.Atoi(cfg.PortProxy)

	var cpApi proto.CpiInterfaceClient
	for _, rawAddr := range cfg.ControlPlaneAddress {
		host, port := SplitAddress(rawAddr)

		conn := EstablishGRPCConnectionPoll([]string{net.JoinHostPort(host, strconv.Itoa(port))})
		if conn == nil {
			continue
		}

		cpApi = proto.NewCpiInterfaceClient(conn)
	}

	if cpApi == nil {
		logrus.Errorf("Error deregistering control plane connection. Cannot establish connection with the API Server.")
		return errors.New("error deregistering control plane")
	}

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
	conn := EstablishGRPCConnectionPoll([]string{net.JoinHostPort(host, port)})
	if conn == nil {
		return nil, errors.New("failed to establish connection with the worker node")
	}

	logrus.Info("Successfully established connection with the worker node")
	return proto.NewWorkerNodeInterfaceClient(conn), nil
}

func InitializeDataPlaneConnection(host string, port string) (proto.DpiInterfaceClient, error) {
	conn := EstablishGRPCConnectionPoll([]string{net.JoinHostPort(host, port)})
	if conn == nil {
		return nil, errors.New("failed to establish connection with the data plane")
	}

	logrus.Info("Successfully established connection with the data plane")

	return proto.NewDpiInterfaceClient(conn), nil
}

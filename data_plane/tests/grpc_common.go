package tests

import (
	proto2 "cluster_manager/api/proto"
	"cluster_manager/common"
	"cluster_manager/tests/proto"
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"testing"
)

type TestServer struct {
	proto.UnimplementedExecutorServer
}

func (s *TestServer) Execute(_ context.Context, _ *proto.FaasRequest) (*proto.FaasReply, error) {
	return &proto.FaasReply{
		Message:            "OK",
		DurationInMicroSec: 123,
		MemoryUsageInKb:    456,
	}, nil
}

func FireInvocation(t *testing.T, host string, dpPort string) error {
	conn := common.EstablishGRPCConnectionPoll(host, dpPort)
	if conn == nil {
		logrus.Fatal("Failed to establish gRPC connection with the data plane")
	}

	executionCxt, cancelExecution := context.WithTimeout(context.Background(), common.GRPCFunctionTimeout)
	defer cancelExecution()

	_, err := proto.NewExecutorClient(conn).Execute(executionCxt, &proto.FaasRequest{
		Message:           "nothing",
		RuntimeInMilliSec: uint32(1000),
		MemoryInMebiBytes: uint32(2048),
	})

	return err
}

func UpdateEndpointList(t *testing.T, host string, port string, endpoints []string) {
	conn := common.EstablishGRPCConnectionPoll(host, port)
	if conn == nil {
		logrus.Fatal("Failed to establish gRPC connection with the data plane")
	}

	executionCxt, cancelExecution := context.WithTimeout(context.Background(), common.GRPCFunctionTimeout)
	defer cancelExecution()

	// TODO: make this not be static and random
	resp, err := proto2.NewDpiInterfaceClient(conn).UpdateEndpointList(executionCxt, &proto2.DeploymentEndpointPatch{
		Service: &proto2.ServiceInfo{
			Name:  "/faas.Executor/Execute",
			Image: "docker.io/cvetkovic/empty_function:latest",
			PortForwarding: []*proto2.PortMapping{
				{
					GuestPort: 80,
					Protocol:  proto2.L4Protocol_TCP,
				},
			},
		},
		Endpoints: endpoints,
	})

	if !resp.Success {
		t.Error("Update endpoint list call did not succeed.")
	} else if err != nil {
		t.Error(fmt.Sprintf("Failed to update endpoint list - %s", err))
	}
}

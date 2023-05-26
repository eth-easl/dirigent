package tests

import (
	proto2 "cluster_manager/api/proto"
	"cluster_manager/common"
	"cluster_manager/tests/proto"
	"context"
	"fmt"
	"net"
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
	conn, err := common.EstablishConnection(context.Background(), net.JoinHostPort(host, dpPort))
	defer common.GRPCConnectionClose(conn)
	if err != nil {
		t.Errorf("gRPC connection timeout - %v\n", err)
	}

	executionCxt, cancelExecution := context.WithTimeout(context.Background(), common.GRPCFunctionTimeout)
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

	conn, err := common.EstablishConnection(context.Background(), apiServerURL)
	defer common.GRPCConnectionClose(conn)
	if err != nil {
		t.Errorf("gRPC connection timeout - %v\n", err)
	}

	executionCxt, cancelExecution := context.WithTimeout(context.Background(), common.GRPCFunctionTimeout)
	defer cancelExecution()

	resp, err := proto2.NewDpiInterfaceClient(conn).UpdateEndpointList(executionCxt, &proto2.DeploymentEndpointPatch{
		Service: &proto2.ServiceInfo{
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

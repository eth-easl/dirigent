package tests

import (
	"cluster_manager/tests/proto"
	"context"
	"fmt"
	"net"
	"testing"
)

func TestInvocationProxying(t *testing.T) {
	endpoint := net.JoinHostPort("localhost", "8080")

	conn, err := establishConnection(endpoint)
	defer gRPCConnectionClose(conn)
	if err != nil {
		t.Errorf("gRPC connection timeout - %v\n", err)
	}

	executionCxt, cancelExecution := context.WithTimeout(context.Background(), grpcFunctionTimeout)
	defer cancelExecution()

	_, err = proto.NewExecutorClient(conn).Execute(executionCxt, &proto.FaasRequest{
		Message:           "nothing",
		RuntimeInMilliSec: uint32(1000),
		MemoryInMebiBytes: uint32(2048),
	})
	if err != nil {
		t.Error(fmt.Sprintf("Function timeout - %s", err))
	}
}

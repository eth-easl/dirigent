package tests

import (
	proto2 "cluster_manager/api/proto"
	"context"
	"fmt"
	"net"
	"testing"
)

func TestInstanceAppeared(t *testing.T) {
	endpoint := net.JoinHostPort("localhost", "8081")

	conn, err := establishConnection(endpoint)
	defer gRPCConnectionClose(conn)
	if err != nil {
		t.Errorf("gRPC connection timeout - %v\n", err)
	}

	executionCxt, cancelExecution := context.WithTimeout(context.Background(), grpcFunctionTimeout)
	defer cancelExecution()

	_, err = proto2.NewDpiInterfaceClient(conn).UpdateEndpointList(executionCxt, &proto2.DeploymentEndpointPatch{
		Deployment: &proto2.DeploymentName{
			Name: "/faas.Executor/Execute",
		},
		Endpoints: []string{"localhost:80"},
	})

	if err != nil {
		t.Error(fmt.Sprintf("Function timeout - %s", err))
	}
}

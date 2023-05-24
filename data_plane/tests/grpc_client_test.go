package tests

import (
	"cluster_manager/tests/proto"
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"net"
	"testing"
	"time"
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

	start := time.Now()

	_, err = proto.NewExecutorClient(conn).Execute(executionCxt, &proto.FaasRequest{
		Message:           "nothing",
		RuntimeInMilliSec: uint32(1000),
		MemoryInMebiBytes: uint32(2048),
	})
	if err != nil {
		t.Error(fmt.Sprintf("Function timeout - %s", err))
	}

	logrus.Info("E2E request latency ", time.Since(start).Microseconds(), "μs")
}

package shared

import (
	"cluster_manager/pkg/utils"
	"cluster_manager/tests/proto"
	"context"
	"github.com/sirupsen/logrus"
	"time"
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

func FireInvocation(client proto.ExecutorClient) error {
	executionCxt, cancelExecution := context.WithTimeout(context.Background(), utils.GRPCFunctionTimeout)
	defer cancelExecution()

	start := time.Now()
	_, err := client.Execute(executionCxt, &proto.FaasRequest{
		Message:           "nothing",
		RuntimeInMilliSec: uint32(1000),
		MemoryInMebiBytes: uint32(2048),
	})

	logrus.Info("Request completed - took ", time.Since(start).Microseconds(), " μs")

	return err
}

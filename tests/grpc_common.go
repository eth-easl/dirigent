package tests

import (
	protoApi "cluster_manager/api/proto"
	"cluster_manager/internal/common"
	"cluster_manager/pkg/utils"
	"cluster_manager/tests/proto"
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
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
	executionCxt, cancelExecution := context.WithTimeout(context.Background(), common.GRPCFunctionTimeout)
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

func UpdateEndpointList(t *testing.T, host string, port string, endpoints []*protoApi.EndpointInfo) {
	conn := common.EstablishGRPCConnectionPoll(host, port)
	if conn == nil {
		logrus.Fatal("Failed to establish gRPC connection with the data plane")
	}

	executionCxt, cancelExecution := context.WithTimeout(context.Background(), common.GRPCFunctionTimeout)
	defer cancelExecution()

	// TODO: make this not be static and random
	resp, err := protoApi.NewDpiInterfaceClient(conn).UpdateEndpointList(executionCxt, &protoApi.DeploymentEndpointPatch{
		Service: &protoApi.ServiceInfo{
			Name:  "/faas.Executor/Execute",
			Image: utils.TestDockerImageName,
			PortForwarding: &protoApi.PortMapping{
				GuestPort: 80,
				Protocol:  protoApi.L4Protocol_TCP,
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

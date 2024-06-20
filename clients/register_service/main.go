package register_service

import (
	"cluster_manager/internal/control_plane/control_plane/autoscalers"
	common "cluster_manager/pkg/grpc_helpers"
	"cluster_manager/pkg/utils"
	"cluster_manager/proto"
	"context"

	"github.com/sirupsen/logrus"
)

func Deployservice() {
	cpApi, err := common.NewControlPlaneConnection([]string{"127.0.0.1:9090"})
	if err != nil {
		logrus.Fatalf("Failed to start control plane connection (error %s)", err.Error())
	}

	ctx, cancel := context.WithTimeout(context.Background(), utils.GRPCFunctionTimeout)
	defer cancel()

	autoscalingConfig := autoscalers.NewDefaultAutoscalingConfiguration()
	autoscalingConfig.ScalingUpperBound = 1

	resp, err := cpApi.RegisterService(ctx, &proto.ServiceInfo{
		Name:  "test",
		Image: "docker.io/cvetkovic/dirigent_empty_function:latest",
		PortForwarding: &proto.PortMapping{
			GuestPort: 80,
			Protocol:  proto.L4Protocol_TCP,
		},
		AutoscalingConfig: autoscalingConfig,
		SandboxConfiguration: &proto.SandboxConfiguration{
			IterationMultiplier: 102,
		},
	})

	if err != nil || !resp.Success {
		logrus.Errorf("Failed to deploy service, maybe service is already registered?")
	}
}

func main() {
	Deployservice()
}

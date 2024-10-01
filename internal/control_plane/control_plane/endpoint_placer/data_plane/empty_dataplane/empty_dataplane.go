package empty_dataplane

import (
	"cluster_manager/internal/control_plane/control_plane/core"
	"cluster_manager/proto"
	"context"
	"time"

	"google.golang.org/protobuf/types/known/emptypb"
)

type emptyDataplane struct{}

func NewDataplaneConnectionEmpty(IP, APIPort, ProxyPort string) core.DataPlaneInterface {
	return &emptyDataplane{}
}

func (e emptyDataplane) InitializeDataPlaneConnection(host string, port string) error {
	return nil
}

func (e emptyDataplane) AddDeployment(ctx context.Context, info *proto.ServiceInfo) (*proto.DeploymentUpdateSuccess, error) {
	return &proto.DeploymentUpdateSuccess{
		Success: true,
		Message: "",
	}, nil
}

func (e emptyDataplane) AddWorkflowDeployment(ctx context.Context, info *proto.WorkflowInfo) (*proto.DeploymentUpdateSuccess, error) {
	return &proto.DeploymentUpdateSuccess{
		Success: true,
		Message: "",
	}, nil
}

func (e emptyDataplane) UpdateDeployment(ctx context.Context, info *proto.ServiceInfo) (*proto.DeploymentUpdateSuccess, error) {
	return &proto.DeploymentUpdateSuccess{
		Success: true,
		Message: "",
	}, nil
}

func (e emptyDataplane) UpdateEndpointList(ctx context.Context, patch *proto.DeploymentEndpointPatch) (*proto.DeploymentUpdateSuccess, error) {
	return &proto.DeploymentUpdateSuccess{
		Success: true,
		Message: "",
	}, nil
}

func (e emptyDataplane) DeleteDeployment(ctx context.Context, info *proto.ServiceInfo) (*proto.DeploymentUpdateSuccess, error) {
	return &proto.DeploymentUpdateSuccess{
		Success: true,
		Message: "",
	}, nil
}

func (e emptyDataplane) DeleteWorkflowDeployment(ctx context.Context, info *proto.WorkflowObjectIdentifier) (*proto.DeploymentUpdateSuccess, error) {
	return &proto.DeploymentUpdateSuccess{
		Success: true,
		Message: "",
	}, nil
}

func (e emptyDataplane) DrainSandbox(context.Context, *proto.DeploymentEndpointPatch) (*proto.DeploymentUpdateSuccess, error) {
	return &proto.DeploymentUpdateSuccess{
		Success: true,
		Message: "",
	}, nil
}

func (e emptyDataplane) ResetMeasurements(ctx context.Context, empty *emptypb.Empty) (*proto.ActionStatus, error) {
	return &proto.ActionStatus{
		Success: true,
		Message: "",
	}, nil
}

func (e emptyDataplane) GetLastHeartBeat() time.Time {
	return time.Now()
}

func (e emptyDataplane) UpdateHeartBeat() {
}

func (e emptyDataplane) GetIP() string {
	return ""
}

func (e emptyDataplane) GetApiPort() string {
	return ""
}

func (e emptyDataplane) GetProxyPort() string {
	return ""
}

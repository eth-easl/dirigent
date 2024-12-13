package data_plane

import (
	"cluster_manager/internal/control_plane/control_plane/core"
	"cluster_manager/pkg/grpc_helpers"
	"cluster_manager/proto"
	"context"
	"google.golang.org/grpc"
	"time"

	"google.golang.org/protobuf/types/known/emptypb"
)

func NewDataplaneConnection(IP, APIPort, ProxyPort string) core.DataPlaneInterface {
	return &DataPlaneConnectionInfo{
		IP:            IP,
		APIPort:       APIPort,
		ProxyPort:     ProxyPort,
		LastHeartBeat: time.Now(),
	}
}

type DataPlaneConnectionInfo struct {
	Iface         proto.DpiInterfaceClient
	IP            string
	APIPort       string
	ProxyPort     string
	LastHeartBeat time.Time
}

func (d *DataPlaneConnectionInfo) GetLastHeartBeat() time.Time {
	return d.LastHeartBeat
}

func (d *DataPlaneConnectionInfo) UpdateHeartBeat() {
	d.LastHeartBeat = time.Now()
}

func (d *DataPlaneConnectionInfo) InitializeDataPlaneConnection(host string, port string) error {
	conn, err := grpc_helpers.InitializeDataPlaneConnection(host, port)
	d.Iface = conn
	return err
}

func (d *DataPlaneConnectionInfo) AddDeployment(ctx context.Context, in *proto.ServiceInfo) (*proto.DeploymentUpdateSuccess, error) {
	return d.Iface.AddDeployment(ctx, in)
}

func (d *DataPlaneConnectionInfo) AddWorkflowDeployment(ctx context.Context, in *proto.WorkflowInfo) (*proto.DeploymentUpdateSuccess, error) {
	return d.Iface.AddWorkflowDeployment(ctx, in)
}

func (d *DataPlaneConnectionInfo) UpdateDeployment(ctx context.Context, in *proto.ServiceInfo) (*proto.DeploymentUpdateSuccess, error) {
	return d.Iface.UpdateDeployment(ctx, in)
}

func (d *DataPlaneConnectionInfo) UpdateEndpointList(ctx context.Context, in *proto.DeploymentEndpointPatch) (*proto.DeploymentUpdateSuccess, error) {
	return d.Iface.UpdateEndpointList(ctx, in)
}

func (d *DataPlaneConnectionInfo) DeleteDeployment(ctx context.Context, in *proto.ServiceInfo) (*proto.DeploymentUpdateSuccess, error) {
	return d.Iface.DeleteDeployment(ctx, in)
}

func (d *DataPlaneConnectionInfo) DeleteWorkflowDeployment(ctx context.Context, in *proto.WorkflowObjectIdentifier) (*proto.DeploymentUpdateSuccess, error) {
	return d.Iface.DeleteWorkflowDeployment(ctx, in)
}

func (d *DataPlaneConnectionInfo) ReceiveRouteUpdate(ctx context.Context, update *proto.RouteUpdate, option ...grpc.CallOption) (*proto.ActionStatus, error) {
	return d.Iface.ReceiveRouteUpdate(ctx, update, option...)
}

func (d *DataPlaneConnectionInfo) DrainSandbox(ctx context.Context, patch *proto.DeploymentEndpointPatch) (*proto.DeploymentUpdateSuccess, error) {
	return d.Iface.DrainSandbox(ctx, patch)
}

func (d *DataPlaneConnectionInfo) ResetMeasurements(ctx context.Context, in *emptypb.Empty) (*proto.ActionStatus, error) {
	return d.Iface.ResetMeasurements(ctx, in)
}

func (d *DataPlaneConnectionInfo) GetIP() string {
	return d.IP
}

func (d *DataPlaneConnectionInfo) GetApiPort() string {
	return d.APIPort
}

func (d *DataPlaneConnectionInfo) GetProxyPort() string {
	return d.ProxyPort
}

package data_plane

import (
	"cluster_manager/api/proto"
	"cluster_manager/internal/control_plane/core"
	"cluster_manager/pkg/grpc_helpers"
	"context"
	"google.golang.org/protobuf/types/known/emptypb"
)

func NewDataplaneConnection(IP, APIPort, ProxyPort string) core.DataPlaneInterface {
	return &DataPlaneConnectionInfo{
		IP:        IP,
		APIPort:   APIPort,
		ProxyPort: ProxyPort,
	}
}

type DataPlaneConnectionInfo struct {
	Iface     proto.DpiInterfaceClient
	IP        string
	APIPort   string
	ProxyPort string
}

func (d *DataPlaneConnectionInfo) InitializeDataPlaneConnection(host string, port string) error {
	conn, err := grpc_helpers.InitializeDataPlaneConnection(host, port)
	d.Iface = conn
	return err
}

func (d *DataPlaneConnectionInfo) AddDeployment(ctx context.Context, in *proto.ServiceInfo) (*proto.DeploymentUpdateSuccess, error) {
	return d.Iface.AddDeployment(ctx, in)
}

func (d *DataPlaneConnectionInfo) UpdateEndpointList(ctx context.Context, in *proto.DeploymentEndpointPatch) (*proto.DeploymentUpdateSuccess, error) {
	return d.Iface.UpdateEndpointList(ctx, in)
}

func (d *DataPlaneConnectionInfo) DeleteDeployment(ctx context.Context, in *proto.ServiceInfo) (*proto.DeploymentUpdateSuccess, error) {
	return d.Iface.DeleteDeployment(ctx, in)
}

func (d *DataPlaneConnectionInfo) ResetMeasurements(ctx context.Context, in *emptypb.Empty) (*proto.ActionStatus, error) {
	return d.ResetMeasurements(ctx, in)
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
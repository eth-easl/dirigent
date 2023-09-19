package api

import (
	"cluster_manager/api/proto"
	"cluster_manager/internal/control_plane"
	"cluster_manager/internal/control_plane/data_plane"
	"cluster_manager/internal/control_plane/persistence"
	"cluster_manager/internal/control_plane/placement_policy"
	"cluster_manager/internal/control_plane/workers"
	config2 "cluster_manager/pkg/config"
	"context"
	"google.golang.org/protobuf/types/known/emptypb"
)

type CpApiServer struct {
	proto.UnimplementedCpiInterfaceServer
	ControlPlane *control_plane.ControlPlane
}

func CreateNewCpApiServer(client persistence.PersistenceLayer, outputFile string, placementPolicy placement_policy.PlacementPolicy) *CpApiServer {
	return &CpApiServer{
		ControlPlane: control_plane.NewControlPlane(client, outputFile, placementPolicy, data_plane.NewDataplaneConnection, workers.NewWorkerNode),
	}
}

func (c *CpApiServer) CheckPeriodicallyWorkerNodes() {
	c.ControlPlane.CheckPeriodicallyWorkerNodes()
}

func (c *CpApiServer) OnMetricsReceive(ctx context.Context, metric *proto.AutoscalingMetric) (*proto.ActionStatus, error) {
	return c.ControlPlane.OnMetricsReceive(ctx, metric)
}

func (c *CpApiServer) ListServices(ctx context.Context, empty *emptypb.Empty) (*proto.ServiceList, error) {
	return c.ControlPlane.ListServices(ctx, empty)
}

func (c *CpApiServer) RegisterNode(ctx context.Context, in *proto.NodeInfo) (*proto.ActionStatus, error) {
	return c.ControlPlane.RegisterNode(ctx, in)
}

func (c *CpApiServer) DeregisterNode(ctx context.Context, in *proto.NodeInfo) (*proto.ActionStatus, error) {
	return c.ControlPlane.DeregisterNode(ctx, in)
}

func (c *CpApiServer) NodeHeartbeat(ctx context.Context, in *proto.NodeHeartbeatMessage) (*proto.ActionStatus, error) {
	return c.ControlPlane.NodeHeartbeat(ctx, in)
}

func (c *CpApiServer) RegisterService(ctx context.Context, serviceInfo *proto.ServiceInfo) (*proto.ActionStatus, error) {
	return c.ControlPlane.RegisterService(ctx, serviceInfo)
}

func (c *CpApiServer) RegisterDataplane(ctx context.Context, in *proto.DataplaneInfo) (*proto.ActionStatus, error) {
	return c.ControlPlane.RegisterDataplane(ctx, in)
}

func (c *CpApiServer) DeregisterDataplane(ctx context.Context, in *proto.DataplaneInfo) (*proto.ActionStatus, error) {
	return c.ControlPlane.DeregisterDataplane(ctx, in)
}

func (c *CpApiServer) ReconstructState(ctx context.Context, config config2.ControlPlaneConfig) error {
	return c.ControlPlane.ReconstructState(ctx, config)
}

func (c *CpApiServer) ResetMeasurements(_ context.Context, _ *emptypb.Empty) (*proto.ActionStatus, error) {
	c.ControlPlane.ColdStartTracing.ResetTracingService()
	return &proto.ActionStatus{Success: true}, nil
}

func (c *CpApiServer) ReportFailure(_ context.Context, in *proto.Failure) (*proto.ActionStatus, error) {
	return &proto.ActionStatus{Success: c.ControlPlane.HandleFailure([]*proto.Failure{in})}, nil
}

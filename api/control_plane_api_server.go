package api

import (
	"cluster_manager/api/proto"
	"cluster_manager/internal/control_plane"
	"cluster_manager/internal/control_plane/core"
	"cluster_manager/internal/control_plane/persistence"
	"cluster_manager/internal/control_plane/placement_policy"
	config2 "cluster_manager/pkg/config"
	"cluster_manager/pkg/grpc_helpers"
	"context"
	"errors"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"net"
)

type CpApiServer struct {
	proto.UnimplementedCpiInterfaceServer

	ControlPlane *control_plane.ControlPlane
}

func CreateNewCpApiServer(client persistence.PersistenceLayer, outputFile string,
	placementPolicy placement_policy.PlacementPolicy, dataplaneCreator core.DataplaneFactory,
	workerNodeCreator core.WorkerNodeFactory, cfg *config2.ControlPlaneConfig) (*CpApiServer, chan bool) {

	cp, isLeader := control_plane.NewControlPlane(client, outputFile, placementPolicy, dataplaneCreator, workerNodeCreator, cfg)
	cpApiServer := &CpApiServer{
		ControlPlane: cp,
	}

	go grpc_helpers.CreateGRPCServer(cfg.Port, func(sr grpc.ServiceRegistrar) {
		proto.RegisterCpiInterfaceServer(sr, cpApiServer)
	})

	// connecting to peers for leader election
	for _, rawAddress := range cfg.Replicas {
		peerID := grpc_helpers.GetPeerPort(rawAddress)
		tcpAddr, _ := net.ResolveTCPAddr("tcp", rawAddress)

		cp.LeaderElectionServer.ConnectToPeer(peerID, tcpAddr)
	}

	return cpApiServer, isLeader
}

func (c *CpApiServer) CheckPeriodicallyWorkerNodes() {
	if !c.ControlPlane.LeaderElectionServer.IsLeader() {
		return
	}

	c.ControlPlane.CheckPeriodicallyWorkerNodes()
}

func (c *CpApiServer) OnMetricsReceive(ctx context.Context, metric *proto.AutoscalingMetric) (*proto.ActionStatus, error) {
	if !c.ControlPlane.LeaderElectionServer.IsLeader() {
		return &proto.ActionStatus{Success: false}, errors.New("Cannot request action from non-leader control plane.")
	}

	return c.ControlPlane.OnMetricsReceive(ctx, metric)
}

func (c *CpApiServer) ListServices(ctx context.Context, empty *emptypb.Empty) (*proto.ServiceList, error) {
	if !c.ControlPlane.LeaderElectionServer.IsLeader() {
		return &proto.ServiceList{}, errors.New("Cannot request action from non-leader control plane.")
	}

	return c.ControlPlane.ListServices(ctx, empty)
}

func (c *CpApiServer) RegisterNode(ctx context.Context, in *proto.NodeInfo) (*proto.ActionStatus, error) {
	if !c.ControlPlane.LeaderElectionServer.IsLeader() {
		return &proto.ActionStatus{Success: false}, errors.New("Cannot request action from non-leader control plane.")
	}

	return c.ControlPlane.RegisterNode(ctx, in)
}

func (c *CpApiServer) DeregisterNode(ctx context.Context, in *proto.NodeInfo) (*proto.ActionStatus, error) {
	if !c.ControlPlane.LeaderElectionServer.IsLeader() {
		return &proto.ActionStatus{Success: false}, errors.New("Cannot request action from non-leader control plane.")
	}

	return c.ControlPlane.DeregisterNode(ctx, in)
}

func (c *CpApiServer) NodeHeartbeat(ctx context.Context, in *proto.NodeHeartbeatMessage) (*proto.ActionStatus, error) {
	if !c.ControlPlane.LeaderElectionServer.IsLeader() {
		return &proto.ActionStatus{Success: false}, errors.New("Cannot request action from non-leader control plane.")
	}

	return c.ControlPlane.NodeHeartbeat(ctx, in)
}

func (c *CpApiServer) RegisterService(ctx context.Context, serviceInfo *proto.ServiceInfo) (*proto.ActionStatus, error) {
	if !c.ControlPlane.LeaderElectionServer.IsLeader() {
		return &proto.ActionStatus{Success: false}, errors.New("Cannot request action from non-leader control plane.")
	}

	return c.ControlPlane.RegisterService(ctx, serviceInfo)
}

func (c *CpApiServer) DeregisterService(ctx context.Context, serviceInfo *proto.ServiceInfo) (*proto.ActionStatus, error) {
	if !c.ControlPlane.LeaderElectionServer.IsLeader() {
		return &proto.ActionStatus{Success: false}, errors.New("Cannot request action from non-leader control plane.")
	}

	return c.ControlPlane.DeregisterService(ctx, serviceInfo)
}

func (c *CpApiServer) RegisterDataplane(ctx context.Context, in *proto.DataplaneInfo) (*proto.ActionStatus, error) {
	if !c.ControlPlane.LeaderElectionServer.IsLeader() {
		return &proto.ActionStatus{Success: false}, errors.New("Cannot request action from non-leader control plane.")
	}

	return c.ControlPlane.RegisterDataplane(ctx, in)
}

func (c *CpApiServer) DeregisterDataplane(ctx context.Context, in *proto.DataplaneInfo) (*proto.ActionStatus, error) {
	if !c.ControlPlane.LeaderElectionServer.IsLeader() {
		return &proto.ActionStatus{Success: false}, errors.New("Cannot request action from non-leader control plane.")
	}

	return c.ControlPlane.DeregisterDataplane(ctx, in)
}

func (c *CpApiServer) ReconstructState(ctx context.Context, config config2.ControlPlaneConfig) error {
	if !c.ControlPlane.LeaderElectionServer.IsLeader() {
		return errors.New("Cannot request action from non-leader control plane.")
	}

	return c.ControlPlane.ReconstructState(ctx, config)
}

func (c *CpApiServer) ResetMeasurements(_ context.Context, _ *emptypb.Empty) (*proto.ActionStatus, error) {
	if !c.ControlPlane.LeaderElectionServer.IsLeader() {
		return &proto.ActionStatus{Success: false}, errors.New("Cannot request action from non-leader control plane.")
	}

	c.ControlPlane.ColdStartTracing.ResetTracingService()
	return &proto.ActionStatus{Success: true}, nil
}

func (c *CpApiServer) ReportFailure(_ context.Context, in *proto.Failure) (*proto.ActionStatus, error) {
	if !c.ControlPlane.LeaderElectionServer.IsLeader() {
		return &proto.ActionStatus{Success: false}, errors.New("Cannot request action from non-leader control plane.")
	}

	return &proto.ActionStatus{Success: c.ControlPlane.HandleFailure([]*proto.Failure{in})}, nil
}

func (c *CpApiServer) RequestVote(_ context.Context, args *proto.RequestVoteArgs) (*proto.RequestVoteReply, error) {
	return c.ControlPlane.LeaderElectionServer.RequestVote(args)
}

func (c *CpApiServer) AppendEntries(_ context.Context, args *proto.AppendEntriesArgs) (*proto.AppendEntriesReply, error) {
	return c.ControlPlane.LeaderElectionServer.AppendEntries(args)
}

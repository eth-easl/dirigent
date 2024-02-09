package api

import (
	"cluster_manager/api/proto"
	"cluster_manager/internal/control_plane"
	"cluster_manager/internal/control_plane/core"
	"cluster_manager/internal/control_plane/leader_election"
	"cluster_manager/internal/control_plane/persistence"
	"cluster_manager/internal/control_plane/placement_policy"
	config2 "cluster_manager/pkg/config"
	"cluster_manager/pkg/grpc_helpers"
	"context"
	"errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"net"
	"strconv"
)

type CpApiServer struct {
	proto.UnimplementedCpiInterfaceServer

	LeaderElectionServer *leader_election.LeaderElectionServer
	ControlPlane         *control_plane.ControlPlane
}

type CpApiServerCreationArguments struct {
	Client            persistence.PersistenceLayer
	OutputFile        string
	PlacementPolicy   placement_policy.PlacementPolicy
	DataplaneCreator  core.DataplaneFactory
	WorkerNodeCreator core.WorkerNodeFactory
	Cfg               *config2.ControlPlaneConfig
}

func CreateNewCpApiServer(args *CpApiServerCreationArguments) (*CpApiServer, chan bool) {
	cp := control_plane.NewControlPlane(
		args.Client,
		args.OutputFile,
		args.PlacementPolicy,
		args.DataplaneCreator,
		args.WorkerNodeCreator,
		args.Cfg,
	)

	readyToElect := make(chan interface{})
	if len(args.Cfg.Replicas) > 0 {
		defer close(readyToElect)
	}

	port, _ := strconv.Atoi(args.Cfg.Port)
	leaderElectionServer, isLeader := leader_election.NewServer(
		int32(port),
		grpc_helpers.ParseReplicaPorts(args.Cfg),
		readyToElect,
	)

	cpApiServer := &CpApiServer{
		LeaderElectionServer: leaderElectionServer,
		ControlPlane:         cp,
	}

	go grpc_helpers.CreateGRPCServer(args.Cfg.Port, func(sr grpc.ServiceRegistrar) {
		proto.RegisterCpiInterfaceServer(sr, cpApiServer)
	})

	// connecting to peers for leader election
	for _, rawAddress := range args.Cfg.Replicas {
		peerID := grpc_helpers.GetPeerPort(rawAddress)
		tcpAddr, _ := net.ResolveTCPAddr("tcp", rawAddress)

		leaderElectionServer.ConnectToPeer(peerID, tcpAddr)
	}

	return cpApiServer, isLeader
}

func (c *CpApiServer) CleanControlPlaneInMemoryData(args *CpApiServerCreationArguments) {
	c.ControlPlane = control_plane.NewControlPlane(
		args.Client,
		args.OutputFile,
		args.PlacementPolicy,
		args.DataplaneCreator,
		args.WorkerNodeCreator,
		args.Cfg,
	)
}

func (c *CpApiServer) StartNodeMonitoringLoop(stopCh chan struct{}) {
	if !c.LeaderElectionServer.IsLeader() {
		logrus.Errorf("Cannot start node monitoring loop as this " +
			"instance of control plane is currently not the leader. " +
			"Probably lost leadership in the meanwhile.")

		return
	}

	go c.ControlPlane.CheckPeriodicallyWorkerNodes(stopCh)
}

func (c *CpApiServer) OnMetricsReceive(ctx context.Context, metric *proto.AutoscalingMetric) (*proto.ActionStatus, error) {
	if !c.LeaderElectionServer.IsLeader() {
		logrus.Warn("Received OnMetricsReceive call although not the leader. Forwarding the call...")
		return c.LeaderElectionServer.GetLeader().OnMetricsReceive(ctx, metric)
	}

	return c.ControlPlane.OnMetricsReceive(ctx, metric)
}

func (c *CpApiServer) ListServices(ctx context.Context, empty *emptypb.Empty) (*proto.ServiceList, error) {
	if !c.LeaderElectionServer.IsLeader() {
		logrus.Warn("Received ListServices call although not the leader. Forwarding the call...")
		return c.LeaderElectionServer.GetLeader().ListServices(ctx, empty)
	}

	return c.ControlPlane.ListServices(ctx, empty)
}

func (c *CpApiServer) RegisterNode(ctx context.Context, info *proto.NodeInfo) (*proto.ActionStatus, error) {
	if !c.LeaderElectionServer.IsLeader() {
		logrus.Warn("Received RegisterNode call although not the leader. Forwarding the call...")
		return c.LeaderElectionServer.GetLeader().RegisterNode(ctx, info)
	}

	return c.ControlPlane.RegisterNode(ctx, info)
}

func (c *CpApiServer) DeregisterNode(ctx context.Context, info *proto.NodeInfo) (*proto.ActionStatus, error) {
	if !c.LeaderElectionServer.IsLeader() {
		logrus.Warn("Received DeregisterNode call although not the leader. Forwarding the call...")
		return c.LeaderElectionServer.GetLeader().DeregisterNode(ctx, info)
	}

	return c.ControlPlane.DeregisterNode(ctx, info)
}

func (c *CpApiServer) NodeHeartbeat(ctx context.Context, in *proto.NodeHeartbeatMessage) (*proto.ActionStatus, error) {
	if !c.LeaderElectionServer.IsLeader() {
		logrus.Warn("Received NodeHeartbeat call although not the leader. Forwarding the call...")
		return c.LeaderElectionServer.GetLeader().NodeHeartbeat(ctx, in)
	}

	return c.ControlPlane.NodeHeartbeat(ctx, in)
}

func (c *CpApiServer) RegisterService(ctx context.Context, serviceInfo *proto.ServiceInfo) (*proto.ActionStatus, error) {
	if !c.LeaderElectionServer.IsLeader() {
		logrus.Warn("Received RegisterService call although not the leader. Forwarding the call...")
		return c.LeaderElectionServer.GetLeader().RegisterService(ctx, serviceInfo)
	}

	return c.ControlPlane.RegisterService(ctx, serviceInfo)
}

func (c *CpApiServer) DeregisterService(ctx context.Context, serviceInfo *proto.ServiceInfo) (*proto.ActionStatus, error) {
	if !c.LeaderElectionServer.IsLeader() {
		logrus.Warn("Received DeregisterService call although not the leader. Forwarding the call...")
		return c.LeaderElectionServer.GetLeader().DeregisterService(ctx, serviceInfo)
	}

	return c.ControlPlane.DeregisterService(ctx, serviceInfo)
}

func (c *CpApiServer) RegisterDataplane(ctx context.Context, in *proto.DataplaneInfo) (*proto.ActionStatus, error) {
	if !c.LeaderElectionServer.IsLeader() {
		logrus.Warn("Received RegisterDataplane call although not the leader. Forwarding the call...")
		return c.LeaderElectionServer.GetLeader().RegisterDataplane(ctx, in)
	}

	return c.ControlPlane.RegisterDataplane(ctx, in)
}

func (c *CpApiServer) DeregisterDataplane(ctx context.Context, in *proto.DataplaneInfo) (*proto.ActionStatus, error) {
	if !c.LeaderElectionServer.IsLeader() {
		logrus.Warn("Received DeregisterDataplane call although not the leader. Forwarding the call...")
		return c.LeaderElectionServer.GetLeader().DeregisterDataplane(ctx, in)
	}

	return c.ControlPlane.DeregisterDataplane(ctx, in)
}

func (c *CpApiServer) ReconstructState(ctx context.Context, config config2.ControlPlaneConfig) error {
	if !c.LeaderElectionServer.IsLeader() {
		// This API call is not exposed to the outside, but it's called only on process startup
		return errors.New("cannot request cluster state reconstruction if not the leader. " +
			"Perhaps the leader has changed in the meanwhile")
	}

	return c.ControlPlane.ReconstructState(ctx, config)
}

func (c *CpApiServer) ResetMeasurements(ctx context.Context, empty *emptypb.Empty) (*proto.ActionStatus, error) {
	if !c.LeaderElectionServer.IsLeader() {
		logrus.Warn("Received ResetMeasurements call although not the leader. Forwarding the call...")
		return c.LeaderElectionServer.GetLeader().ResetMeasurements(ctx, empty)
	}

	c.ControlPlane.ColdStartTracing.ResetTracingService()
	return &proto.ActionStatus{Success: true}, nil
}

func (c *CpApiServer) ReportFailure(ctx context.Context, in *proto.Failure) (*proto.ActionStatus, error) {
	if !c.LeaderElectionServer.IsLeader() {
		logrus.Warn("Received ReportFailure call although not the leader. Forwarding the call...")
		return c.LeaderElectionServer.GetLeader().ReportFailure(ctx, in)
	}

	return &proto.ActionStatus{Success: c.ControlPlane.HandleFailure([]*proto.Failure{in})}, nil
}

func (c *CpApiServer) RequestVote(_ context.Context, args *proto.RequestVoteArgs) (*proto.RequestVoteReply, error) {
	// Leader election call -> Should not be forwarded to any other node.
	return c.LeaderElectionServer.RequestVote(args)
}

func (c *CpApiServer) AppendEntries(_ context.Context, args *proto.AppendEntriesArgs) (*proto.AppendEntriesReply, error) {
	// Leader election call -> Should not be forwarded to any other node.
	return c.LeaderElectionServer.AppendEntries(args)
}

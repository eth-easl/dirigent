/*
 * MIT License
 *
 * Copyright (c) 2024 EASL
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */

package api

import (
	"cluster_manager/api/proto"
	"cluster_manager/internal/control_plane"
	"cluster_manager/internal/control_plane/core"
	"cluster_manager/internal/control_plane/leader_election"
	"cluster_manager/internal/control_plane/persistence"
	"cluster_manager/internal/control_plane/placement_policy"
	"cluster_manager/internal/data_plane/haproxy"
	config2 "cluster_manager/pkg/config"
	"cluster_manager/pkg/grpc_helpers"
	"context"
	"errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"strconv"
)

type CpApiServer struct {
	proto.UnimplementedCpiInterfaceServer

	LeaderElectionServer *leader_election.LeaderElectionServer
	ControlPlane         *control_plane.ControlPlane
	HAProxyAPI           *haproxy.API
}

type CpApiServerCreationArguments struct {
	Client            persistence.PersistenceLayer
	OutputFile        string
	PlacementPolicy   placement_policy.PlacementPolicy
	DataplaneCreator  core.DataplaneFactory
	WorkerNodeCreator core.WorkerNodeFactory
	Cfg               *config2.ControlPlaneConfig
}

func CreateNewCpApiServer(args *CpApiServerCreationArguments) (*CpApiServer, chan leader_election.AnnounceLeadership) {
	cp := control_plane.NewControlPlane(
		args.Client,
		args.OutputFile,
		args.PlacementPolicy,
		args.DataplaneCreator,
		args.WorkerNodeCreator,
		args.Cfg,
	)

	readyToElect := make(chan interface{})
	port, _ := strconv.Atoi(args.Cfg.Port)
	leaderElectionServer, isLeader := leader_election.NewServer(
		int32(port),
		grpc_helpers.ParseReplicaPorts(args.Cfg),
		readyToElect,
	)

	cpApiServer := &CpApiServer{
		LeaderElectionServer: leaderElectionServer,
		ControlPlane:         cp,
		HAProxyAPI:           haproxy.NewHAProxyAPI(args.Cfg.LoadBalancerAddress),
	}

	go grpc_helpers.CreateGRPCServer(args.Cfg.Port, func(sr grpc.ServiceRegistrar) {
		proto.RegisterCpiInterfaceServer(sr, cpApiServer)
	})

	// connecting to peers for leader election (at least half of them to become ready)
	if len(args.Cfg.Replicas) > 0 {
		logrus.Infof("Trying to establish connection with other control plane replicas for leader election...")

		cpApiServer.LeaderElectionServer.EstablishLeaderElectionMesh(args.Cfg.Replicas)
		close(readyToElect)
	}

	return cpApiServer, isLeader
}

func (c *CpApiServer) CleanControlPlaneInMemoryData(args *CpApiServerCreationArguments) {
	// there might be some scaling loops we need to stop to prevent resource leaks
	c.ControlPlane.StopAllScalingLoops()

	c.ControlPlane = control_plane.NewControlPlane(
		args.Client,
		args.OutputFile,
		args.PlacementPolicy,
		args.DataplaneCreator,
		args.WorkerNodeCreator,
		args.Cfg,
	)
}

func (c *CpApiServer) StartNodeMonitoringLoop() chan struct{} {
	if !c.LeaderElectionServer.IsLeader() {
		logrus.Errorf("Cannot start node monitoring loop as this " +
			"instance of control plane is currently not the leader. " +
			"Probably lost leadership in the meanwhile.")

		return nil
	}

	return c.ControlPlane.StartNodeMonitoring()
}

func (c *CpApiServer) OnMetricsReceive(ctx context.Context, metric *proto.AutoscalingMetric) (*proto.ActionStatus, error) {
	if !c.LeaderElectionServer.IsLeader() {
		leader := c.LeaderElectionServer.GetLeader()

		if leader != nil {
			logrus.Warn("Received OnMetricsReceive call although not the leader. Forwarding the call...")
			return leader.OnMetricsReceive(ctx, metric)
		} else {
			return &proto.ActionStatus{Success: false}, nil
		}
	}

	return c.ControlPlane.OnMetricsReceive(ctx, metric)
}

func (c *CpApiServer) ListServices(ctx context.Context, empty *emptypb.Empty) (*proto.ServiceList, error) {
	if !c.LeaderElectionServer.IsLeader() {
		logrus.Warn("Received ListServices call although not the leader. Forwarding the call...")
		leader := c.LeaderElectionServer.GetLeader()

		if leader != nil {
			return leader.ListServices(ctx, empty)
		} else {
			return &proto.ServiceList{}, nil
		}
	}

	return c.ControlPlane.ListServices(ctx, empty)
}

func (c *CpApiServer) RegisterNode(ctx context.Context, info *proto.NodeInfo) (*proto.ActionStatus, error) {
	if !c.LeaderElectionServer.IsLeader() {
		leader := c.LeaderElectionServer.GetLeader()

		if leader != nil {
			logrus.Warn("Received RegisterNode call although not the leader. Forwarding the call...")
			return leader.RegisterNode(ctx, info)
		} else {
			return &proto.ActionStatus{Success: false}, nil
		}
	}

	return c.ControlPlane.RegisterNode(ctx, info)
}

func (c *CpApiServer) DeregisterNode(ctx context.Context, info *proto.NodeInfo) (*proto.ActionStatus, error) {
	if !c.LeaderElectionServer.IsLeader() {
		leader := c.LeaderElectionServer.GetLeader()

		if leader != nil {
			logrus.Warn("Received DeregisterNode call although not the leader. Forwarding the call...")
			return leader.DeregisterNode(ctx, info)
		} else {
			return &proto.ActionStatus{Success: false}, nil
		}
	}

	return c.ControlPlane.DeregisterNode(ctx, info)
}

func (c *CpApiServer) NodeHeartbeat(ctx context.Context, in *proto.NodeHeartbeatMessage) (*proto.ActionStatus, error) {
	if !c.LeaderElectionServer.IsLeader() {
		leader := c.LeaderElectionServer.GetLeader()

		if leader != nil {
			logrus.Warn("Received NodeHeartbeat call although not the leader. Forwarding the call...")
			return leader.NodeHeartbeat(ctx, in)
		} else {
			return &proto.ActionStatus{Success: false}, nil
		}
	}

	return c.ControlPlane.NodeHeartbeat(ctx, in)
}

func (c *CpApiServer) RegisterService(ctx context.Context, serviceInfo *proto.ServiceInfo) (*proto.ActionStatus, error) {
	if !c.LeaderElectionServer.IsLeader() {
		leader := c.LeaderElectionServer.GetLeader()

		if leader != nil {
			logrus.Warn("Received RegisterService call although not the leader. Forwarding the call...")
			return leader.RegisterService(ctx, serviceInfo)
		} else {
			return &proto.ActionStatus{Success: false}, nil
		}
	}

	return c.ControlPlane.RegisterService(ctx, serviceInfo)
}

func (c *CpApiServer) DeregisterService(ctx context.Context, serviceInfo *proto.ServiceInfo) (*proto.ActionStatus, error) {
	if !c.LeaderElectionServer.IsLeader() {
		leader := c.LeaderElectionServer.GetLeader()

		if leader != nil {
			logrus.Warn("Received DeregisterService call although not the leader. Forwarding the call...")
			return leader.DeregisterService(ctx, serviceInfo)
		} else {
			return &proto.ActionStatus{Success: false}, nil
		}
	}

	return c.ControlPlane.DeregisterService(ctx, serviceInfo)
}

func (c *CpApiServer) RegisterDataplane(ctx context.Context, in *proto.DataplaneInfo) (*proto.ActionStatus, error) {
	if !c.LeaderElectionServer.IsLeader() {
		leader := c.LeaderElectionServer.GetLeader()

		if leader != nil {
			logrus.Warn("Received RegisterDataplane call although not the leader. Forwarding the call...")
			return leader.RegisterDataplane(ctx, in)
		} else {
			return &proto.ActionStatus{Success: false}, nil
		}
	}

	status, err, isHeartbeat := c.ControlPlane.RegisterDataplane(ctx, in)
	if status.Success && err == nil && !isHeartbeat {
		c.HAProxyAPI.AddDataplane(in.IP, int(in.ProxyPort), true)
		c.DisseminateHAProxyConfig()
	}

	return status, err
}

func (c *CpApiServer) DeregisterDataplane(ctx context.Context, in *proto.DataplaneInfo) (*proto.ActionStatus, error) {
	if !c.LeaderElectionServer.IsLeader() {
		leader := c.LeaderElectionServer.GetLeader()

		if leader != nil {
			logrus.Warn("Received DeregisterDataplane call although not the leader. Forwarding the call...")
			return leader.DeregisterDataplane(ctx, in)
		} else {
			return &proto.ActionStatus{Success: false}, nil
		}
	}

	status, err := c.ControlPlane.DeregisterDataplane(ctx, in)
	if status.Success && err == nil {
		c.HAProxyAPI.RemoveDataplane(in.IP, int(in.ProxyPort), true)
		c.DisseminateHAProxyConfig()
	}

	return status, err
}

func (c *CpApiServer) ReconstructState(ctx context.Context, config config2.ControlPlaneConfig) error {
	if !c.LeaderElectionServer.IsLeader() {
		// This API call is not exposed to the outside, but it's called only on process startup
		return errors.New("cannot request cluster state reconstruction if not the leader. " +
			"Perhaps the leader has changed in the meanwhile")
	}

	return c.ControlPlane.ReconstructState(ctx, config, c.HAProxyAPI)
}

func (c *CpApiServer) ResetMeasurements(ctx context.Context, empty *emptypb.Empty) (*proto.ActionStatus, error) {
	if !c.LeaderElectionServer.IsLeader() {
		leader := c.LeaderElectionServer.GetLeader()

		if leader != nil {
			logrus.Warn("Received ResetMeasurements call although not the leader. Forwarding the call...")
			return leader.ResetMeasurements(ctx, empty)
		} else {
			return &proto.ActionStatus{Success: false}, nil
		}
	}

	c.ControlPlane.ColdStartTracing.ResetTracingService()
	return &proto.ActionStatus{Success: true}, nil
}

func (c *CpApiServer) ReportFailure(ctx context.Context, in *proto.Failure) (*proto.ActionStatus, error) {
	if !c.LeaderElectionServer.IsLeader() {
		leader := c.LeaderElectionServer.GetLeader()

		if leader != nil {
			logrus.Warn("Received ReportFailure call although not the leader. Forwarding the call...")
			return leader.ReportFailure(ctx, in)
		} else {
			return &proto.ActionStatus{Success: false}, nil
		}
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

// ReviseHAProxyConfiguration Local method for updating HAProxy configuration. Called by the remote leader to disseminate configuration.
func (c *CpApiServer) ReviseHAProxyConfiguration(_ context.Context, args *proto.HAProxyConfig) (*proto.ActionStatus, error) {
	return c.HAProxyAPI.ReviseHAProxyConfiguration(args)
}

func (c *CpApiServer) DisseminateHAProxyConfig() {
	haproxy.DisseminateHAProxyConfig(c.ControlPlane.GetHAProxyConfig(), c.LeaderElectionServer)
}

package leader_election

import (
	"cluster_manager/api/proto"
	"cluster_manager/mock/mock_cp_api"
	"context"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc"
	"sync"
	"testing"
	"time"
)

func TestNotNil(t *testing.T) {
	leaderShipChannel := make(chan AnnounceLeadership, 1)
	assert.NotNil(t, NewConsensusModule(1, make([]int, 0), nil, make(chan interface{}), leaderShipChannel))
}

func TestReport(t *testing.T) {
	module := NewConsensusModule(1, make([]int, 0), nil, make(chan interface{}), make(chan AnnounceLeadership, 1))

	id, _, isLeader := module.Report()

	assert.True(t, isLeader)
	assert.Equal(t, int32(1), id)
}

func TestStopModule(t *testing.T) {
	module := NewConsensusModule(1, make([]int, 0), nil, make(chan interface{}), make(chan AnnounceLeadership, 1))
	assert.NotNil(t, module)

	module.Stop()

	assert.Equal(t, module.state, Dead)
}

func TestIsLeader(t *testing.T) {
	assert.True(t, NewConsensusModule(1, make([]int, 0), nil, make(chan interface{}), make(chan AnnounceLeadership, 1)).IsLeader())
}

func TestIsFollower(t *testing.T) {
	module := NewConsensusModule(1, make([]int, 0), nil, make(chan interface{}), make(chan AnnounceLeadership, 2))

	module.becomeFollower(1)

	assert.False(t, module.IsLeader())
}

func TestGetLeaderId(t *testing.T) {
	module := NewConsensusModule(1, make([]int, 0), nil, make(chan interface{}), make(chan AnnounceLeadership, 1))

	assert.Zero(t, module.GetLeaderID())
}

func TestSimulateSuccessfulElection(t *testing.T) {
	peerClients := make(map[int]proto.CpiInterfaceClient)

	for i := 1; i <= 5; i++ {
		ctrl := gomock.NewController(t)

		peerClient := mock_cp_api.NewMockCpiInterfaceClient(ctrl)

		peerClient.EXPECT().RequestVote(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, _ *proto.RequestVoteArgs, opts ...grpc.CallOption) (*proto.RequestVoteReply, error) {
			return &proto.RequestVoteReply{
				Term:        1,
				VoteGranted: true,
			}, nil
		}).Times(1)

		peerClient.EXPECT().AppendEntries(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, in *proto.AppendEntriesArgs, opts ...grpc.CallOption) (*proto.AppendEntriesReply, error) {
			return &proto.AppendEntriesReply{
				Term:    1,
				Success: true,
			}, nil
		}).AnyTimes()

		peerClients[i] = peerClient
	}

	module := NewConsensusModule(0, []int{1, 2, 3, 4, 5}, &LeaderElectionServer{
		mu:                 sync.Mutex{},
		serverId:           0,
		peerIds:            nil,
		cm:                 nil,
		rpcServer:          nil,
		listener:           nil,
		peerClients:        peerClients,
		ready:              nil,
		quit:               nil,
		wg:                 sync.WaitGroup{},
		announceLeadership: nil,
	}, make(chan interface{}), make(chan AnnounceLeadership, 1))

	module.startElection()

	assert.Equal(t, module.state, Candidate)

	time.Sleep(time.Second)

	assert.True(t, module.IsLeader())
}

func TestSimulateUnsuccessfulelection(t *testing.T) {
	peerClients := make(map[int]proto.CpiInterfaceClient)

	for i := 1; i <= 5; i++ {
		ctrl := gomock.NewController(t)

		peerClient := mock_cp_api.NewMockCpiInterfaceClient(ctrl)

		peerClient.EXPECT().RequestVote(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, _ *proto.RequestVoteArgs, opts ...grpc.CallOption) (*proto.RequestVoteReply, error) {
			return &proto.RequestVoteReply{
				Term:        1,
				VoteGranted: false,
			}, nil
		}).AnyTimes()

		peerClient.EXPECT().AppendEntries(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, in *proto.AppendEntriesArgs, opts ...grpc.CallOption) (*proto.AppendEntriesReply, error) {
			return &proto.AppendEntriesReply{
				Term:    1,
				Success: false,
			}, nil
		}).AnyTimes()

		peerClients[i] = peerClient
	}

	module := NewConsensusModule(0, []int{1, 2, 3, 4, 5}, &LeaderElectionServer{
		mu:                 sync.Mutex{},
		serverId:           0,
		peerIds:            nil,
		cm:                 nil,
		rpcServer:          nil,
		listener:           nil,
		peerClients:        peerClients,
		ready:              nil,
		quit:               nil,
		wg:                 sync.WaitGroup{},
		announceLeadership: nil,
	}, make(chan interface{}), make(chan AnnounceLeadership, 1))

	module.startElection()

	assert.Equal(t, module.state, Candidate)

	time.Sleep(time.Second)

	assert.False(t, module.IsLeader())
}
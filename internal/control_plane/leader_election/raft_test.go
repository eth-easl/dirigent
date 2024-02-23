package leader_election

import (
	"github.com/stretchr/testify/assert"
	"testing"
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

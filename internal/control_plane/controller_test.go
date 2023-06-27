package control_plane

import (
	"cluster_manager/internal/algorithms/placement"
	"cluster_manager/internal/common"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRandomPolicy(t *testing.T) {
	policy := placement.RANDOM
	storage := &NodeInfoStorage{
		NodeInfo: make(map[string]*WorkerNode),
	}

	storage.NodeInfo["w1"] = &WorkerNode{}
	storage.NodeInfo["w2"] = &WorkerNode{}

	requested := &placement.ResourceMap{}

	for i := 0; i < 100; i++ {
		currentStorage := placementPolicy(policy, storage, requested)
		assert.NotNil(t, currentStorage)
		assert.True(t, currentStorage == storage.NodeInfo["w1"] || currentStorage == storage.NodeInfo["w2"])
	}
}

func TestRoundRobin(t *testing.T) {
	policy := placement.ROUND_ROBIN
	storage := &NodeInfoStorage{
		NodeInfo: make(map[string]*WorkerNode),
	}

	requested := &placement.ResourceMap{}

	storage.NodeInfo["w1"] = &WorkerNode{}
	storage.NodeInfo["w2"] = &WorkerNode{}
	storage.NodeInfo["w3"] = &WorkerNode{}

	nodes := sort.StringSlice(common.Keys(storage.NodeInfo))
	nodes.Sort()

	for i := 0; i < 100; i++ {
		{
			currentStorage := placementPolicy(policy, storage, requested)
			assert.NotNil(t, currentStorage)
			assert.True(t, currentStorage == storage.NodeInfo[nodes[0]])
		}
		{
			currentStorage := placementPolicy(policy, storage, requested)
			assert.NotNil(t, currentStorage)
			assert.True(t, currentStorage == storage.NodeInfo[nodes[1]])
		}
		{
			currentStorage := placementPolicy(policy, storage, requested)
			assert.NotNil(t, currentStorage)
			assert.True(t, currentStorage == storage.NodeInfo[nodes[2]])
		}
	}
}

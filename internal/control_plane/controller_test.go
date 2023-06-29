package control_plane

import (
	"cluster_manager/internal/common"
	placement2 "cluster_manager/internal/control_plane/placement"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRandomPolicy(t *testing.T) {
	policy := placement2.PLACMENT_RANDOM
	storage := &NodeInfoStorage{
		NodeInfo: make(map[string]*WorkerNode),
	}

	storage.NodeInfo["w1"] = &WorkerNode{}
	storage.NodeInfo["w2"] = &WorkerNode{}

	requested := &placement2.ResourceMap{}

	for i := 0; i < 100; i++ {
		currentStorage := placementPolicy(policy, storage, requested)
		assert.NotNil(t, currentStorage)
		assert.True(t, currentStorage == storage.NodeInfo["w1"] || currentStorage == storage.NodeInfo["w2"])
	}
}

func TestRoundRobin(t *testing.T) {
	policy := placement2.PLACEMENT_ROUND_ROBIN
	storage := &NodeInfoStorage{
		NodeInfo: make(map[string]*WorkerNode),
	}

	requested := &placement2.ResourceMap{}

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

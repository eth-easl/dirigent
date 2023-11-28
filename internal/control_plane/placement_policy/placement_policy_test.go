package placement_policy

import (
	"cluster_manager/internal/control_plane/core"
	"cluster_manager/internal/control_plane/workers"
	_map "cluster_manager/pkg/map"
	"cluster_manager/pkg/synchronization"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRandomPolicy(t *testing.T) {
	policy := NewRandomPlacement()
	storage := synchronization.NewControlPlaneSyncStructure[string, core.WorkerNodeInterface]()

	storage.Set("w1", &workers.WorkerNode{Schedulable: true})
	storage.Set("w2", &workers.WorkerNode{Schedulable: true})

	requested := &ResourceMap{}

	for i := 0; i < 100; i++ {
		currentStorage := policy.Place(storage, requested)
		assert.NotNil(t, currentStorage)
		assert.True(t, currentStorage == storage.GetNoCheck("w1") || currentStorage == storage.GetNoCheck("w2"))
	}
}

func TestRoundRobin(t *testing.T) {
	policy := NewRoundRobinPlacement()
	storage := synchronization.NewControlPlaneSyncStructure[string, core.WorkerNodeInterface]()

	requested := &ResourceMap{}

	storage.Set("w1", &workers.WorkerNode{Schedulable: true})
	storage.Set("w2", &workers.WorkerNode{Schedulable: true})
	storage.Set("w3", &workers.WorkerNode{Schedulable: true})

	nodes := sort.StringSlice(_map.Keys(storage.GetMap()))
	nodes.Sort()

	for i := 0; i < 100; i++ {
		{
			currentStorage := policy.Place(storage, requested)
			assert.NotNil(t, currentStorage)
			assert.Equal(t, currentStorage, storage.GetNoCheck(nodes[0]))
		}
		{
			currentStorage := policy.Place(storage, requested)
			assert.NotNil(t, currentStorage)
			assert.Equal(t, currentStorage, storage.GetNoCheck(nodes[1]))
		}
		{
			currentStorage := policy.Place(storage, requested)
			assert.NotNil(t, currentStorage)
			assert.Equal(t, currentStorage, storage.GetNoCheck(nodes[2]))
		}
	}
}

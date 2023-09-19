package placement_policy

import (
	"cluster_manager/internal/control_plane/core"
	"cluster_manager/internal/control_plane/workers"
	"cluster_manager/pkg/atomic_map"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRandomPolicy(t *testing.T) {
	policy := NewRandomPolicy()
	storage := atomic_map.NewAtomicMap[string, core.WorkerNodeInterface]()

	storage.Set("w1", &workers.WorkerNode{})
	storage.Set("w2", &workers.WorkerNode{})

	requested := &ResourceMap{}

	for i := 0; i < 100; i++ {
		currentStorage := policy.Place(storage, requested)
		assert.NotNil(t, currentStorage)
		assert.True(t, currentStorage == storage.GetUnsafe("w1") || currentStorage == storage.GetUnsafe("w2"))
	}
}

func TestRoundRobin(t *testing.T) {
	policy := NewRoundRobinPolicy()
	storage := atomic_map.NewAtomicMap[string, core.WorkerNodeInterface]()

	requested := &ResourceMap{}

	storage.Set("w1", &workers.WorkerNode{})
	storage.Set("w2", &workers.WorkerNode{})
	storage.Set("w3", &workers.WorkerNode{})

	nodes := sort.StringSlice(storage.Keys())
	nodes.Sort()

	for i := 0; i < 100; i++ {
		{
			currentStorage := policy.Place(storage, requested)
			assert.NotNil(t, currentStorage)
			assert.Equal(t, currentStorage, storage.GetUnsafe(nodes[0]))
		}
		{
			currentStorage := policy.Place(storage, requested)
			assert.NotNil(t, currentStorage)
			assert.Equal(t, currentStorage, storage.GetUnsafe(nodes[1]))
		}
		{
			currentStorage := policy.Place(storage, requested)
			assert.NotNil(t, currentStorage)
			assert.Equal(t, currentStorage, storage.GetUnsafe(nodes[2]))
		}
	}
}

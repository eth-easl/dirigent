package placement_policy

import (
	"cluster_manager/internal/control_plane/control_plane/core"
	"cluster_manager/internal/control_plane/control_plane/workers"
	_map "cluster_manager/pkg/map"
	"cluster_manager/pkg/synchronization"
	"fmt"
	"github.com/montanaflynn/stats"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"sort"
	"sync"
	"testing"
	"time"
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

func TestMinimumNumberOfNodesToFilter(t *testing.T) {
	tests := []struct {
		name     string
		storage  synchronization.SyncStructure[string, core.WorkerNodeInterface]
		expected int
	}{
		{
			name:     "filter_50",
			storage:  createStorageWithXNodes(50),
			expected: 50,
		},
		{
			name:     "filter_100",
			storage:  createStorageWithXNodes(100),
			expected: 100,
		},
		{
			name:     "filter_150",
			storage:  createStorageWithXNodes(150),
			expected: 100,
		},
		{
			name:     "filter_250",
			storage:  createStorageWithXNodes(250),
			expected: 120,
		},
		{
			name:     "filter_1000",
			storage:  createStorageWithXNodes(1000),
			expected: 420,
		},
		{
			name:     "filter_2500",
			storage:  createStorageWithXNodes(2500),
			expected: 750,
		},
		{
			name:     "filter_10000",
			storage:  createStorageWithXNodes(10000),
			expected: 500,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := minimumNumberOfNodesToFilter(test.storage)
			if got != test.expected {
				t.Error("Got:", got, "Expected:", test.expected)
			}
		})
	}
}

func createStorageWithXNodes(n int) synchronization.SyncStructure[string, core.WorkerNodeInterface] {
	storage := synchronization.NewControlPlaneSyncStructure[string, core.WorkerNodeInterface]()

	for i := 0; i < n; i++ {
		storage.Set(fmt.Sprintf("w%d", i), &workers.WorkerNode{
			Schedulable: true,
			CpuCores:    10_000,
			Memory:      65536,
		})
	}

	return storage
}

func TestPlacementOnXKNodes(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)

	NODES := 1000
	ITERATIONS := 10
	PARALLELISM := 1

	// create cluster
	storage := createStorageWithXNodes(NODES)
	policy := NewKubernetesPolicy()

	var data []float64
	wg := sync.WaitGroup{}
	mutex := sync.Mutex{}

	for i := 0; i < ITERATIONS; i++ {
		wg.Add(PARALLELISM)

		for p := 0; p < PARALLELISM; p++ {
			go func() {
				defer wg.Done()

				start := time.Now()
				policy.Place(storage, CreateResourceMap(1, 1))
				dt := time.Since(start).Milliseconds()

				mutex.Lock()
				data = append(data, float64(dt))
				mutex.Unlock()

				logrus.Tracef("Scheduling on a %d-node cluster took %d ms", NODES, dt)
			}()
		}

		wg.Wait()
	}

	p50, _ := stats.Percentile(data, 50)
	p99, _ := stats.Percentile(data, 99)
	logrus.Debugf("p50: %.2f; p99: %.2f", p50, p99)
}

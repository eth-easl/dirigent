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

func TestPlacementOnXKNodes(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)

	policy := NewKubernetesPolicy()
	storage := synchronization.NewControlPlaneSyncStructure[string, core.WorkerNodeInterface]()

	NODES := 1000
	ITERATIONS := 10
	PARALLELISM := 1

	// create cluster
	for i := 0; i < NODES; i++ {
		storage.Set(fmt.Sprintf("w%d", i), &workers.WorkerNode{
			Schedulable: true,
			CpuCores:    10_000,
			Memory:      65536,
		})
	}

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

				logrus.Tracef("Scheduling on a 5K-node cluster took %d ms", dt)
			}()
		}

		wg.Wait()
	}

	p50, _ := stats.Percentile(data, 50)
	p99, _ := stats.Percentile(data, 99)
	logrus.Debugf("p50: %.2f; p99: %.2f", p50, p99)
}

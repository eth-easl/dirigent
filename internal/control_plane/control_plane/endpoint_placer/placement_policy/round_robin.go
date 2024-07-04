package placement_policy

import (
	"cluster_manager/internal/control_plane/control_plane/core"
	"cluster_manager/internal/control_plane/control_plane/image_storage"
	"cluster_manager/pkg/synchronization"
	"sync/atomic"
)

type RoundRobinPlacement struct {
	schedulingCounterRoundRobin int32
}

func NewRoundRobinPlacement() *RoundRobinPlacement {
	return &RoundRobinPlacement{
		schedulingCounterRoundRobin: 0,
	}
}

func (policy *RoundRobinPlacement) Place(storage synchronization.SyncStructure[string, core.WorkerNodeInterface], _ image_storage.ImageStorage, _ *ResourceMap, _ *synchronization.SyncStructure[string, bool]) core.WorkerNodeInterface {
	schedulable := getSchedulableNodes(storage.GetValues())
	if len(schedulable) == 0 {
		return nil
	}

	var index int32
	swapped := false
	for !swapped {
		index = atomic.LoadInt32(&policy.schedulingCounterRoundRobin) % int32(len(schedulable))
		newIndex := (index + 1) % int32(len(schedulable))

		swapped = atomic.CompareAndSwapInt32(&policy.schedulingCounterRoundRobin, index, newIndex)
	}

	return schedulable[index]
}

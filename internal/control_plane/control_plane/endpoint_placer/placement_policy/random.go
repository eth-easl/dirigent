package placement_policy

import (
	"cluster_manager/internal/control_plane/control_plane/core"
	"cluster_manager/internal/control_plane/control_plane/image_storage"
	"cluster_manager/pkg/synchronization"
	"math/rand"
)

type Random struct{}

func NewRandomPlacement() *Random {
	return &Random{}
}

func (p *Random) Place(storage synchronization.SyncStructure[string, core.WorkerNodeInterface], _ image_storage.ImageStorage, _ *ResourceMap, _ *synchronization.SyncStructure[string, bool]) core.WorkerNodeInterface {
	schedulable := getSchedulableNodes(storage.GetValues())

	if len(schedulable) != 0 {
		return schedulable[rand.Intn(len(schedulable))]
	} else {
		return nil
	}
}

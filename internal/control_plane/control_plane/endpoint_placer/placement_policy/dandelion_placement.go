package placement_policy

import (
	"cluster_manager/internal/control_plane/control_plane/core"
	"cluster_manager/pkg/synchronization"
	"github.com/sirupsen/logrus"
	"math/rand"
)

type DandelionPlacementPolicy struct {
}

func NewDandelionPlacementPolicy() *DandelionPlacementPolicy {
	return &DandelionPlacementPolicy{}
}

func (d *DandelionPlacementPolicy) Place(storage synchronization.SyncStructure[string, core.WorkerNodeInterface], _ *ResourceMap, registeredNodes *synchronization.SyncStructure[string, bool]) core.WorkerNodeInterface {
	schedulables := getSchedulableNodes(storage.GetValues())

	(*registeredNodes).Lock()
	defer (*registeredNodes).Unlock()

	for _, schedulable := range schedulables {
		// find one node in schedulables and not in registeredNode
		if exist, ok := (*registeredNodes).Get(schedulable.GetName()); !ok || !exist {
			logrus.Debugf("dandelion placement chooses node %v", schedulable.GetName())
			(*registeredNodes).Set(schedulable.GetName(), true)

			return schedulable
		}
	}

	logrus.Debugf("dandelion placement fallback to random placement")
	return schedulables[rand.Intn(len(schedulables))]
}

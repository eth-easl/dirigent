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
	schedulableNodes := getSchedulableNodes(storage.GetValues())

	(*registeredNodes).Lock()
	defer (*registeredNodes).Unlock()

	for _, schedulable := range schedulableNodes {
		// find one node in schedulableNodes and not in registeredNode
		if _, ok := (*registeredNodes).Get(schedulable.GetName()); !ok {
			logrus.Tracef("Dandelion placement policy chose node %v", schedulable.GetName())
			(*registeredNodes).Set(schedulable.GetName(), true)

			return schedulable
		}
	}

	logrus.Debugf("Dandelion placement policy fallbacked on random placement")
	return schedulableNodes[rand.Intn(len(schedulableNodes))]
}

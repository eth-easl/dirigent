package placement_policy

import (
	"cluster_manager/internal/control_plane/control_plane/core"
	"cluster_manager/internal/control_plane/control_plane/image_storage"
	"cluster_manager/pkg/synchronization"
)

type PlacementPolicy interface {
	Place(synchronization.SyncStructure[string, core.WorkerNodeInterface], image_storage.ImageStorage, *ResourceMap, *synchronization.SyncStructure[string, bool]) core.WorkerNodeInterface
}

func getSchedulableNodes(allNodes []core.WorkerNodeInterface) []core.WorkerNodeInterface {
	var result []core.WorkerNodeInterface
	for _, node := range allNodes {
		if node.GetSchedulability() {
			result = append(result, node)
		}
	}

	return result
}

func ApplyPlacementPolicy(placementPolicy PlacementPolicy, NIStorage synchronization.SyncStructure[string, core.WorkerNodeInterface], images image_storage.ImageStorage, requested *ResourceMap, dandelionNodes *synchronization.SyncStructure[string, bool]) core.WorkerNodeInterface {
	NIStorage.RLock()
	defer NIStorage.RUnlock()

	if noNodesInCluster(NIStorage) {
		return nil
	}

	return placementPolicy.Place(NIStorage, images, requested, dandelionNodes)
}

func getNumberNodes(storage synchronization.SyncStructure[string, core.WorkerNodeInterface]) int {
	return storage.AtomicLen()
}

func noNodesInCluster(storage synchronization.SyncStructure[string, core.WorkerNodeInterface]) bool {
	return getNumberNodes(storage) == 0
}

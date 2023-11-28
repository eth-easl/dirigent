package placement_policy

import (
	"cluster_manager/internal/control_plane/core"
	_map "cluster_manager/pkg/map"
	"cluster_manager/pkg/synchronization"
	"sort"
)

type PlacementPolicy interface {
	Place(synchronization.SyncStructure[string, core.WorkerNodeInterface], *ResourceMap) core.WorkerNodeInterface
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

func NewKubernetesPolicy() *KubernetesPolicy {
	return &KubernetesPolicy{
		// TODO: Make it dynamic in the future
		resourceMap: CreateResourceMap(1, 1),
	}
}

type KubernetesPolicy struct {
	resourceMap *ResourceMap
}

func ApplyPlacementPolicy(placementPolicy PlacementPolicy, storage synchronization.SyncStructure[string, core.WorkerNodeInterface], requested *ResourceMap) core.WorkerNodeInterface {
	storage.RLock()
	defer storage.RUnlock()

	if noNodesInCluster(storage) {
		return nil
	}

	return placementPolicy.Place(storage, requested)
}

func getNumberNodes(storage synchronization.SyncStructure[string, core.WorkerNodeInterface]) int {
	return storage.Len()
}

func noNodesInCluster(storage synchronization.SyncStructure[string, core.WorkerNodeInterface]) bool {
	return getNumberNodes(storage) == 0
}

func nodeFromIndex(storage synchronization.SyncStructure[string, core.WorkerNodeInterface], index int) core.WorkerNodeInterface {
	nodes := sort.StringSlice(_map.Keys(storage.GetMap()))
	nodes.Sort()
	nodeName := nodes[index]

	value, _ := storage.Get(nodeName)
	return value
}

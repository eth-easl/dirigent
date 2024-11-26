/*
 * MIT License
 *
 * Copyright (c) 2024 EASL
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */

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
	return storage.AtomicLen()
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

package placement_policy

import (
	"cluster_manager/internal/control_plane/core"
	_map "cluster_manager/pkg/map"
	"cluster_manager/pkg/synchronization"
	"math/rand"
	"sort"
	"sync/atomic"

	"github.com/sirupsen/logrus"
)

type PlacementPolicy interface {
	Place(synchronization.SyncStructure[string, core.WorkerNodeInterface], *ResourceMap) core.WorkerNodeInterface
}

func NewRandomPolicy() *RandomPolicy {
	return &RandomPolicy{}
}

type RandomPolicy struct {
}

func NewRoundRobinPolicy() *RoundRobinPolicy {
	return &RoundRobinPolicy{
		schedulingCounterRoundRobin: 0,
	}
}

type RoundRobinPolicy struct {
	schedulingCounterRoundRobin int32
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

func filterMachines(storage synchronization.SyncStructure[string, core.WorkerNodeInterface], resourceMap *ResourceMap) synchronization.SyncStructure[string, core.WorkerNodeInterface] {
	var resultingNodes synchronization.SyncStructure[string, core.WorkerNodeInterface]

	// Model implementation - kubernetes/pkg/scheduler/framework/plugins/noderesources/fit.go:fitsRequest:256
	for key, value := range storage.GetMap() {
		isMemoryBigEnough := value.GetMemory() >= resourceMap.GetMemory()
		isCpuBigEnough := value.GetCpuCores() >= resourceMap.GetCPUCores()

		if !isMemoryBigEnough || !isCpuBigEnough {
			continue
		}

		resultingNodes.Set(key, value)
	}

	return resultingNodes
}

func getInstalledResources(machine core.WorkerNodeInterface) *ResourceMap {
	return CreateResourceMap(machine.GetCpuCores(), machine.GetMemory())
}

func getRequestedResources(machine core.WorkerNodeInterface, request *ResourceMap) *ResourceMap {
	currentUsage := CreateResourceMap(machine.GetCpuUsage()*machine.GetCpuCores(), machine.GetMemoryUsage()*machine.GetMemory())
	return SumResources(currentUsage, request)
}

func prioritizeNodes(storage synchronization.SyncStructure[string, core.WorkerNodeInterface], request *ResourceMap) map[string]uint64 {
	scores := make(map[string]uint64)

	filterAlgorithms := CreateScoringPipeline()

	for _, alg := range filterAlgorithms {
		for key, machine := range storage.GetMap() {
			installedResources := getInstalledResources(machine)
			requestedResources := getRequestedResources(machine, request)

			sc := alg.Score(*installedResources, *requestedResources)
			scores[key] += sc

			logrus.Debugf("%s on node #%s has scored %d.\n", alg.Name, machine.GetName(), sc)
		}
	}

	return scores
}

func selectOneMachine(storage synchronization.SyncStructure[string, core.WorkerNodeInterface], scores map[string]uint64) core.WorkerNodeInterface {
	if storage.Len() == 0 {
		logrus.Fatal("There is no candidate machine to select from.")
	}

	var (
		selected      core.WorkerNodeInterface = nil
		maxScore      uint64                   = 0
		cntOfMaxScore uint64                   = 1
	)

	for key, element := range storage.GetMap() {
		if scores[key] > maxScore {
			maxScore = scores[key]
			selected = element
			cntOfMaxScore = 1
		} else if scores[key] == maxScore {
			cntOfMaxScore++
			if rand.Intn(int(cntOfMaxScore)) == 0 {
				// Replace the candidate with probability of 1/cntOfMaxScore
				selected = element
			}
		}
	}

	return selected
}

func (policy *RandomPolicy) Place(storage synchronization.SyncStructure[string, core.WorkerNodeInterface], requested *ResourceMap) core.WorkerNodeInterface {
	nbNodes := getNumberNodes(storage)

	index := rand.Intn(nbNodes)

	return nodeFromIndex(storage, index)
}

func (policy *RoundRobinPolicy) Place(storage synchronization.SyncStructure[string, core.WorkerNodeInterface], _ *ResourceMap) core.WorkerNodeInterface {
	nbNodes := getNumberNodes(storage)
	if nbNodes == 0 {
		return nil
	}

	var index int32
	swapped := false
	for !swapped {
		index = atomic.LoadInt32(&policy.schedulingCounterRoundRobin) % int32(nbNodes)
		newIndex := (index + 1) % int32(nbNodes)

		swapped = atomic.CompareAndSwapInt32(&policy.schedulingCounterRoundRobin, int32(index), int32(newIndex))
	}

	return nodeFromIndex(storage, int(index))
}

func (policy *KubernetesPolicy) Place(storage synchronization.SyncStructure[string, core.WorkerNodeInterface], requested *ResourceMap) core.WorkerNodeInterface {
	filteredStorage := filterMachines(storage, policy.resourceMap)

	scores := prioritizeNodes(filteredStorage, requested)

	return selectOneMachine(filteredStorage, scores)
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

package placement_policy

import (
	"cluster_manager/internal/control_plane/core"
	"cluster_manager/pkg/atomic_map"
	"math/rand"
	"sort"

	"github.com/sirupsen/logrus"
)

type PlacementPolicy interface {
	Place(*atomic_map.AtomicMap[string, core.WorkerNodeInterface], *ResourceMap) core.WorkerNodeInterface
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
	schedulingCounterRoundRobin int
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

func ApplyPlacementPolicy(placementPolicy PlacementPolicy, storage *atomic_map.AtomicMap[string, core.WorkerNodeInterface], requested *ResourceMap) core.WorkerNodeInterface {
	if noNodesInCluster(storage) {
		return nil
	}

	return placementPolicy.Place(storage, requested)
}

func filterMachines(storage *atomic_map.AtomicMap[string, core.WorkerNodeInterface], resourceMap *ResourceMap) *atomic_map.AtomicMap[string, core.WorkerNodeInterface] {
	var resultingNodes atomic_map.AtomicMap[string, core.WorkerNodeInterface]

	// Model implementation - kubernetes/pkg/scheduler/framework/plugins/noderesources/fit.go:fitsRequest:256
	keys, values := storage.KeyValues()
	for i := 0; i < len(keys); i++ {
		isMemoryBigEnough := values[i].GetMemory() >= resourceMap.GetMemory()
		isCpuBigEnough := values[i].GetCpuCores() >= resourceMap.GetCPUCores()

		if !isMemoryBigEnough || !isCpuBigEnough {
			continue
		}

		resultingNodes.Set(keys[i], values[i])
	}

	return &resultingNodes
}

func getInstalledResources(machine core.WorkerNodeInterface) *ResourceMap {
	return CreateResourceMap(machine.GetCpuCores(), machine.GetMemory())
}

func getRequestedResources(machine core.WorkerNodeInterface, request *ResourceMap) *ResourceMap {
	currentUsage := CreateResourceMap(machine.GetCpuUsage()*machine.GetCpuCores(), machine.GetMemoryUsage()*machine.GetMemory())
	return SumResources(currentUsage, request)
}

func prioritizeNodes(storage *atomic_map.AtomicMap[string, core.WorkerNodeInterface], request *ResourceMap) map[string]int {
	scores := make(map[string]int)

	filterAlgorithms := CreateScoringPipeline()

	for _, alg := range filterAlgorithms {
		keys, machines := storage.KeyValues()
		for i := 0; i < len(keys); i++ {
			installedResources := getInstalledResources(machines[i])
			requestedResources := getRequestedResources(machines[i], request)

			sc := alg.Score(*installedResources, *requestedResources)
			scores[keys[i]] += sc

			logrus.Debugf("%s on node #%s has scored %d.\n", alg.Name, machines[i].GetName(), sc)
		}
	}

	return scores
}

func selectOneMachine(storage *atomic_map.AtomicMap[string, core.WorkerNodeInterface], scores map[string]int) core.WorkerNodeInterface {
	if storage.Len() == 0 {
		logrus.Fatal("There is no candidate machine to select from.")
	}

	var (
		selected      core.WorkerNodeInterface = nil
		maxScore      int                      = -1
		cntOfMaxScore int                      = 1
	)

	keys, elements := storage.KeyValues()
	for i := 0; i < len(keys); i++ {
		if scores[keys[i]] > maxScore {
			maxScore = scores[keys[i]]
			selected = elements[i]
			cntOfMaxScore = 1
		} else if scores[keys[i]] == maxScore {
			cntOfMaxScore++
			if rand.Intn(cntOfMaxScore) == 0 {
				// Replace the candidate with probability of 1/cntOfMaxScore
				selected = elements[i]
			}
		}
	}

	return selected
}

func (policy *RandomPolicy) Place(storage *atomic_map.AtomicMap[string, core.WorkerNodeInterface], requested *ResourceMap) core.WorkerNodeInterface {
	nbNodes := getNumberNodes(storage)

	index := rand.Intn(nbNodes)

	return nodeFromIndex(storage, index)
}

func (policy *RoundRobinPolicy) Place(storage *atomic_map.AtomicMap[string, core.WorkerNodeInterface], _ *ResourceMap) core.WorkerNodeInterface {
	nbNodes := getNumberNodes(storage)
	if nbNodes == 0 {
		return nil
	}

	index := policy.schedulingCounterRoundRobin % nbNodes
	policy.schedulingCounterRoundRobin = (policy.schedulingCounterRoundRobin + 1) % nbNodes

	return nodeFromIndex(storage, index)
}

func (policy *KubernetesPolicy) Place(storage *atomic_map.AtomicMap[string, core.WorkerNodeInterface], requested *ResourceMap) core.WorkerNodeInterface {
	filteredStorage := filterMachines(storage, policy.resourceMap)

	scores := prioritizeNodes(filteredStorage, requested)

	return selectOneMachine(filteredStorage, scores)
}

func getNumberNodes(storage *atomic_map.AtomicMap[string, core.WorkerNodeInterface]) int {
	return storage.Len()
}

func noNodesInCluster(storage *atomic_map.AtomicMap[string, core.WorkerNodeInterface]) bool {
	return getNumberNodes(storage) == 0
}

func nodeFromIndex(storage *atomic_map.AtomicMap[string, core.WorkerNodeInterface], index int) core.WorkerNodeInterface {
	nodes := sort.StringSlice(storage.Keys())
	nodes.Sort()
	nodeName := nodes[index]

	value, _ := storage.Get(nodeName)
	return value
}

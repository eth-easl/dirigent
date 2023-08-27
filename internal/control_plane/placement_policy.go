package control_plane

import (
	"cluster_manager/internal/control_plane/placement_policy"
	_map "cluster_manager/pkg/map"
	"math/rand"
	"sort"

	"github.com/sirupsen/logrus"
)

type PlacementPolicy interface {
	Place(*NodeInfoStorage, *placement_policy.ResourceMap) *WorkerNode
}

func NewRandomPolicy() *randomPolicy {
	return &randomPolicy{}
}

type randomPolicy struct {
}

func NewRoundRobinPolicy() *roundRobinPolicy {
	return &roundRobinPolicy{
		schedulingCounterRoundRobin: 0,
	}
}

type roundRobinPolicy struct {
	schedulingCounterRoundRobin int
}

func NewKubernetesPolicy() *kubernetesPolicy {
	return &kubernetesPolicy{
		// TODO: Make it dynamic in the future
		resourceMap: placement_policy.CreateResourceMap(1, 1),
	}
}

type kubernetesPolicy struct {
	resourceMap *placement_policy.ResourceMap
}

func ApplyPlacementPolicy(placementPolicy PlacementPolicy, storage *NodeInfoStorage, requested *placement_policy.ResourceMap) *WorkerNode {
	storage.Lock()
	defer storage.Unlock()

	if noNodesInCluster(storage) {
		return nil
	}

	return placementPolicy.Place(storage, requested)
}

func filterMachines(storage *NodeInfoStorage, resourceMap *placement_policy.ResourceMap) *NodeInfoStorage {
	var resultingNodes NodeInfoStorage

	storage.Lock()
	defer storage.Unlock()

	// Model implementation - kubernetes/pkg/scheduler/framework/plugins/noderesources/fit.go:fitsRequest:256
	for key, value := range storage.NodeInfo {
		isMemoryBigEnough := value.Memory >= resourceMap.GetMemory()
		isCpuBigEnough := value.CpuCores >= resourceMap.GetCPUCores()

		if !isMemoryBigEnough || !isCpuBigEnough {
			continue
		}

		resultingNodes.NodeInfo[key] = value
	}

	return &resultingNodes
}

func getInstalledResources(machine *WorkerNode) *placement_policy.ResourceMap {
	return placement_policy.CreateResourceMap(machine.CpuCores, machine.Memory)
}

func getRequestedResources(machine *WorkerNode, request *placement_policy.ResourceMap) *placement_policy.ResourceMap {
	currentUsage := placement_policy.CreateResourceMap(machine.CpuUsage*machine.CpuCores, machine.MemoryUsage*machine.Memory)
	return placement_policy.SumResources(currentUsage, request)
}

func prioritizeNodes(storage *NodeInfoStorage, request *placement_policy.ResourceMap) map[string]int {
	scores := make(map[string]int)

	filterAlgorithms := placement_policy.CreateScoringPipeline()

	for _, alg := range filterAlgorithms {
		for key, machine := range storage.NodeInfo {
			installedResources := getInstalledResources(machine)
			requestedResources := getRequestedResources(machine, request)

			sc := alg.Score(*installedResources, *requestedResources)
			scores[key] += sc

			logrus.Debugf("%s on node #%s has scored %d.\n", alg.Name, machine.Name, sc)
		}
	}

	return scores
}

func selectOneMachine(storage *NodeInfoStorage, scores map[string]int) *WorkerNode {
	if len(storage.NodeInfo) == 0 {
		logrus.Fatal("There is no candidate machine to select from.")
	}

	var (
		selected      *WorkerNode = nil
		maxScore      int         = -1
		cntOfMaxScore int         = 1
	)

	for key, element := range storage.NodeInfo {
		if scores[key] > maxScore {
			maxScore = scores[key]
			selected = element
			cntOfMaxScore = 1
		} else if scores[key] == maxScore {
			cntOfMaxScore++
			if rand.Intn(cntOfMaxScore) == 0 {
				// Replace the candidate with probability of 1/cntOfMaxScore
				selected = element
			}
		}
	}

	return selected
}

func (policy *randomPolicy) Place(storage *NodeInfoStorage, requested *placement_policy.ResourceMap) *WorkerNode {
	nbNodes := getNumberNodes(storage)

	index := rand.Intn(nbNodes)

	return nodeFromIndex(storage, index)
}

func (policy *roundRobinPolicy) Place(storage *NodeInfoStorage, _ *placement_policy.ResourceMap) *WorkerNode {
	nbNodes := getNumberNodes(storage)
	if nbNodes == 0 {
		return nil
	}

	index := policy.schedulingCounterRoundRobin % nbNodes
	policy.schedulingCounterRoundRobin = (policy.schedulingCounterRoundRobin + 1) % nbNodes

	return nodeFromIndex(storage, index)
}

func (policy *kubernetesPolicy) Place(storage *NodeInfoStorage, requested *placement_policy.ResourceMap) *WorkerNode {
	filteredStorage := filterMachines(storage, policy.resourceMap)

	scores := prioritizeNodes(filteredStorage, requested)

	return selectOneMachine(filteredStorage, scores)
}

func getNumberNodes(storage *NodeInfoStorage) int {
	return len(_map.Keys(storage.NodeInfo))
}

func noNodesInCluster(storage *NodeInfoStorage) bool {
	return getNumberNodes(storage) == 0
}

func nodeFromIndex(storage *NodeInfoStorage, index int) *WorkerNode {
	nodes := sort.StringSlice(_map.Keys(storage.NodeInfo))
	nodes.Sort()
	nodeName := nodes[index]

	return storage.NodeInfo[nodeName]
}

func EvictionPolicy(endpoint []*Endpoint) (*Endpoint, []*Endpoint) {
	if len(endpoint) == 0 {
		return nil, []*Endpoint{}
	}

	index := rand.Intn(len(endpoint))

	var newEndpointList []*Endpoint

	for i := 0; i < len(endpoint); i++ {
		if i != index {
			newEndpointList = append(newEndpointList, endpoint[i])
		}
	}

	return endpoint[index], newEndpointList
}

package control_plane

import (
	"cluster_manager/internal/control_plane/k8s_placement"
	_map "cluster_manager/pkg/map"
	"github.com/sirupsen/logrus"
	"math/rand"
	"sort"
)

type PlacementPolicy = int

const (
	PLACEMENT_RANDOM PlacementPolicy = iota
	PLACEMENT_ROUND_ROBIN
	PLACEMENT_KUBERNETES
)

func placementPolicy(placementPolicy PlacementPolicy, storage *NodeInfoStorage, requested *k8s_placement.ResourceMap) *WorkerNode {
	storage.Lock()
	defer storage.Unlock()

	switch placementPolicy {
	case PLACEMENT_RANDOM:
		return randomPolicy(storage, requested)
	case PLACEMENT_ROUND_ROBIN:
		return roundRobinPolicy(storage, requested)
	case PLACEMENT_KUBERNETES:
		return kubernetesPolicy(storage, requested)
	default:
		return kubernetesPolicy(storage, requested)
	}
}

func filterMachines(storage *NodeInfoStorage) *NodeInfoStorage {
	return storage // TODO: Implement this function
}

func getInstalledResources(machine *WorkerNode) *k8s_placement.ResourceMap {
	return k8s_placement.CreateResourceMap(machine.CpuCores, machine.Memory)
}

func getRequestedResources(machine *WorkerNode, request *k8s_placement.ResourceMap) *k8s_placement.ResourceMap {
	currentUsage := k8s_placement.CreateResourceMap(machine.CpuUsage*machine.CpuCores, machine.MemoryUsage*machine.Memory)
	return k8s_placement.SumResources(currentUsage, request)
}

func prioritizeNodes(storage *NodeInfoStorage, request *k8s_placement.ResourceMap) map[string]int {
	scores := make(map[string]int)

	filterAlgorithms := k8s_placement.CreateScoringPipeline()

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

func kubernetesPolicy(storage *NodeInfoStorage, requested *k8s_placement.ResourceMap) *WorkerNode {
	filteredStorage := filterMachines(storage)

	scores := prioritizeNodes(filteredStorage, requested)

	return selectOneMachine(filteredStorage, scores)
}

func randomPolicy(storage *NodeInfoStorage, requested *k8s_placement.ResourceMap) *WorkerNode {
	nbNodes := getNumberNodes(storage)

	index := rand.Intn(nbNodes)

	return nodeFromIndex(storage, index)
}

// TODO: Refactor this with side effect handling.
var schedulingCounterRoundRobin int = 0

func roundRobinPolicy(storage *NodeInfoStorage, requested *k8s_placement.ResourceMap) *WorkerNode {
	nbNodes := getNumberNodes(storage)

	index := schedulingCounterRoundRobin % nbNodes
	schedulingCounterRoundRobin = (schedulingCounterRoundRobin + 1) % nbNodes

	return nodeFromIndex(storage, index)
}

func getNumberNodes(storage *NodeInfoStorage) int {
	return len(_map.Keys(storage.NodeInfo))
}

func nodeFromIndex(storage *NodeInfoStorage, index int) *WorkerNode {
	nodes := sort.StringSlice(_map.Keys(storage.NodeInfo))
	nodes.Sort()
	nodeName := nodes[index]

	return storage.NodeInfo[nodeName]
}

func evictionPolicy(endpoint []*Endpoint) (*Endpoint, []*Endpoint) {
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

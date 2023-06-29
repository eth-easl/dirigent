package control_plane

import (
	"cluster_manager/internal/common"
	"cluster_manager/internal/control_plane/placement"
	"math/rand"
	"sort"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

type ScalingMetric string

type PFStateController struct {
	sync.Mutex

	DesiredStateChannel *chan int

	AutoscalingRunning bool
	NotifyChannel      *chan int

	ScalingMetadata AutoscalingMetadata
	Endpoints       []*Endpoint

	Period time.Duration
}

//////////////////////////////////////////////////////
//////////////////////////////////////////////////////
////////////////////////////////////////////////////.//.

func (as *PFStateController) Start() {
	as.Lock()
	defer as.Unlock()

	if !as.AutoscalingRunning {
		as.AutoscalingRunning = true
		go as.ScalingLoop()
	}
}

func (as *PFStateController) ScalingLoop() {
	ticker := time.NewTicker(as.Period)

	isScaleFromZero := true

	logrus.Debug("Starting scaling loop")

	for ; true; <-ticker.C {
		desiredScale := as.ScalingMetadata.KnativeScaling(isScaleFromZero)
		logrus.Debug("Desired scale: ", desiredScale)

		*as.NotifyChannel <- desiredScale

		if isScaleFromZero {
			isScaleFromZero = false
		}

		if desiredScale == 0 {
			as.Lock()
			as.AutoscalingRunning = false
			as.Unlock()

			logrus.Debug("Existed scaling loop")

			break
		}
	}
}

//////////////////////////////////////////////////////
// POLICIES
////////////////////////////////////////////////////.//.

func placementPolicy(placementPolicy placement.PlacementPolicy, storage *NodeInfoStorage, requested *placement.ResourceMap) *WorkerNode {
	storage.Lock()
	defer storage.Unlock()

	switch placementPolicy {
	case placement.PLACMENT_RANDOM:
		return randomPolicy(storage, requested)
	case placement.PLACEMENT_ROUND_ROBIN:
		return roundRobinPolicy(storage, requested)
	case placement.PLACEMENT_KUBERNETES:
		return kubernetesPolicy(storage, requested)
	default:
		return kubernetesPolicy(storage, requested)
	}
}

func filterMachines(storage *NodeInfoStorage) *NodeInfoStorage {
	return storage // TODO: Implement this function
}

func getInstalledResources(machine *WorkerNode) *placement.ResourceMap {
	return placement.CreateResourceMap(machine.CpuCores, machine.Memory)
}

func getRequestedResources(machine *WorkerNode, request *placement.ResourceMap) *placement.ResourceMap {
	currentUsage := placement.CreateResourceMap(machine.CpuUsage*machine.CpuCores, machine.MemoryUsage*machine.Memory)
	return placement.SumResources(currentUsage, request)
}

func prioritizeNodes(storage *NodeInfoStorage, request *placement.ResourceMap) map[string]int {
	scores := make(map[string]int)

	filterAlgorithms := placement.CreateScoringPipeline()

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

func kubernetesPolicy(storage *NodeInfoStorage, requested *placement.ResourceMap) *WorkerNode {
	filteredStorage := filterMachines(storage)

	scores := prioritizeNodes(filteredStorage, requested)

	return selectOneMachine(filteredStorage, scores)
}

func randomPolicy(storage *NodeInfoStorage, requested *placement.ResourceMap) *WorkerNode {
	nbNodes := getNumberNodes(storage)

	index := rand.Intn(nbNodes)

	return nodeFromIndex(storage, index)
}

// TODO: Refactor this with side effect handling.
var schedulingCounterRoundRobin int = 0

func roundRobinPolicy(storage *NodeInfoStorage, requested *placement.ResourceMap) *WorkerNode {
	nbNodes := getNumberNodes(storage)

	index := schedulingCounterRoundRobin % nbNodes
	schedulingCounterRoundRobin = (schedulingCounterRoundRobin + 1) % nbNodes

	return nodeFromIndex(storage, index)
}

func getNumberNodes(storage *NodeInfoStorage) int {
	return len(common.Keys(storage.NodeInfo))
}

func nodeFromIndex(storage *NodeInfoStorage, index int) *WorkerNode {
	nodes := sort.StringSlice(common.Keys(storage.NodeInfo))
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

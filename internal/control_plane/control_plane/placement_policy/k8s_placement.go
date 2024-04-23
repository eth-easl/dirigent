package placement_policy

import (
	"cluster_manager/internal/control_plane/control_plane/core"
	"cluster_manager/pkg/synchronization"
	"github.com/sirupsen/logrus"
	"math"
	"math/rand"
)

type ScoringAlgorithmType func(ResourceMap, ResourceMap) uint64

type ScoringAlgorithm struct {
	Name  string
	Score ScoringAlgorithmType
}

func CreateScoringPipeline() []ScoringAlgorithm {
	return []ScoringAlgorithm{
		{
			Name:  "NodeResourcesFit",
			Score: ScoreFitLeastAllocated,
		},
		{
			Name:  "NodeResourcesBalancedAllocation",
			Score: ScoreBalancedAllocation,
		},
	}
}

func leastRequestedScore(requested, capacity uint64) uint64 {
	if capacity == 0 {
		return 0
	}

	if requested > capacity {
		return 0
	}

	// The more unused resources the higher the score is
	return ((capacity - requested) * 100) / capacity
}

func ScoreFitLeastAllocated(installed ResourceMap, requested ResourceMap) uint64 {
	var nodeScore uint64 = 0
	var weightSum uint64 = 0

	weights := map[string]uint64{
		RM_CPU_KEY:    1,
		RM_MEMORY_KEY: 1,
	}

	for _, key := range installed.ResourceKeys() {
		req, av := requested.GetByKey(key), installed.GetByKey(key)

		resourceScore := leastRequestedScore(req, av)
		nodeScore += resourceScore * weights[key]
		weightSum += weights[key]
	}

	if weightSum == 0 {
		return 0
	}

	return nodeScore / weightSum
}

func ScoreBalancedAllocation(installed ResourceMap, requested ResourceMap) uint64 {
	var (
		totalFraction float64
	)

	resourceToFractions := make([]float64, 0)

	for _, key := range installed.ResourceKeys() {
		req, av := requested.GetByKey(key), installed.GetByKey(key)

		fraction := float64(req) / float64(av)
		if fraction > 1 {
			fraction = 1
		}

		totalFraction += fraction
		resourceToFractions = append(resourceToFractions, fraction)
	}

	std := 0.0

	// For most cases, resources are limited to cpu and memory, the std could be simplified to std := (fraction1-fraction2)/2
	// len(fractions) > 2: calculate std based on the well-known formula - root square of Σ((fraction(i)-mean)^2)/len(fractions)
	// Otherwise, set the std to zero is enough.
	if len(resourceToFractions) == 2 {
		std = math.Abs((resourceToFractions[0] - resourceToFractions[1]) / 2)
	} else if len(resourceToFractions) > 2 {
		mean := totalFraction / float64(len(resourceToFractions))
		var sum float64
		for _, fraction := range resourceToFractions {
			sum = sum + (fraction-mean)*(fraction-mean)
		}
		std = math.Sqrt(sum / float64(len(resourceToFractions)))
	}

	// STD (standard deviation) is always a positive value. 1-deviation lets the score to be higher for node which has least deviation and
	// multiplying it with `MaxNodeScore` provides the scaling factor needed.
	return uint64((1 - std) * float64(100))
}

func filterMachines(storage synchronization.SyncStructure[string, core.WorkerNodeInterface], resourceMap *ResourceMap) synchronization.SyncStructure[string, core.WorkerNodeInterface] {
	var resultingNodes synchronization.SyncStructure[string, core.WorkerNodeInterface]

	// Model implementation - kubernetes/pkg/scheduler/framework/plugins/noderesources/fit.go:fitsRequest:256
	for key, value := range storage.GetMap() {
		isMemoryBigEnough := value.GetMemory() >= resourceMap.GetMemory()
		isCpuBigEnough := value.GetCpuCores() >= resourceMap.GetCPUCores()

		if !isMemoryBigEnough || !isCpuBigEnough || !value.GetSchedulability() {
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
	if storage.AtomicLen() == 0 {
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

func (policy *KubernetesPolicy) Place(storage synchronization.SyncStructure[string, core.WorkerNodeInterface], requested *ResourceMap) core.WorkerNodeInterface {
	filteredStorage := filterMachines(storage, policy.resourceMap)
	scores := prioritizeNodes(filteredStorage, requested)

	return selectOneMachine(filteredStorage, scores)
}

package placement_policy

import (
	"cluster_manager/internal/control_plane/control_plane/core"
	"cluster_manager/internal/control_plane/control_plane/image_storage"
	"cluster_manager/pkg/synchronization"
	"math"
	"math/rand"

	"github.com/sirupsen/logrus"
)

const (
	oneMebibyte  = 1024 * 1024
	minImageSize = uint64(50 * oneMebibyte)
	maxImageSize = uint64(1000 * oneMebibyte)
)

func NewKubernetesPolicy() *KubernetesPolicy {
	return &KubernetesPolicy{
		// TODO: Make it dynamic in the future
		resourceMap: CreateResourceMap(1, 1, ""),
	}
}

type KubernetesPolicy struct {
	resourceMap *ResourceMap
}

func (policy *KubernetesPolicy) Place(nodes synchronization.SyncStructure[string, core.WorkerNodeInterface], images image_storage.ImageStorage, requested *ResourceMap, _ *synchronization.SyncStructure[string, bool]) core.WorkerNodeInterface {
	filteredNodes := filterMachines(nodes, policy.resourceMap)
	scores := prioritizeNodes(filteredNodes, nodes, images, requested)

	return selectOneMachine(nodes, scores)
}

type ScoringAlgorithmType func(ResourceMap, ResourceMap, image_storage.ImageStorage, synchronization.SyncStructure[string, core.WorkerNodeInterface], core.WorkerNodeInterface) uint64

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
		{
			Name:  "ImageLocality",
			Score: ScoreImageLocality,
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

func ScoreFitLeastAllocated(installed ResourceMap, requested ResourceMap, _ image_storage.ImageStorage, _ synchronization.SyncStructure[string, core.WorkerNodeInterface], _ core.WorkerNodeInterface) uint64 {
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

func ScoreBalancedAllocation(installed ResourceMap, requested ResourceMap, _ image_storage.ImageStorage, _ synchronization.SyncStructure[string, core.WorkerNodeInterface], _ core.WorkerNodeInterface) uint64 {
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

func ScoreImageLocality(installed ResourceMap, requested ResourceMap, images image_storage.ImageStorage, nodes synchronization.SyncStructure[string, core.WorkerNodeInterface], node core.WorkerNodeInterface) uint64 {
	image := requested.GetImage()
	imageInfo, ok := images.Get(image)
	if !ok {
		logrus.Warnf("K8s placement policy cannot determine image locality for unregistered image %s", image)
		return 0
	}
	if !node.HasImage(requested.GetImage()) {
		return 0
	}
	spread := float64(imageInfo.Count) / float64(nodes.AtomicLen())
	scaledImageSize := uint64(float64(imageInfo.Size) * spread)
	if scaledImageSize > maxImageSize {
		scaledImageSize = maxImageSize
	} else if scaledImageSize < minImageSize {
		scaledImageSize = minImageSize
	}
	return 100 * (scaledImageSize - minImageSize) / (maxImageSize - minImageSize)
}

func minimumNumberOfNodesToFilter(storage synchronization.SyncStructure[string, core.WorkerNodeInterface]) int {
	count := storage.Len()
	minimum := 100

	if count < minimum {
		return count
	}

	pct := math.Max(5, float64(50-count/125)) // count is integer in K8s original implementation
	res := int(math.Max(100, float64(count)*pct/100))

	logrus.Tracef("Filtered machines to %d/%d", res, count)
	return res
}

func filterMachines(storage synchronization.SyncStructure[string, core.WorkerNodeInterface], resourceMap *ResourceMap) []string {
	var resultingNodes []string
	count := 0
	minCount := minimumNumberOfNodesToFilter(storage)

	// Model implementation - kubernetes/pkg/scheduler/framework/plugins/noderesources/fit.go:fitsRequest:256
	for key, value := range storage.GetMap() {
		isMemoryBigEnough := value.GetMemoryAvailable() >= resourceMap.GetMemory()
		isCpuBigEnough := value.GetCpuAvailable() >= resourceMap.GetCpu()

		if !isMemoryBigEnough || !isCpuBigEnough || !value.GetSchedulability() {
			continue
		}

		resultingNodes = append(resultingNodes, key)
		count++

		if count >= minCount {
			break
		}
	}

	return resultingNodes
}

func getInstalledResources(machine core.WorkerNodeInterface) *ResourceMap {
	return CreateResourceMap(machine.GetCpuAvailable(), machine.GetMemoryAvailable(), "")
}

func getRequestedResources(machine core.WorkerNodeInterface, request *ResourceMap) *ResourceMap {
	currentUsage := CreateResourceMap(machine.GetCpuUsed(), machine.GetCpuAvailable(), request.GetImage())
	return SumResources(currentUsage, request)
}

func prioritizeNodes(nodes []string, storage synchronization.SyncStructure[string, core.WorkerNodeInterface], images image_storage.ImageStorage, request *ResourceMap) map[string]uint64 {
	scores := make(map[string]uint64)
	if len(nodes) == 0 {
		return scores
	}

	filterAlgorithms := CreateScoringPipeline()
	for _, machine := range nodes {
		wni := storage.GetNoCheck(machine)

		installedResources := getInstalledResources(wni)
		requestedResources := getRequestedResources(wni, request)
		score := uint64(0)
		for _, alg := range filterAlgorithms {
			sc := alg.Score(*installedResources, *requestedResources, images, storage, wni)
			score += sc

			logrus.Tracef("%s on node #%s has scored %d.\n", alg.Name, machine, sc)
		}
		scores[machine] = score
	}

	return scores
}

func selectOneMachine(storage synchronization.SyncStructure[string, core.WorkerNodeInterface], scores map[string]uint64) core.WorkerNodeInterface {
	var (
		selected      core.WorkerNodeInterface = nil
		maxScore      uint64                   = 0
		cntOfMaxScore uint64                   = 1
	)

	if storage.AtomicLen() == 0 || len(scores) == 0 {
		return selected
	}

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
	logrus.Tracef("Kubernetes policy selected node %s out of %d", selected.GetName(), cntOfMaxScore)

	return selected
}

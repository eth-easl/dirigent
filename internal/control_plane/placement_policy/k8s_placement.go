package placement_policy

import (
	"math"
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

package placement

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLeastAllocated(t *testing.T) {
	// values taken from k8s.io/kubernetes/pkg/scheduler/framework/plugins/noderesources:690
	request := ExtendCPU(CreateResourceMap(1000, 2000))
	n1_occupied := ExtendCPU(CreateResourceMap(2000, 4000))
	n2_occupied := ExtendCPU(CreateResourceMap(1000, 2000))
	n1_capacity := ExtendCPU(CreateResourceMap(4000, 10000))
	n2_capacity := ExtendCPU(CreateResourceMap(6000, 10000))

	n1_score := ScoreFitLeastAllocated(*n1_capacity, *SumResources(request, n1_occupied))
	n2_score := ScoreFitLeastAllocated(*n2_capacity, *SumResources(request, n2_occupied))

	assert.True(t, n1_score == 32 && n2_score == 63, "Least allocated scoring test failed.")
}

func TestBalancedAllocation(t *testing.T) {
	request := ExtendCPU(CreateResourceMap(3000, 5000))
	n1_occupied := ExtendCPU(CreateResourceMap(3000, 0))
	n2_occupied := ExtendCPU(CreateResourceMap(3000, 5000))
	n1_capacity := ExtendCPU(CreateResourceMap(10000, 20000))
	n2_capacity := ExtendCPU(CreateResourceMap(10000, 50000))

	n1_score := ScoreBalancedAllocation(*n1_capacity, *SumResources(request, n1_occupied))
	n2_score := ScoreBalancedAllocation(*n2_capacity, *SumResources(request, n2_occupied))

	assert.True(t, n1_score == 82 && n2_score == 80, "Least allocated scoring test failed.")
}

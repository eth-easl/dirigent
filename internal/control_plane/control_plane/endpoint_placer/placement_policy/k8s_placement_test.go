package placement_policy

import (
	"cluster_manager/internal/control_plane/control_plane/image_storage"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLeastAllocated(t *testing.T) {
	images := image_storage.NewDefaultImageStorage()
	// values taken from k8s.io/kubernetes/pkg/scheduler/framework/plugins/noderesources:690
	request := ExtendCPU(CreateResourceMap(1000, 2000, 0, "image"))
	n1_occupied := ExtendCPU(CreateResourceMap(2000, 4000, 0, "image"))
	n2_occupied := ExtendCPU(CreateResourceMap(1000, 2000, 0, "image"))
	n1_capacity := ExtendCPU(CreateResourceMap(4000, 10000, 0, "image"))
	n2_capacity := ExtendCPU(CreateResourceMap(6000, 10000, 0, "image"))

	n1_score := ScoreFitLeastAllocated(*n1_capacity, *SumResources(request, n1_occupied), images, nil, nil)
	n2_score := ScoreFitLeastAllocated(*n2_capacity, *SumResources(request, n2_occupied), images, nil, nil)

	assert.True(t, n1_score == 32 && n2_score == 63, "Least allocated scoring test failed.")
}

func TestBalancedAllocation(t *testing.T) {
	images := image_storage.NewDefaultImageStorage()
	request := ExtendCPU(CreateResourceMap(3000, 5000, 0, "image"))
	n1_occupied := ExtendCPU(CreateResourceMap(3000, 0, 0, "image"))
	n2_occupied := ExtendCPU(CreateResourceMap(3000, 5000, 0, "image"))
	n1_capacity := ExtendCPU(CreateResourceMap(10000, 20000, 0, "image"))
	n2_capacity := ExtendCPU(CreateResourceMap(10000, 50000, 0, "image"))

	n1_score := ScoreBalancedAllocation(*n1_capacity, *SumResources(request, n1_occupied), images, nil, nil)
	n2_score := ScoreBalancedAllocation(*n2_capacity, *SumResources(request, n2_occupied), images, nil, nil)

	assert.True(t, n1_score == 82 && n2_score == 80, "Least allocated scoring test failed.")
}

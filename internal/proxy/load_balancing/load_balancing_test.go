package load_balancing

import (
	"cluster_manager/internal/common"
	"testing"

	"github.com/stretchr/testify/assert"
)

func getTestEndpoints() ([]*common.UpstreamEndpoint, int) {
	return []*common.UpstreamEndpoint{
		{
			ID: "1",
		},
		{
			ID: "2",
		},
		{
			ID: "3",
		},
		{
			ID: "4",
		},
		{
			ID: "5",
		},
		{
			ID: "6",
		},
		{
			ID: "7",
		},
	}, 7 // Size of the array
}

func TestRandomLoadBalancing(t *testing.T) {
	endpoints, _ := getTestEndpoints()

	endpointsMap := make(map[*common.UpstreamEndpoint]interface{})
	for _, elem := range endpoints {
		endpointsMap[elem] = 0
	}

	for i := 0; i < 10; i++ {
		endpoint := randomLoadBalancing(endpoints)
		if _, ok := endpointsMap[endpoint]; ok {
			assert.True(t, ok, "Element isn't present in the endpoints")
		}
	}
}

func TestRoundRobinLoadBalancing(t *testing.T) {
	endpoints, sizeEndoints := getTestEndpoints()

	for i := 0; i < 100; i++ {
		for j := 0; j < sizeEndoints; j++ {
			endpoint := roundRobinLoadBalancing(endpoints)
			assert.Equal(t, endpoints[j], endpoint, "Endpoint isn't the correct one")
		}
	}
}

func TestLeastProcessedLoadBalancing(t *testing.T) {
	endpoints, endpointsSize := getTestEndpoints()
	metadata := common.NewFunctionMetadata("mockMetaData")

	metadata.GetRequestCountPerInstance()[endpoints[0]]++
	metadata.GetRequestCountPerInstance()[endpoints[1]]++

	// TODO: Refactor the last statement, François Costa
	for i := 2; i < endpointsSize; i++ {
		endpoint := leastProcessedLoadBalancing(metadata, endpoints)
		assert.Equal(t, endpoints[i], endpoint, "Endpoint isn't the correct one")
		metadata.GetRequestCountPerInstance()[endpoint]++
	}

	for i := 0; i < endpointsSize; i++ {
		endpoint := leastProcessedLoadBalancing(metadata, endpoints)
		assert.Equal(t, endpoints[i], endpoint, "Endpoint isn't the correct one")
		metadata.GetRequestCountPerInstance()[endpoint]++
	}
}

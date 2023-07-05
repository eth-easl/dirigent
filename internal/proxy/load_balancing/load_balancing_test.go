package load_balancing

import (
	"cluster_manager/internal/common"
	"testing"

	"github.com/stretchr/testify/assert"
)

func getTestEndpoints() (*common.FunctionMetadata, int) {
	endpoints := []*common.UpstreamEndpoint{
		{
			ID:       "1",
			Capacity: make(chan struct{}, 10),
		},
		{
			ID:       "2",
			Capacity: make(chan struct{}, 10),
		},
		{
			ID:       "3",
			Capacity: make(chan struct{}, 10),
		},
		{
			ID:       "4",
			Capacity: make(chan struct{}, 10),
		},
		{
			ID:       "5",
			Capacity: make(chan struct{}, 10),
		},
		{
			ID:       "6",
			Capacity: make(chan struct{}, 10),
		},
		{
			ID:       "7",
			Capacity: make(chan struct{}, 10),
		},
	}

	metadata := common.NewFunctionMetadata("mockName")
	metadata.SetEndpoints(endpoints)

	return metadata, len(endpoints)
}

func TestRandomLoadBalancing(t *testing.T) {
	metadata, _ := getTestEndpoints()

	endpointsMap := make(map[*common.UpstreamEndpoint]interface{})
	for _, elem := range metadata.GetUpstreamEndpoints() {
		endpointsMap[elem] = 0
	}

	for i := 0; i < 10; i++ {
		endpoint := randomLoadBalancing(metadata)
		if _, ok := endpointsMap[endpoint]; ok {
			assert.True(t, ok, "Element isn't present in the endpoints")
		}
	}
}

func TestRoundRobinLoadBalancing(t *testing.T) {
	metadata, sizeEndoints := getTestEndpoints()
	endpoints := metadata.GetUpstreamEndpoints()

	for i := 0; i < 100; i++ {
		for j := 0; j < sizeEndoints; j++ {
			endpoint := roundRobinLoadBalancing(metadata)
			assert.Equal(t, endpoints[j], endpoint, "Endpoint isn't the correct one")
		}
	}
}

func TestLeastProcessedLoadBalancing(t *testing.T) {
	metadata, endpointsSize := getTestEndpoints()
	endpoints := metadata.GetUpstreamEndpoints()

	metadata.GetRequestCountPerInstance()[endpoints[0]]++
	metadata.GetRequestCountPerInstance()[endpoints[1]]++

	for i := 2; i < endpointsSize; i++ {
		endpoint := leastProcessedLoadBalancing(metadata)
		assert.Equal(t, endpoints[i], endpoint, "Endpoint isn't the correct one")
		metadata.GetRequestCountPerInstance()[endpoint]++
	}

	for i := 0; i < endpointsSize; i++ {
		endpoint := leastProcessedLoadBalancing(metadata)
		assert.Equal(t, endpoints[i], endpoint, "Endpoint isn't the correct one")
		metadata.GetRequestCountPerInstance()[endpoint]++
	}
}

func TestGenerateTwoUniformRandomEndpoints(t *testing.T) {
	metadata, _ := getTestEndpoints()
	for i := 0; i < 10000; i++ {
		endpoint1, endpoint2 := generateTwoUniformRandomEndpoints(metadata.GetUpstreamEndpoints())
		assert.NotSamef(t, endpoint1, endpoint2, "Endpoints should not be the same")
	}
}

func TestBestOfTwoRandoms(t *testing.T) {
	metadata, _ := getTestEndpoints()

	endpointsMap := make(map[*common.UpstreamEndpoint]interface{})
	for _, elem := range metadata.GetUpstreamEndpoints() {
		endpointsMap[elem] = 0
	}

	for i := 0; i < 10; i++ {
		endpoint := bestOfTwoRandoms(metadata)
		if _, ok := endpointsMap[endpoint]; ok {
			assert.True(t, ok, "Element isn't present in the endpoints")
		}
	}
}

func TestKubernetesRoundRobinLoadBalancing(t *testing.T) {
	metadata, size := getTestEndpoints()
	endpoints := metadata.GetUpstreamEndpoints()

	for j := 0; j < 200; j++ {
		for i := 0; i < size; i++ {
			endpoints[i].Capacity <- struct{}{}

			endpoint := kubernetesRoundRobinLoadBalancing(metadata)
			assert.Equal(t, endpoints[i], endpoint, "Endpoints aren't the same")

			<-endpoint.Capacity
		}
	}

	/*for i := 0; i < 1; i++ {
		index := rand.Intn(len(endpoints))

		endpoints[index].Capacity <- struct{}{}

		endpoint := kubernetesRoundRobinLoadBalancing(metadata)
		assert.Equal(t, endpoints[index], endpoint, "Endpoints aren't the same")

		<-endpoint.Capacity
	}*/
}

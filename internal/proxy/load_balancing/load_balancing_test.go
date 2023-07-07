package load_balancing

import (
	"cluster_manager/internal/common"
	"math/rand"
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
	metadata, sizeEndpoints := getTestEndpoints()
	endpoints := metadata.GetUpstreamEndpoints()

	for i := 0; i < 100; i++ {
		for j := 0; j < sizeEndpoints; j++ {
			endpoint := roundRobinLoadBalancing(metadata)
			assert.Equal(t, endpoints[j], endpoint, "Endpoint isn't the correct one")
		}
	}
}

func TestLeastProcessedLoadBalancing(t *testing.T) {
	metadata, endpointsSize := getTestEndpoints()
	endpoints := metadata.GetUpstreamEndpoints()

	countPerInstance := metadata.GetRequestCountPerInstance()
	countPerInstance.AtomicIncrement(endpoints[0])
	countPerInstance.AtomicIncrement(endpoints[1])

	for i := 2; i < endpointsSize; i++ {
		endpoint := leastProcessedLoadBalancing(metadata)
		assert.Equal(t, endpoints[i], endpoint, "Endpoint isn't the correct one")
		countPerInstance.AtomicIncrement(endpoints[i])
	}

	for i := 0; i < endpointsSize; i++ {
		endpoint := leastProcessedLoadBalancing(metadata)
		assert.Equal(t, endpoints[i], endpoint, "Endpoint isn't the correct one")
		countPerInstance.AtomicIncrement(endpoints[i])
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

func TestKubernetesRoundRobinLoadBalancingSimple(t *testing.T) {
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
}

func TestKubernetesRoundRobinLoadBalancingRandomFree(t *testing.T) {
	metadata, size := getTestEndpoints()
	endpoints := metadata.GetUpstreamEndpoints()

	for i := 0; i < 200; i++ {
		index := rand.Intn(size)

		endpoints[index].Capacity <- struct{}{}

		endpoint := kubernetesRoundRobinLoadBalancing(metadata)
		assert.Equal(t, endpoints[index], endpoint, "Endpoints aren't the same")

		<-endpoint.Capacity
	}
}

func TestKubernetesFirstAvailableLoadBalancingSimple(t *testing.T) {
	metadata, size := getTestEndpoints()
	endpoints := metadata.GetUpstreamEndpoints()

	for j := 0; j < 200; j++ {
		for i := 0; i < size; i++ {
			endpoints[i].Capacity <- struct{}{}

			endpoint := kubernetesFirstAvailableLoadBalancing(metadata)
			assert.Equal(t, endpoints[i], endpoint, "Endpoints aren't the same")

			<-endpoint.Capacity
		}
	}
}

func TestKubernetesFirstAvailableLoadBalancingRandom(t *testing.T) {
	metadata, size := getTestEndpoints()
	endpoints := metadata.GetUpstreamEndpoints()

	endpoints[size-1].Capacity <- struct{}{}

	for i := 0; i < 200; i++ {
		index := rand.Intn(size - 1)

		endpoints[index].Capacity <- struct{}{}

		endpoint := kubernetesFirstAvailableLoadBalancing(metadata)
		assert.Equal(t, endpoints[index], endpoint, "Endpoints aren't the same")

		<-endpoint.Capacity
	}
}

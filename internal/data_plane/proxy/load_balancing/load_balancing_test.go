package load_balancing

import (
	"cluster_manager/internal/data_plane/service_metadata"
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/montanaflynn/stats"
	"github.com/sirupsen/logrus"

	"github.com/stretchr/testify/assert"
)

func getTestEndpoints(numberOfEndpoints int, containerConcurrency uint) (*service_metadata.ServiceMetadata, int) {
	var endpoints []*service_metadata.UpstreamEndpoint
	for i := 0; i < numberOfEndpoints; i++ {
		endpoints = append(endpoints, &service_metadata.UpstreamEndpoint{
			ID:       fmt.Sprintf("%d", i+1),
			Capacity: make(chan struct{}, 10),
		})
	}

	metadata := service_metadata.NewFunctionMetadata("mockName", "test", containerConcurrency)
	metadata.SetEndpoints(endpoints)

	return metadata, len(endpoints)
}

func getTestEndpointWithCapacityAdditions(numberOfEndpoints int, containerConcurrency int) (*service_metadata.ServiceMetadata, int) {
	metadata, size := getTestEndpoints(numberOfEndpoints, uint(containerConcurrency))
	for i := 0; i < size; i++ {
		for t := 0; t < containerConcurrency; t++ {
			metadata.GetUpstreamEndpoints()[i].Capacity <- struct{}{}
		}
	}

	return metadata, size
}

func TestRandomLoadBalancing(t *testing.T) {
	metadata, _ := getTestEndpoints(7, 1)

	endpointsMap := make(map[*service_metadata.UpstreamEndpoint]interface{})
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
	metadata, sizeEndpoints := getTestEndpoints(7, 1)
	endpoints := metadata.GetUpstreamEndpoints()

	for i := 0; i < 100; i++ {
		for j := 0; j < sizeEndpoints; j++ {
			endpoint := roundRobinLoadBalancing(metadata)
			assert.Equal(t, endpoints[j], endpoint, "Endpoint isn't the correct one")
		}
	}
}

func TestLeastProcessedLoadBalancing(t *testing.T) {
	metadata, endpointsSize := getTestEndpoints(7, 1)
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
	metadata, _ := getTestEndpoints(7, 1)
	for i := 0; i < 10000; i++ {
		endpoint1, endpoint2 := generateTwoUniformRandomEndpoints(metadata.GetUpstreamEndpoints())
		assert.NotSamef(t, endpoint1, endpoint2, "Endpoints should not be the same")
	}
}

func TestBestOfTwoRandoms(t *testing.T) {
	metadata, _ := getTestEndpoints(7, 1)

	endpointsMap := make(map[*service_metadata.UpstreamEndpoint]interface{})
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
	metadata, size := getTestEndpoints(7, 1)
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
	metadata, size := getTestEndpoints(7, 1)
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
	metadata, size := getTestEndpoints(7, 1)
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
	metadata, size := getTestEndpoints(7, 1)
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

func TestLoadBalancingOnXKEndpoints(t *testing.T) {
	tests := []struct {
		Name        string
		Endpoints   int
		Iterations  int
		Parallelism int
	}{
		{
			Name:        "endpoints_1000",
			Endpoints:   1000,
			Iterations:  1,
			Parallelism: 1000,
		},
		{
			Name:        "endpoints_5000",
			Endpoints:   5000,
			Iterations:  1,
			Parallelism: 5000,
		},
	}

	logrus.SetFormatter(&logrus.TextFormatter{TimestampFormat: time.StampMilli, FullTimestamp: true})

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			var data []float64
			wg := sync.WaitGroup{}
			mutex := sync.Mutex{}

			for i := 0; i < test.Iterations; i++ {
				metadata, _ := getTestEndpointWithCapacityAdditions(test.Endpoints, 1)

				wg.Add(test.Parallelism)
				for p := 0; p < test.Parallelism; p++ {
					go func() {
						defer wg.Done()

						start := time.Now()
						var m *service_metadata.UpstreamEndpoint
						for m == nil {
							m = runLoadBalancingAlgorithm(metadata, LOAD_BALANCING_KNATIVE)
						}
						dt := time.Since(start).Milliseconds()

						go func() {
							time.Sleep(time.Duration(rand.Intn(1000)) * time.Millisecond)
							m.Capacity <- struct{}{}
						}()
						<-m.Capacity

						mutex.Lock()
						data = append(data, float64(dt))
						mutex.Unlock()

						logrus.Tracef("Load balancing across %d endpoint set took %d ms", test.Endpoints, dt)
					}()
				}

				wg.Wait()
			}

			p50, _ := stats.Percentile(data, 50)
			p99, _ := stats.Percentile(data, 99)
			logrus.Infof("p50: %.2f ms; p99: %.2f ms", p50, p99)
		})
	}
}

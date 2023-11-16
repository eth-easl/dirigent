package load_balancing

import (
	"cluster_manager/internal/data_plane/function_metadata"
	"math/rand"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
)

type LoadBalancingPolicy = int

const (
	LOAD_BALANCING_RANDOM LoadBalancingPolicy = iota
	LOAD_BALANCING_ROUND_ROBIN
	LOAD_BALANCING_LEAST_PROCESSED
	LOAD_BALANCING_KNATIVE
)

func runLoadBalancingAlgorithm(metadata *function_metadata.FunctionMetadata, loadBalancingPolicy LoadBalancingPolicy) *function_metadata.UpstreamEndpoint {
	metadata.RLock()
	defer metadata.RUnlock()

	endpoints := metadata.GetUpstreamEndpoints()
	if endpoints == nil || len(endpoints) == 0 {
		return nil
	}

	var endpoint *function_metadata.UpstreamEndpoint
	switch loadBalancingPolicy {
	case LOAD_BALANCING_RANDOM:
		endpoint = randomLoadBalancing(metadata)
	case LOAD_BALANCING_ROUND_ROBIN:
		endpoint = roundRobinLoadBalancing(metadata)
	case LOAD_BALANCING_LEAST_PROCESSED:
		endpoint = leastProcessedLoadBalancing(metadata)
	case LOAD_BALANCING_KNATIVE:
		endpoint = kubernetesNativeLoadBalancing(metadata)
	default:
		endpoint = randomLoadBalancing(metadata)
	}

	return endpoint
}

func DoLoadBalancing(req *http.Request, metadata *function_metadata.FunctionMetadata, loadBalancingPolicy LoadBalancingPolicy) (*function_metadata.UpstreamEndpoint, time.Duration, time.Duration) {
	lbDuration, ccDuration := time.Duration(0), time.Duration(0)

	var endpoint *function_metadata.UpstreamEndpoint
	loadBalancingRetries := 3

	// trying to get an endpoint and capacity for a couple of times
	for i := 0; i < loadBalancingRetries; i++ {
		lbStart := time.Now()
		endpoint = runLoadBalancingAlgorithm(metadata, loadBalancingPolicy)
		lbDuration += time.Since(lbStart)

		// CC throttler
		gotCapacity, ccLength := waitForCapacityOrDie(endpoint.Capacity)
		ccDuration += ccLength

		if gotCapacity {
			break
		} else {
			endpoint = nil
		}
	}

	if endpoint == nil {
		return nil, lbDuration, ccDuration
	}

	atomic.AddInt32(&endpoint.InFlight, 1)

	req.URL.Scheme = "http"
	req.URL.Host = endpoint.URL
	logrus.Debugf("Invocation for %s forwarded to %s.", metadata.GetIdentifier(), endpoint.URL)

	metadata.IncrementRequestCountPerInstance(endpoint)

	return endpoint, lbDuration, ccDuration
}

func waitForCapacityOrDie(throttler function_metadata.RequestThrottler) (bool, time.Duration) {
	start := time.Now()

	select {
	case <-throttler:
		return true, time.Since(start)
	case <-time.After(30 * time.Second):
		return false, time.Since(start)
	}
}

func randomEndpoint(endpoints []*function_metadata.UpstreamEndpoint) *function_metadata.UpstreamEndpoint {
	index := rand.Intn(len(endpoints))
	return endpoints[index]
}

func randomLoadBalancing(metadata *function_metadata.FunctionMetadata) *function_metadata.UpstreamEndpoint {
	return randomEndpoint(metadata.GetUpstreamEndpoints())
}

func roundRobinLoadBalancing(metadata *function_metadata.FunctionMetadata) *function_metadata.UpstreamEndpoint {
	endpoints := metadata.GetUpstreamEndpoints()
	outputEndpoint := endpoints[metadata.GetRoundRobinCounter()]
	metadata.IncrementRoundRobinCounter()

	return outputEndpoint
}

func leastProcessedLoadBalancing(metadata *function_metadata.FunctionMetadata) *function_metadata.UpstreamEndpoint {
	endpoints := metadata.GetUpstreamEndpoints()
	requestCountPerInstance := metadata.GetRequestCountPerInstance()

	minEndpoint := endpoints[0]
	minValue := requestCountPerInstance.Get(minEndpoint)

	for _, endpoint := range endpoints {
		countPerInstance := requestCountPerInstance.Get(endpoint)

		if countPerInstance < minValue {
			minEndpoint = endpoint
			minValue = countPerInstance
		}
	}

	return minEndpoint
}

func generateTwoUniformRandomEndpoints(endpoints []*function_metadata.UpstreamEndpoint) (*function_metadata.UpstreamEndpoint, *function_metadata.UpstreamEndpoint) {
	/*
		The trick is that we generate from 0 to l-1 and if the generated value is bigger than the first.
		We increase by one. We get two uniform randoms distributions and the number are different.
	*/
	size := len(endpoints)
	r1, r2 := rand.Intn(size), rand.Intn(size-1)

	if r2 >= r1 {
		r2++
	}

	return endpoints[r1], endpoints[r2]
}

func bestOfTwoRandoms(metadata *function_metadata.FunctionMetadata) *function_metadata.UpstreamEndpoint {
	endpoints := metadata.GetUpstreamEndpoints()

	var output *function_metadata.UpstreamEndpoint

	if len(endpoints) == 1 {
		output = endpoints[0]
	} else {
		endpoint1, endpoint2 := generateTwoUniformRandomEndpoints(endpoints)
		if metadata.GetLocalQueueLength(endpoint1) > metadata.GetLocalQueueLength(endpoint2) {
			output = endpoint1
		} else {
			output = endpoint2
		}
	}

	metadata.IncrementLocalQueueLength(output)

	return output
}

func kubernetesFirstAvailableLoadBalancing(metadata *function_metadata.FunctionMetadata) *function_metadata.UpstreamEndpoint {
	endpoints := metadata.GetUpstreamEndpoints()
	for _, endpoint := range endpoints {
		if len(endpoint.Capacity) > 0 {
			return endpoint
		}
	}

	logrus.Debug("No worker free, performing random choice")

	// If no node is available, we assign a random node
	return randomEndpoint(endpoints)
}

func kubernetesRoundRobinLoadBalancing(metadata *function_metadata.FunctionMetadata) *function_metadata.UpstreamEndpoint {
	endpoints := metadata.GetUpstreamEndpoints()

	baseIdx := metadata.GetKubernetesRoundRobinCounter()

	for len(endpoints[metadata.GetKubernetesRoundRobinCounter()].Capacity) == 0 {
		metadata.IncrementKubernetesRoundRobinCounter()

		if metadata.GetKubernetesRoundRobinCounter() == baseIdx {
			break
		}
	}

	idx := metadata.GetKubernetesRoundRobinCounter()

	// Check if we found a node
	if len(endpoints[idx].Capacity) > 0 {
		metadata.IncrementKubernetesRoundRobinCounter()
		return endpoints[idx]
	}

	logrus.Debug("No worker free, performing random choice")

	// If no node is available, we assign a random node
	return randomEndpoint(endpoints)
}

func kubernetesNativeLoadBalancing(metadata *function_metadata.FunctionMetadata) *function_metadata.UpstreamEndpoint {
	containerConcurrency := metadata.GetSandboxParallelism()

	if containerConcurrency == function_metadata.UnlimitedConcurrency {
		return bestOfTwoRandoms(metadata)
	} else if containerConcurrency <= 3 {
		return kubernetesFirstAvailableLoadBalancing(metadata)
	} else {
		return kubernetesRoundRobinLoadBalancing(metadata)
	}
}

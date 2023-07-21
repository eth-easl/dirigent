package load_balancing

import (
	"cluster_manager/internal/data_plane/function_metadata"
	"math/rand"
	"net/http"
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

func DoLoadBalancing(req *http.Request, metadata *function_metadata.FunctionMetadata, loadBalancingPolicy LoadBalancingPolicy) (bool, *function_metadata.UpstreamEndpoint, time.Duration, time.Duration) {
	start := time.Now()

	metadata.RLock()
	defer metadata.RUnlock()

	endpoints := metadata.GetUpstreamEndpoints()
	if endpoints == nil || len(endpoints) == 0 {
		return false, nil, time.Since(start), 0
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

	ccStart := time.Now()

	<-endpoint.Capacity // CC throttler

	ccDuration := time.Since(ccStart)

	req.URL.Scheme = "http"
	req.URL.Host = endpoint.URL
	logrus.Debug("Invocation forwarded to ", endpoint.URL)

	metadata.UpdateRequestMetadata(endpoint)

	return true, endpoint, time.Since(start), ccDuration
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
	minValue := requestCountPerInstance.AtomicGet(minEndpoint)

	for _, endpoint := range endpoints {
		countPerInstance := requestCountPerInstance.AtomicGet(endpoint)

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

	if containerConcurrency == function_metadata.UNLIMITED_CONCURENCY {
		return bestOfTwoRandoms(metadata)
	} else if containerConcurrency <= 3 {
		return kubernetesFirstAvailableLoadBalancing(metadata)
	} else {
		return kubernetesRoundRobinLoadBalancing(metadata)
	}
}

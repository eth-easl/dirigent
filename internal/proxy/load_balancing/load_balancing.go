package load_balancing

import (
	"cluster_manager/internal/common"
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

func DoLoadBalancing(req *http.Request, metadata *common.FunctionMetadata, loadBalancingPolicy LoadBalancingPolicy) (bool, *common.UpstreamEndpoint, time.Duration, time.Duration) {
	start := time.Now()

	metadata.RLock()
	defer metadata.RUnlock()

	endpoints := metadata.GetUpstreamEndpoints()
	if endpoints == nil || len(endpoints) == 0 {
		return false, nil, time.Since(start), 0
	}

	var endpoint *common.UpstreamEndpoint

	switch loadBalancingPolicy {
	case LOAD_BALANCING_RANDOM:
		endpoint = randomLoadBalancing(endpoints)
	case LOAD_BALANCING_ROUND_ROBIN:
		endpoint = roundRobinLoadBalancing(endpoints)
	case LOAD_BALANCING_LEAST_PROCESSED:
		endpoint = leastProcessedLoadBalancing(metadata, endpoints) // TODO: François Costa, check how to implement it & implement it
	case LOAD_BALANCING_KNATIVE:
		panic("Implement this") // TODO: François Costa, check how to implement it & implement it
	default:
		endpoint = randomLoadBalancing(endpoints)
	}

	ccStart := time.Now()

	<-endpoint.Capacity // CC throttler

	ccDuration := time.Since(ccStart)

	req.URL.Scheme = "http"
	req.URL.Host = endpoint.URL
	logrus.Debug("Invocation forwarded to ", endpoint.URL)

	metadata.GetRequestCountPerInstance()[endpoint]++

	return true, endpoint, time.Since(start), ccDuration
}

func randomLoadBalancing(endpoints []*common.UpstreamEndpoint) *common.UpstreamEndpoint {
	index := rand.Intn(len(endpoints))
	return endpoints[index]
}

var roundRobinCounterIndex int = 0 // TODO: Refactor this François Costa
func roundRobinLoadBalancing(endpoints []*common.UpstreamEndpoint) *common.UpstreamEndpoint {
	outputEndpoint := endpoints[roundRobinCounterIndex]
	roundRobinCounterIndex = (roundRobinCounterIndex + 1) % len(endpoints)

	return outputEndpoint
}

func leastProcessedLoadBalancing(metadata *common.FunctionMetadata, endpoints []*common.UpstreamEndpoint) *common.UpstreamEndpoint {
	requestCountPerInstance := metadata.GetRequestCountPerInstance()

	minEndpoint := endpoints[0]
	minValue := requestCountPerInstance[minEndpoint]

	for _, endpoint := range endpoints {
		countPerInstance := requestCountPerInstance[endpoint]

		if countPerInstance < minValue {
			minEndpoint = endpoint
			minValue = countPerInstance
		}
	}

	return minEndpoint
}

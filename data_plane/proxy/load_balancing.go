package proxy

import (
	"cluster_manager/common"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/semaphore"
	"math/rand"
	"net/http"
)

func DoLoadBalancing(req *http.Request, metadata *common.FunctionMetadata) (bool, *semaphore.Weighted) {
	metadata.RLock()
	defer metadata.RUnlock()

	endpoints := metadata.GetUpstreamEndpoints()
	if endpoints == nil || len(endpoints) == 0 {
		return false, nil
	}

	index := rand.Intn(len(endpoints))
	endpoint := endpoints[index]

	err := endpoint.Capacity.Acquire(req.Context(), 1)

	req.URL.Scheme = "http"
	req.URL.Host = endpoint.URL
	logrus.Debug("Invocation forwarded to ", endpoint.URL)

	if err == nil {
		return true, endpoint.Capacity
	} else {
		return false, nil
	}
}

package proxy

import (
	"cluster_manager/common"
	"github.com/sirupsen/logrus"
	"math/rand"
	"net/http"
)

func DoLoadBalancing(req *http.Request, metadata *common.FunctionMetadata) (bool, common.RequestThrottler) {
	metadata.RLock()
	defer metadata.RUnlock()

	endpoints := metadata.GetUpstreamEndpoints()
	if endpoints == nil || len(endpoints) == 0 {
		return false, nil
	}

	index := rand.Intn(len(endpoints))
	endpoint := endpoints[index]

	<-endpoint.Capacity // CC throttler

	req.URL.Scheme = "http"
	req.URL.Host = endpoint.URL
	logrus.Debug("Invocation forwarded to ", endpoint.URL)

	return true, endpoint.Capacity
}

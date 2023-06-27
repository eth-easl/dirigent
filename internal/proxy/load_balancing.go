package proxy

import (
	"cluster_manager/internal/common"
	"github.com/sirupsen/logrus"
	"math/rand"
	"net/http"
	"time"
)

func DoLoadBalancing(req *http.Request, metadata *common.FunctionMetadata) (bool, *common.UpstreamEndpoint, time.Duration, time.Duration) {
	start := time.Now()

	metadata.RLock()
	defer metadata.RUnlock()

	endpoints := metadata.GetUpstreamEndpoints()
	if endpoints == nil || len(endpoints) == 0 {
		return false, nil, time.Since(start), 0
	}

	index := rand.Intn(len(endpoints))
	endpoint := endpoints[index]

	ccStart := time.Now()
	<-endpoint.Capacity // CC throttler
	ccDuration := time.Since(ccStart)

	req.URL.Scheme = "http"
	req.URL.Host = endpoint.URL
	logrus.Debug("Invocation forwarded to ", endpoint.URL)

	return true, endpoint, time.Since(start), ccDuration
}

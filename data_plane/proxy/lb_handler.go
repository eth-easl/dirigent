package proxy

import (
	"cluster_manager/common"
	"context"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/semaphore"
	"math/rand"
	"net/http"
)

func lbPolicy(destinations []common.UpstreamEndpoint) (string, *semaphore.Weighted) {
	index := rand.Intn(len(destinations))
	endpoint := destinations[index]

	err := endpoint.Capacity.Acquire(context.Background(), 1)
	if err == nil {
		return endpoint.URL, endpoint.Capacity
	} else {
		return "", nil
	}
}

func LoadBalancingHandler(next http.Handler, cache *common.Deployments) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		serviceName := getServiceName(r)
		deployment := cache.GetDeployment(serviceName)
		if deployment == nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		endpoints := deployment.GetUpstreamEndpoints()
		if endpoints == nil || len(endpoints) == 0 {
			w.WriteHeader(http.StatusGone)
			logrus.Debug("Cold start passed by no sandbox available.")
			// TODO: make sure to implement drain logic
			return
		}

		destination, finished := lbPolicy(endpoints)
		if destination == "" {
			w.WriteHeader(http.StatusGone)
			return
		}
		defer finished.Release(1) // increase capacity on request completion

		r.URL.Scheme = "http"
		r.URL.Host = destination
		logrus.Debug("Invocation forwarded to ", destination)

		next.ServeHTTP(w, r)
	}
}

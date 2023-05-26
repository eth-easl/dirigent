package proxy

import (
	"cluster_manager/api/proto"
	"cluster_manager/common"
	"github.com/sirupsen/logrus"
	"net/http"
	"time"
)

func getServiceName(r *http.Request) string {
	return r.RequestURI
}

func InvocationHandler(next http.Handler, cache *common.Deployments, cp *proto.CpiInterfaceClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		serviceName := getServiceName(r)

		///////////////////////////////////////////////
		// METADATA FETCHING
		///////////////////////////////////////////////
		metadata := cache.GetDeployment(serviceName)
		if metadata == nil {
			// Request function has not been deployed
			w.WriteHeader(http.StatusNotFound)
			logrus.Trace("Invocation for non-existing service ", serviceName, " has been dumped.")

			return
		}
		logrus.Trace("Invocation for service ", serviceName, " has been received.")

		///////////////////////////////////////////////
		// COLD/WARM START
		///////////////////////////////////////////////
		coldStartChannel := metadata.TryWarmStart(cp)
		if coldStartChannel != nil {
			logrus.Debug("Enqueued invocation for ", serviceName)

			// wait until a cold start is resolved
			<-coldStartChannel
		}

		///////////////////////////////////////////////
		// LOAD BALANCING AND ROUTING
		///////////////////////////////////////////////
		lbSuccess, cc := DoLoadBalancing(r, metadata)
		if !lbSuccess {
			w.WriteHeader(http.StatusGone)
			logrus.Debug("Cold start passed by no sandbox available.")
			return
		}
		if cc != nil {
			defer cc.Release(1) // release CC on request complete
		}

		///////////////////////////////////////////////
		// SERVING
		///////////////////////////////////////////////
		next.ServeHTTP(w, r)

		///////////////////////////////////////////////
		// ON THE WAY BACK
		///////////////////////////////////////////////
		metadata.DecreaseInflight()

		end := time.Now()
		logrus.Info("Request took ", end.Sub(start).Microseconds(), " μs")
	}
}

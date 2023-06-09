package proxy

import (
	"cluster_manager/api/proto"
	"cluster_manager/common"
	"fmt"
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
		gotDeployment := time.Now()

		///////////////////////////////////////////////
		// COLD/WARM START
		///////////////////////////////////////////////
		coldStartChannel := metadata.TryWarmStart(cp)
		defer metadata.DecreaseInflight()
		if coldStartChannel != nil {
			logrus.Debug("Enqueued invocation for ", serviceName)

			// wait until a cold start is resolved
			<-coldStartChannel
		}
		passedColdstart := time.Now()

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
		passedLB := time.Now()

		///////////////////////////////////////////////
		// SERVING
		///////////////////////////////////////////////
		next.ServeHTTP(w, r)

		///////////////////////////////////////////////
		// ON THE WAY BACK
		///////////////////////////////////////////////
		end := time.Now()

		breakdown := fmt.Sprintf("Request took %d μs (deployment fetch: %d μs, cold start %d μs, load balancing %d μs, function execution %d μs)",
			end.Sub(start).Microseconds(),
			gotDeployment.Sub(start).Microseconds(),
			passedColdstart.Sub(gotDeployment).Microseconds(),
			passedLB.Sub(passedColdstart).Microseconds(),
			end.Sub(passedLB).Microseconds(),
		)
		logrus.Info(breakdown)
	}
}

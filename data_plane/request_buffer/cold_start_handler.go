package request_buffer

import (
	"cluster_manager/common"
	"github.com/sirupsen/logrus"
	"net/http"
)

func getServiceName(r *http.Request) string {
	return r.RequestURI
}

func ColdStartHandler(next http.Handler, cache *common.Deployments) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		serviceName := getServiceName(r)
		logrus.Debug("Invocation for service ", serviceName, " has been received.")

		metadata := cache.GetDeployment(serviceName)
		if metadata == nil {
			// Request function has not been deployed
			w.WriteHeader(http.StatusNotFound)
			return
		}

		coldStartChannel := metadata.TryWarmStart()
		if coldStartChannel != nil {
			logrus.Debug("Enqueued invocation for ", serviceName)

			// wait until a cold start is resolved
			<-coldStartChannel
		}

		next.ServeHTTP(w, r)
	}
}

package proxy

import (
	"cluster_manager/api/proto"
	common "cluster_manager/internal/data_plane/function_metadata"
	"cluster_manager/internal/data_plane/proxy/load_balancing"
	net2 "cluster_manager/internal/data_plane/proxy/net"
	"cluster_manager/pkg/tracing"
	"net"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

type ProxyingService struct {
	Host                string
	Port                string
	Cache               *common.Deployments
	CPInterface         *proto.CpiInterfaceClient
	LoadBalancingPolicy load_balancing.LoadBalancingPolicy

	Tracing *tracing.TracingService[tracing.ProxyLogEntry]
}

func NewProxyingService(host string, port string, cache *common.Deployments, cp *proto.CpiInterfaceClient, outputFile string, loadBalancingPolicy load_balancing.LoadBalancingPolicy) *ProxyingService {
	return &ProxyingService{
		Host:                host,
		Port:                port,
		Cache:               cache,
		CPInterface:         cp,
		LoadBalancingPolicy: loadBalancingPolicy,

		Tracing: tracing.NewProxyTracingService(outputFile),
	}
}

func (ps *ProxyingService) StartProxyServer() {
	proxy := net2.NewProxy()

	// function composition <=> [cold start handler -> forwarded shim handler -> proxy]
	var composedHandler http.Handler = proxy
	//composedHandler = ForwardedShimHandler(composedHandler)
	composedHandler = ps.createInvocationHandler(composedHandler, ps.Cache, ps.CPInterface)

	proxyRxAddress := net.JoinHostPort(ps.Host, ps.Port)
	logrus.Info("Creating a proxy server at ", proxyRxAddress)

	server := &http.Server{
		Addr:    proxyRxAddress,
		Handler: h2c.NewHandler(composedHandler, &http2.Server{}),
		// TODO: add timeout for each request on the gateway side
	}

	err := server.ListenAndServe()
	if err != nil {
		logrus.Fatalf("Failed to create a proxy server : (err : %s)", err.Error())
	}
}

func getServiceName(r *http.Request) string {
	// provided by the gRPC client through WithAuthority(...) call
	return r.Host
}

func (ps *ProxyingService) createInvocationHandler(next http.Handler, cache *common.Deployments, cp *proto.CpiInterfaceClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		///////////////////////////////////////////////
		// METADATA FETCHING
		///////////////////////////////////////////////
		serviceName := getServiceName(r)
		metadata, durationGetDeployment := cache.GetDeployment(serviceName)
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
		coldStartChannel, durationColdStart := metadata.TryWarmStart(cp)
		defer metadata.DecreaseInflight()

		if coldStartChannel != nil {
			logrus.Debug("Enqueued invocation for ", serviceName)

			// wait until a cold start is resolved
			coldStartWaitTime := time.Now()
			<-coldStartChannel
			/*if !waitForColdStartOrDie(coldStartChannel, metadata.GetIdentifier()) {
				w.WriteHeader(http.StatusRequestTimeout)
				return
			}*/
			durationColdStart = time.Since(coldStartWaitTime)
		}

		///////////////////////////////////////////////
		// LOAD BALANCING AND ROUTING
		///////////////////////////////////////////////
		lbSuccess, endpoint, durationLB, durationCC := load_balancing.DoLoadBalancing(r, metadata, ps.LoadBalancingPolicy)
		if !lbSuccess {
			w.WriteHeader(http.StatusGone)
			logrus.Debug("Cold start passed, but no sandbox available.")

			return
		}

		defer giveBackCCCapacity(endpoint)

		///////////////////////////////////////////////
		// PROXYING - LEAVING THE MACHINE
		///////////////////////////////////////////////
		startProxy := time.Now()

		next.ServeHTTP(w, r)
		///////////////////////////////////////////////
		// ON THE WAY BACK
		///////////////////////////////////////////////

		ps.Tracing.InputChannel <- tracing.ProxyLogEntry{
			ServiceName:   serviceName,
			ContainerID:   endpoint.ID,
			Total:         time.Since(start),
			GetMetadata:   durationGetDeployment,
			ColdStart:     durationColdStart,
			LoadBalancing: durationLB,
			CCThrottling:  durationCC,
			Proxying:      time.Since(startProxy),
		}

		defer metadata.DecrementLocalQueueLength(endpoint)
	}
}

func waitForColdStartOrDie(coldStartChannel chan struct{}, id string) bool {
	select {
	case <-coldStartChannel:
		logrus.Debug("Passed cold start channel for %s.", id)
		return true
	case <-time.After(time.Minute):
		logrus.Debugf("Canceled invocation for %s. Cold start timeout.", id)
		return false
	}
}

func giveBackCCCapacity(endpoint *common.UpstreamEndpoint) {
	endpointInflight := atomic.AddInt32(&endpoint.InFlight, -1)
	if atomic.LoadInt32(&endpoint.Drained) != 0 && endpointInflight == 0 {
		endpoint.DrainingCallback <- struct{}{}
	}

	if endpoint.Capacity != nil {
		endpoint.Capacity <- struct{}{}
	}
}

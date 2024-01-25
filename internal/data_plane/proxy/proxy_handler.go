package proxy

import (
	"cluster_manager/api/proto"
	common "cluster_manager/internal/data_plane/function_metadata"
	"cluster_manager/internal/data_plane/proxy/load_balancing"
	net2 "cluster_manager/internal/data_plane/proxy/net"
	"cluster_manager/internal/data_plane/proxy/requests"
	"cluster_manager/pkg/tracing"
	"cluster_manager/pkg/utils"
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

func NewProxyingService(port string, cache *common.Deployments, cp *proto.CpiInterfaceClient, outputFile string, loadBalancingPolicy load_balancing.LoadBalancingPolicy) *ProxyingService {
	return &ProxyingService{
		Host:                utils.Localhost,
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
	}

	err := server.ListenAndServe()
	if err != nil {
		logrus.Fatalf("Failed to create a proxy server : (err : %s)", err.Error())
	}
}

func (ps *ProxyingService) StartTracingService() {
	ps.Tracing.StartTracingService()
}

func (ps *ProxyingService) createInvocationHandler(next http.Handler, cache *common.Deployments, cp *proto.CpiInterfaceClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		///////////////////////////////////////////////
		// METADATA FETCHING
		///////////////////////////////////////////////
		serviceName := requests.GetServiceName(r)
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
		addDeploymentDuration := time.Duration(0)

		defer metadata.DecreaseInflight()
		go contextTerminationHandler(r, coldStartChannel)

		if coldStartChannel != nil {
			logrus.Debug("Enqueued invocation for ", serviceName)

			// wait until a cold start is resolved
			coldStartWaitTime := time.Now()
			waitOutcome := <-coldStartChannel
			durationColdStart = time.Since(coldStartWaitTime) - waitOutcome.AddEndpointDuration
			addDeploymentDuration = waitOutcome.AddEndpointDuration

			if waitOutcome.Outcome == common.CanceledColdStart {
				w.WriteHeader(http.StatusGatewayTimeout)
				logrus.Warnf("Invocation context canceled for %s.", serviceName)

				return
			}
		}

		///////////////////////////////////////////////
		// LOAD BALANCING AND ROUTING
		///////////////////////////////////////////////
		endpoint, durationLB, durationCC := load_balancing.DoLoadBalancing(r, metadata, ps.LoadBalancingPolicy)
		if endpoint == nil {
			w.WriteHeader(http.StatusGone)
			logrus.Warnf("Cold start passed, but no sandbox available for %s.", serviceName)

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
			AddDeployment: addDeploymentDuration,
			ColdStart:     durationColdStart,
			LoadBalancing: durationLB,
			CCThrottling:  durationCC,
			Proxying:      time.Since(startProxy),
		}
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

func contextTerminationHandler(r *http.Request, coldStartChannel chan common.ColdStartChannelStruct) {
	select {
	case <-r.Context().Done():
		if coldStartChannel != nil {
			coldStartChannel <- common.ColdStartChannelStruct{
				Outcome:             common.CanceledColdStart,
				AddEndpointDuration: 0,
			}
		}
	}
}

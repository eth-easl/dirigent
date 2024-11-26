/*
 * MIT License
 *
 * Copyright (c) 2024 EASL
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */

package proxy

import (
	"cluster_manager/api/proto"
	common "cluster_manager/internal/data_plane/function_metadata"
	"cluster_manager/internal/data_plane/proxy/load_balancing"
	net2 "cluster_manager/internal/data_plane/proxy/net"
	"cluster_manager/pkg/tracing"
	"cluster_manager/pkg/utils"
	"net"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

type ProxyingService struct {
	Host                string
	Port                string
	Cache               *common.Deployments
	CPInterface         proto.CpiInterfaceClient
	LoadBalancingPolicy load_balancing.LoadBalancingPolicy

	Tracing *tracing.TracingService[tracing.ProxyLogEntry]
}

func (ps *ProxyingService) GetCpApiServer() proto.CpiInterfaceClient {
	return ps.CPInterface
}

func (ps *ProxyingService) SetCpApiServer(client proto.CpiInterfaceClient) {
	ps.CPInterface = client
}

func NewProxyingService(port string, cache *common.Deployments, cp proto.CpiInterfaceClient, outputFile string, loadBalancingPolicy load_balancing.LoadBalancingPolicy) *ProxyingService {
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

	var composedHandler http.Handler = proxy
	composedHandler = ps.createInvocationHandler(composedHandler)

	mux := http.NewServeMux()
	mux.Handle("/", composedHandler)
	mux.HandleFunc("/health", HealthHandler)
	mux.HandleFunc("/metrics", CreateMetricsHandler(ps.Cache))

	proxyRxAddress := net.JoinHostPort(ps.Host, ps.Port)
	logrus.Info("Creating a proxy server at ", proxyRxAddress)

	server := &http.Server{
		Addr:    proxyRxAddress,
		Handler: h2c.NewHandler(mux, &http2.Server{}),
	}

	err := server.ListenAndServe()
	if err != nil {
		logrus.Fatalf("Failed to create a proxy server : (err : %s)", err.Error())
	}
}

func (ps *ProxyingService) StartTracingService() {
	ps.Tracing.StartTracingService()
}

// TODO: Deduplicate code
func (ps *ProxyingService) createInvocationHandler(next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, request *http.Request) {
		start := time.Now()

		///////////////////////////////////////////////
		// METADATA FETCHING
		///////////////////////////////////////////////
		serviceName := GetServiceName(request)
		metadata, durationGetDeployment := ps.Cache.GetDeployment(serviceName)
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
		coldStartChannel, durationColdStart := metadata.TryWarmStart(ps.CPInterface)
		addDeploymentDuration := time.Duration(0)

		defer metadata.GetStatistics().DecrementInflight() // for statistics
		defer metadata.DecreaseInflight()                  // for autoscaling

		// unblock from cold start if context gets cancelled
		go contextTerminationHandler(request, coldStartChannel)

		if coldStartChannel != nil {
			logrus.Debug("Enqueued invocation for ", serviceName)

			// wait until a cold start is resolved
			coldStartWaitTime := time.Now()
			waitOutcome := <-coldStartChannel
			metadata.GetStatistics().DecrementQueueDepth()
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
		endpoint, durationLB, durationCC := load_balancing.DoLoadBalancing(request, metadata, ps.LoadBalancingPolicy)
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

		next.ServeHTTP(w, request)
		///////////////////////////////////////////////
		// ON THE WAY BACK
		///////////////////////////////////////////////

		ps.Tracing.InputChannel <- tracing.ProxyLogEntry{
			ServiceName:      serviceName,
			ContainerID:      endpoint.ID,
			Total:            time.Since(start),
			GetMetadata:      durationGetDeployment,
			AddDeployment:    addDeploymentDuration,
			ColdStart:        durationColdStart,
			LoadBalancing:    durationLB,
			CCThrottling:     durationCC,
			Proxying:         time.Since(startProxy),
			PersistenceLayer: 0,
		}
		metadata.GetStatistics().IncrementSuccessfulInvocations()
	}
}

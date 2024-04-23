package proxy

import (
	"bytes"
	common "cluster_manager/internal/data_plane/function_metadata"
	"cluster_manager/internal/data_plane/proxy/load_balancing"
	"cluster_manager/internal/data_plane/proxy/requests"
	"cluster_manager/pkg/tracing"
	"cluster_manager/proto"
	"fmt"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"net"
	"net/http"
	"time"
)

type proxyContext struct {
	host string
	port string

	cache               *common.Deployments
	cpInterface         proto.CpiInterfaceClient
	loadBalancingPolicy load_balancing.LoadBalancingPolicy

	tracing *tracing.TracingService[tracing.ProxyLogEntry]
}

type requestMetadata struct {
	start            time.Time
	serialization    time.Duration
	persistenceLayer time.Duration
}

func startProxy(handler http.HandlerFunc, context proxyContext, host string, port string) {
	mux := http.NewServeMux()
	mux.Handle("/", handler)
	mux.HandleFunc("/health", HealthHandler)
	mux.HandleFunc("/metrics", CreateMetricsHandler(context.cache))

	proxyRxAddress := net.JoinHostPort(host, port)
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

func proxyHandler(request *http.Request, requestMetadata requestMetadata, proxyContext proxyContext) *requests.BufferedResponse {

	///////////////////////////////////////////////
	// METADATA FETCHING
	///////////////////////////////////////////////
	serviceName := GetServiceName(request)
	metadata, durationGetDeployment := proxyContext.cache.GetDeployment(serviceName)
	if metadata == nil {
		logrus.Trace("Invocation for non-existing service ", serviceName, " has been dumped.")
		return &requests.BufferedResponse{
			StatusCode: http.StatusBadRequest,
			Body:       "Invocation for non-existing service " + serviceName + " has been dumped.",
		}
	}

	logrus.Trace("Invocation for service ", serviceName, " has been received.")

	///////////////////////////////////////////////
	// COLD/WARM START
	///////////////////////////////////////////////
	coldStartChannel, durationColdStart := metadata.TryWarmStart(proxyContext.cpInterface)
	addDeploymentDuration := time.Duration(0)

	defer metadata.GetStatistics().DecrementInflight()

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

		// TODO: Resend the request in the channel & execute until it works
		if waitOutcome.Outcome == common.CanceledColdStart {
			return &requests.BufferedResponse{
				StatusCode: http.StatusGatewayTimeout,
				Body:       fmt.Sprintf("Invocation context canceled for %s.", serviceName),
			}
		}
	}

	///////////////////////////////////////////////
	// LOAD BALANCING AND ROUTING
	///////////////////////////////////////////////
	endpoint, durationLB, durationCC := load_balancing.DoLoadBalancing(request, metadata, proxyContext.loadBalancingPolicy)
	if endpoint == nil {

		return &requests.BufferedResponse{
			StatusCode: http.StatusGone,
			Body:       fmt.Sprintf("Cold start passed, but no sandbox available for %s.", serviceName),
		}
	}

	defer giveBackCCCapacity(endpoint)

	///////////////////////////////////////////////
	// PROXYING - LEAVING THE MACHINE
	///////////////////////////////////////////////
	startProxy := time.Now()

	request.RequestURI = ""
	workerResponse, err := http.DefaultClient.Do(request)
	if err != nil {
		logrus.Infof(err.Error())
		return &requests.BufferedResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       err.Error(),
		}
	}

	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(workerResponse.Body); err != nil {
		return &requests.BufferedResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       err.Error(),
		}
	}

	///////////////////////////////////////////////
	// ON THE WAY BACK
	///////////////////////////////////////////////

	proxyContext.tracing.InputChannel <- tracing.ProxyLogEntry{
		ServiceName:      serviceName,
		ContainerID:      endpoint.ID,
		Total:            time.Since(requestMetadata.start),
		GetMetadata:      durationGetDeployment,
		AddDeployment:    addDeploymentDuration,
		ColdStart:        durationColdStart,
		LoadBalancing:    durationLB,
		CCThrottling:     durationCC,
		Proxying:         time.Since(startProxy),
		Serialization:    requestMetadata.serialization,
		PersistenceLayer: requestMetadata.persistenceLayer,
	}
	metadata.GetStatistics().IncrementSuccessfulInvocations()

	return &requests.BufferedResponse{
		StatusCode: http.StatusOK,
		Body:       buf.String(),
	}
}

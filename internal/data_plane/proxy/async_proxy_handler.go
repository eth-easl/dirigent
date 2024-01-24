package proxy

import "C"
import (
	"bytes"
	"cluster_manager/api/proto"
	common "cluster_manager/internal/data_plane/function_metadata"
	"cluster_manager/internal/data_plane/proxy/load_balancing"
	net2 "cluster_manager/internal/data_plane/proxy/net"
	"cluster_manager/internal/data_plane/proxy/requests"
	"cluster_manager/pkg/tracing"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

type AsyncProxyingService struct {
	Host                string
	Port                string
	Cache               *common.Deployments
	CPInterface         *proto.CpiInterfaceClient
	LoadBalancingPolicy load_balancing.LoadBalancingPolicy

	Tracing *tracing.TracingService[tracing.ProxyLogEntry]

	RequestChannel chan *requests.BufferedRequest
	Responses      map[string]*requests.BufferedResponse
}

func NewAsyncProxyingService(host string, port string, cache *common.Deployments, cp *proto.CpiInterfaceClient, outputFile string, loadBalancingPolicy load_balancing.LoadBalancingPolicy) *AsyncProxyingService {
	return &AsyncProxyingService{
		Host:                host,
		Port:                port,
		Cache:               cache,
		CPInterface:         cp,
		LoadBalancingPolicy: loadBalancingPolicy,

		Tracing: tracing.NewProxyTracingService(outputFile),

		RequestChannel: make(chan *requests.BufferedRequest),
		Responses:      make(map[string]*requests.BufferedResponse),
	}
}

func (ps *AsyncProxyingService) StartAsyncProxyserver() {
	proxy := net2.NewProxy()

	// function composition <=> [cold start handler -> forwarded shim handler -> proxy]
	var composedHandler http.Handler = proxy

	composedHandler = ps.createAsyncInvocationHandler(composedHandler, ps.Cache, ps.CPInterface)

	proxyRxAddress := net.JoinHostPort(ps.Host, ps.Port)
	logrus.Info("Creating a proxy server at ", proxyRxAddress)

	requestServer := &http.Server{
		Addr:    proxyRxAddress,
		Handler: h2c.NewHandler(composedHandler, &http2.Server{}),
	}

	readServer := http.Server{
		Addr:    proxyRxAddress,
		Handler: h2c.NewHandler(composedHandler, &http2.Server{}),
	}

	go func() {
		if err := requestServer.ListenAndServe(); err != nil {
			logrus.Fatalf("Failed to create a proxy server : (err : %s)", err.Error())
		}
	}()

	go func() {
		if err := readServer.ListenAndServe(); err != nil {
			logrus.Fatalf("Failed to create a async proxy server : (err : %s)", err.Error())
		}
	}()

	ps.asyncRequestHandler()
}

func (ps *AsyncProxyingService) createAsyncInvocationHandler(next http.Handler, cache *common.Deployments, cp *proto.CpiInterfaceClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		code := requests.GetUniqueRequestCode()
		ps.RequestChannel <- requests.BufferedRequestFromRequest(r, code)

		w.WriteHeader(http.StatusOK)
		io.Copy(w, strings.NewReader(code+"\n"))
	}
}

func (ps *AsyncProxyingService) asyncRequestHandler() {
	for {
		bufferedRequest := <-ps.RequestChannel
		go func(bufferedRequest *requests.BufferedRequest) {
			response := ps.fireRequest(requests.RequestFromBufferedRequest(bufferedRequest))
			ps.Responses[bufferedRequest.Code] = response
		}(bufferedRequest)
	}
}

func (ps *AsyncProxyingService) fireRequest(request *http.Request) *requests.BufferedResponse {
	start := time.Now()

	///////////////////////////////////////////////
	// METADATA FETCHING
	///////////////////////////////////////////////
	serviceName := getServiceName(request)
	metadata, durationGetDeployment := ps.Cache.GetDeployment(serviceName)
	if metadata == nil {
		logrus.Trace("Invocation for non-existing service ", serviceName, " has been dumped.")
		return &requests.BufferedResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       "Invocation for non-existing service " + serviceName + " has been dumped.",
		}
	}

	logrus.Trace("Invocation for service ", serviceName, " has been received.")

	///////////////////////////////////////////////
	// COLD/WARM START
	///////////////////////////////////////////////
	coldStartChannel, durationColdStart := metadata.TryWarmStart(ps.CPInterface)
	addDeploymentDuration := time.Duration(0)

	defer metadata.DecreaseInflight()
	go contextTerminationHandler(request, coldStartChannel)

	if coldStartChannel != nil {
		logrus.Debug("Enqueued invocation for ", serviceName)

		// wait until a cold start is resolved
		coldStartWaitTime := time.Now()
		waitOutcome := <-coldStartChannel
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
	endpoint, durationLB, durationCC := load_balancing.DoLoadBalancing(request, metadata, ps.LoadBalancingPolicy)
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

	workerResponse, err := http.DefaultClient.Do(request)
	if err != nil {
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

	return &requests.BufferedResponse{
		StatusCode: http.StatusOK,
		Body:       buf.String(),
	}
}

func (ps *AsyncProxyingService) createAsyncReadHandler(next http.Handler, cache *common.Deployments, cp *proto.CpiInterfaceClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logrus.Tracef("[reverse proxy server] received code request at: %s\n", time.Now())

		buf := new(bytes.Buffer)
		buf.ReadFrom(r.Body)

		if val, ok := ps.Responses[buf.String()]; ok {
			w.WriteHeader(http.StatusOK)
			requests.FillResponseWithBufferedResponse(w, val)
		} else {
			w.WriteHeader(http.StatusNoContent)
		}
	}
}

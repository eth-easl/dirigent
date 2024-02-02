package proxy

import (
	"bytes"
	"cluster_manager/api/proto"
	common "cluster_manager/internal/data_plane/function_metadata"
	"cluster_manager/internal/data_plane/proxy/load_balancing"
	request_persistence "cluster_manager/internal/data_plane/proxy/persistence"
	"cluster_manager/internal/data_plane/proxy/requests"
	"cluster_manager/pkg/config"
	"cluster_manager/pkg/tracing"
	"cluster_manager/pkg/utils"
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

type AsyncProxyingService struct {
	Host                string
	Port                string
	PortRead            string
	Cache               *common.Deployments
	CPInterface         *proto.CpiInterfaceClient
	LoadBalancingPolicy load_balancing.LoadBalancingPolicy

	Tracing *tracing.TracingService[tracing.ProxyLogEntry]

	RequestChannel chan *requests.BufferedRequest
	Responses      map[string]*requests.BufferedResponse

	Persistence    request_persistence.RequestPersistence
	AllowedRetries int
}

func NewAsyncProxyingService(cfg config.DataPlaneConfig, cache *common.Deployments, cp *proto.CpiInterfaceClient, outputFile string, loadBalancingPolicy load_balancing.LoadBalancingPolicy) *AsyncProxyingService {
	var persistenceLayer request_persistence.RequestPersistence
	var err error

	if cfg.PersistRequests {
		persistenceLayer, err = request_persistence.CreateRequestRedisClient(context.Background(), cfg.RedisConf)
		if err != nil {
			return nil
		}
	} else {
		persistenceLayer = request_persistence.CreateEmptyRequestPersistence()
	}

	return &AsyncProxyingService{
		Host:                utils.Localhost,
		Port:                cfg.PortProxy,
		PortRead:            cfg.PortProxyRead,
		Cache:               cache,
		CPInterface:         cp,
		LoadBalancingPolicy: loadBalancingPolicy,

		Tracing: tracing.NewProxyTracingService(outputFile),

		RequestChannel: make(chan *requests.BufferedRequest),
		Responses:      make(map[string]*requests.BufferedResponse),

		Persistence:    persistenceLayer,
		AllowedRetries: 3, // TODO: Remove hard code
	}
}

func (ps *AsyncProxyingService) StartProxyServer() {
	go func() {
		composedHandler := ps.createAsyncInvocationHandler()

		proxyRxAddress := net.JoinHostPort(ps.Host, ps.Port)
		logrus.Info("Creating a proxy server at ", proxyRxAddress)

		requestServer := &http.Server{
			Addr:    proxyRxAddress,
			Handler: h2c.NewHandler(composedHandler, &http2.Server{}),
		}
		if err := requestServer.ListenAndServe(); err != nil {
			logrus.Fatalf("Failed to create a proxy server : (err : %s)", err.Error())
		}
	}()

	go func() {
		readRequestHandler := ps.createAsyncReadHandler()

		proxyRxReadAddress := net.JoinHostPort(ps.Host, ps.PortRead)
		logrus.Info("Creating a read proxy server at ", proxyRxReadAddress)

		readServer := http.Server{
			Addr:    proxyRxReadAddress,
			Handler: h2c.NewHandler(readRequestHandler, &http2.Server{}),
		}

		if err := readServer.ListenAndServe(); err != nil {
			logrus.Fatalf("Failed to create a async proxy server : (err : %s)", err.Error())
		}
	}()

	ps.asyncRequestHandler()
}

func (ps *AsyncProxyingService) StartTracingService() {
	ps.Tracing.StartTracingService()
}

func (ps *AsyncProxyingService) createAsyncInvocationHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		code := requests.GetUniqueRequestCode()
		bufferedRequest := requests.BufferedRequestFromRequest(r, code)

		err := ps.Persistence.PersistBufferedRequest(context.Background(), bufferedRequest)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			io.Copy(w, strings.NewReader(err.Error()))
			return
		}

		ps.RequestChannel <- bufferedRequest

		logrus.Tracef("[reverse proxy server] received request and generated code :%s\n", code)

		w.WriteHeader(http.StatusOK)
		io.Copy(w, strings.NewReader(code))
	}
}

func (ps *AsyncProxyingService) asyncRequestHandler() {
	for {
		bufferedRequest := <-ps.RequestChannel
		go func(bufferedRequest *requests.BufferedRequest) {
			logrus.Tracef("[reverse proxy server] firing a request")

			if bufferedRequest.NumberTries >= ps.AllowedRetries {
				ps.Responses[bufferedRequest.Code] = &requests.BufferedResponse{
					StatusCode: http.StatusInternalServerError,
					Body:       "Server failed many times",
				}
			} else {
				response := ps.fireRequest(requests.RequestFromBufferedRequest(bufferedRequest))
				if response.StatusCode == http.StatusOK || response.StatusCode == http.StatusBadRequest {
					ps.Responses[bufferedRequest.Code] = response
				} else {
					bufferedRequest.NumberTries++
					ps.RequestChannel <- bufferedRequest
				}
			}
		}(bufferedRequest)
	}
}

func (ps *AsyncProxyingService) fireRequest(request *http.Request) *requests.BufferedResponse {
	start := time.Now()
	///////////////////////////////////////////////
	// METADATA FETCHING
	///////////////////////////////////////////////
	serviceName := requests.GetServiceName(request)
	metadata, durationGetDeployment := ps.Cache.GetDeployment(serviceName)
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

func (ps *AsyncProxyingService) createAsyncReadHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		buf := new(bytes.Buffer)
		buf.ReadFrom(r.Body)

		logrus.Tracef("[reverse proxy server] received code request with code : %s\n", buf.String())

		if val, ok := ps.Responses[buf.String()]; ok {
			w.WriteHeader(http.StatusOK)
			requests.FillResponseWithBufferedResponse(w, val)
		} else {
			w.WriteHeader(http.StatusNoContent)
		}
	}
}

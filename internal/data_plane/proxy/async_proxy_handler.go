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
	"sync"
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
	ResponsesLock  sync.RWMutex

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

	proxy := &AsyncProxyingService{
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
		AllowedRetries: cfg.NumberRetries,
	}

	go proxy.startGarbageCollector()

	return proxy
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

	// Load responses and requests and resend requests without response
	responses, err := ps.Persistence.ScanBufferedResponses(context.Background())
	if err != nil {
		logrus.Fatalf("Failed to recover responses : error %s", err.Error())
	}

	ps.ResponsesLock.Lock()
	for _, response := range responses {
		ps.Responses[response.Code] = response
	}
	ps.ResponsesLock.Unlock()

	bufferedRequests, err := ps.Persistence.ScanBufferedRequests(context.Background())
	if err != nil {
		logrus.Fatalf("Failed to recover requests : error")
	}

	ps.ResponsesLock.RLock()
	for _, request := range bufferedRequests {
		if _, ok := ps.Responses[request.Code]; !ok {
			ps.RequestChannel <- request
		}
	}
	ps.ResponsesLock.RUnlock()

	ps.asyncRequestHandler()
}

func (ps *AsyncProxyingService) StartTracingService() {
	ps.Tracing.StartTracingService()
}

func (ps *AsyncProxyingService) createAsyncInvocationHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		code := requests.GetUniqueRequestCode()
		bufferedRequest := requests.BufferedRequestFromRequest(r, code)

		var err error
		err, bufferedRequest.SerializationDuration, bufferedRequest.PersistenceDuration = ps.Persistence.PersistBufferedRequest(context.Background(), bufferedRequest)
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
				ps.ResponsesLock.Lock()
				ps.Responses[bufferedRequest.Code] = &requests.BufferedResponse{
					StatusCode: http.StatusInternalServerError,
					Body:       "Server failed many times",
				}
				ps.ResponsesLock.Unlock()
			} else {
				response := ps.fireRequest(bufferedRequest)
				if response.StatusCode == http.StatusOK || response.StatusCode == http.StatusBadRequest {
					response.Timestamp = time.Now()
					response.Code = bufferedRequest.Code
					ps.ResponsesLock.Lock()
					ps.Responses[bufferedRequest.Code] = response
					ps.ResponsesLock.Unlock()

					if err := ps.Persistence.PersistBufferedResponse(context.Background(), response); err != nil {
						logrus.Warnf("Failed to buffer response, error : %s", err.Error())
					}

					if err := ps.Persistence.DeleteBufferedRequest(context.Background(), bufferedRequest.Code); err != nil {
						logrus.Warnf("Failed to delete buffered request, error : %s", err.Error())
					}
				} else {
					bufferedRequest.NumberTries++
					ps.RequestChannel <- bufferedRequest
				}
			}
		}(bufferedRequest)
	}
}

// TODO: Deduplicate code
func (ps *AsyncProxyingService) fireRequest(bufferedRequest *requests.BufferedRequest) *requests.BufferedResponse {
	request := requests.RequestFromBufferedRequest(bufferedRequest)

	///////////////////////////////////////////////
	// METADATA FETCHING
	///////////////////////////////////////////////
	serviceName := GetServiceName(request)
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
		ServiceName:      serviceName,
		ContainerID:      endpoint.ID,
		Total:            time.Since(bufferedRequest.Start),
		GetMetadata:      durationGetDeployment,
		AddDeployment:    addDeploymentDuration,
		ColdStart:        durationColdStart,
		LoadBalancing:    durationLB,
		CCThrottling:     durationCC,
		Proxying:         time.Since(startProxy),
		Serialization:    bufferedRequest.SerializationDuration,
		PersistenceLayer: bufferedRequest.PersistenceDuration,
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

		ps.ResponsesLock.RLock()
		if val, ok := ps.Responses[buf.String()]; ok {
			w.WriteHeader(http.StatusOK)
			requests.FillResponseWithBufferedResponse(w, val)
		} else {
			w.WriteHeader(http.StatusNoContent)
		}
		ps.ResponsesLock.RUnlock()
	}
}

func (ps *AsyncProxyingService) startGarbageCollector() {
	for {
		time.Sleep(15 * time.Minute)

		ps.ResponsesLock.Lock()
		for key, response := range ps.Responses {
			if time.Now().Unix()-response.Timestamp.Unix() > time.Minute.Nanoseconds() {
				delete(ps.Responses, key)
			}
		}
		ps.ResponsesLock.Unlock()
	}
}
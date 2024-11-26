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
	CPInterface         proto.CpiInterfaceClient
	LoadBalancingPolicy load_balancing.LoadBalancingPolicy

	Tracing *tracing.TracingService[tracing.ProxyLogEntry]

	RequestChannel chan *requests.BufferedRequest
	Responses      map[string]*requests.BufferedResponse
	ResponsesLock  sync.RWMutex

	Persistence    request_persistence.RequestPersistence
	AllowedRetries int
}

func (ps *AsyncProxyingService) GetCpApiServer() proto.CpiInterfaceClient {
	return ps.CPInterface
}

func (ps *AsyncProxyingService) SetCpApiServer(client proto.CpiInterfaceClient) {
	ps.CPInterface = client
}

func NewAsyncProxyingService(cfg config.DataPlaneConfig, cache *common.Deployments, cp proto.CpiInterfaceClient, outputFile string, loadBalancingPolicy load_balancing.LoadBalancingPolicy) *AsyncProxyingService {
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

		mux := http.NewServeMux()
		mux.Handle("/", composedHandler)
		mux.HandleFunc("/health", HealthHandler)
		mux.HandleFunc("/metrics", CreateMetricsHandler(ps.Cache))

		proxyRxAddress := net.JoinHostPort(ps.Host, ps.Port)
		logrus.Info("Creating a proxy server at ", proxyRxAddress)

		requestServer := &http.Server{
			Addr:    proxyRxAddress,
			Handler: h2c.NewHandler(mux, &http2.Server{}),
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

	go ps.asyncRequestHandler()

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
	metadata.GetStatistics().IncrementSuccessfulInvocations()

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

package proxy

import (
	"bytes"
	common "cluster_manager/internal/data_plane/function_metadata"
	"cluster_manager/internal/data_plane/proxy/load_balancing"
	"cluster_manager/internal/data_plane/proxy/metrics_collection"
	request_persistence "cluster_manager/internal/data_plane/proxy/persistence"
	"cluster_manager/internal/data_plane/proxy/requests"
	"cluster_manager/pkg/config"
	"cluster_manager/pkg/tracing"
	"cluster_manager/pkg/utils"
	"cluster_manager/proto"
	"context"
	"github.com/sirupsen/logrus"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type AsyncProxyingService struct {
	Host     string
	Port     string
	PortRead string

	Context proxyContext

	RequestChannel chan *requests.BufferedRequest
	Responses      map[string]*requests.BufferedResponse
	ResponsesLock  sync.RWMutex

	Persistence    request_persistence.RequestPersistence
	AllowedRetries int
}

func (ps *AsyncProxyingService) GetCpApiServer() proto.CpiInterfaceClient {
	return ps.Context.cpInterface
}

func (ps *AsyncProxyingService) SetCpApiServer(client proto.CpiInterfaceClient) {
	ps.Context.cpInterface = client
}

func NewAsyncProxyingService(cfg config.DataPlaneConfig, cache *common.Deployments, cp proto.CpiInterfaceClient, outputFile string, loadBalancingPolicy load_balancing.LoadBalancingPolicy) *AsyncProxyingService {
	var persistenceLayer request_persistence.RequestPersistence
	var err error

	if cfg.PersistRequests {
		persistenceLayer, err = request_persistence.CreateRequestRedisClient(context.Background(), cfg.RedisConf)
		if err != nil {
			logrus.Fatalf("Failed to connect to redis client %s", err.Error())
		}
	} else {
		persistenceLayer = request_persistence.CreateEmptyRequestPersistence()
	}

	incomingRequestChannel, doneRequestChannel := metrics_collection.NewMetricsCollector(cfg.ControlPlaneNotifyIntervalInMinutes, cp)

	proxy := &AsyncProxyingService{
		Host:     utils.Localhost,
		Port:     cfg.PortProxy,
		PortRead: cfg.PortProxyRead,

		Context: proxyContext{
			cache:               cache,
			cpInterface:         cp,
			loadBalancingPolicy: loadBalancingPolicy,

			tracing: tracing.NewProxyTracingService(outputFile),

			incomingRequestChannel: incomingRequestChannel,
			doneRequestChannel:     doneRequestChannel,
		},

		RequestChannel: make(chan *requests.BufferedRequest),
		Responses:      make(map[string]*requests.BufferedResponse),

		Persistence:    persistenceLayer,
		AllowedRetries: cfg.NumberRetries,
	}

	go proxy.startGarbageCollector()

	return proxy
}

func (ps *AsyncProxyingService) StartProxyServer() {
	go startProxy(ps.createAsyncInvocationHandler(), ps.Context, ps.Host, ps.Port)
	go startProxy(ps.createAsyncResponseHandler(), ps.Context, ps.Host, ps.PortRead)

	// Load responses and requests and resend requests without response
	responses, err := ps.Persistence.ScanBufferedResponses(context.Background())
	if err != nil {
		logrus.Fatalf("Failed to recover responses : error %s", err.Error())
	}

	ps.ResponsesLock.Lock()
	for _, response := range responses {
		ps.Responses[response.UniqueCodeIdentifier] = response
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
	ps.Context.tracing.StartTracingService()
}

func (ps *AsyncProxyingService) createAsyncInvocationHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		code := requests.GetUniqueRequestCode()
		bufferedRequest := requests.BufferedRequestFromRequest(r, code)

		var err error
		err, bufferedRequest.SerializationDuration, bufferedRequest.PersistenceDuration = ps.Persistence.PersistBufferedRequest(context.Background(), bufferedRequest)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = io.Copy(w, strings.NewReader(err.Error()))
			return
		}

		ps.RequestChannel <- bufferedRequest

		logrus.Tracef("[reverse proxy server] received request and generated code :%s\n", code)

		w.WriteHeader(http.StatusOK)
		_, _ = io.Copy(w, strings.NewReader(code))
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
				response := ps.triggerRequest(bufferedRequest)
				if response.StatusCode == http.StatusOK || response.StatusCode == http.StatusBadRequest {
					response.Timestamp = time.Now()
					response.UniqueCodeIdentifier = bufferedRequest.Code
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

func (ps *AsyncProxyingService) triggerRequest(bufferedRequest *requests.BufferedRequest) *requests.BufferedResponse {
	return proxyHandler(
		&bufferedRequest.Request,
		requestMetadata{
			start:            bufferedRequest.Start,
			serialization:    bufferedRequest.SerializationDuration,
			persistenceLayer: bufferedRequest.PersistenceDuration,
		},
		ps.Context,
	)
}

func (ps *AsyncProxyingService) createAsyncResponseHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		buf := new(bytes.Buffer)

		_, err := buf.ReadFrom(r.Body)
		if err != nil {
			logrus.Errorf("[reverse proxy server] failed to read response body, error : %s", err.Error())
		}

		logrus.Tracef("[reverse proxy server] received code request with code : %s", buf.String())

		ps.ResponsesLock.RLock()
		defer ps.ResponsesLock.RUnlock()

		if val, ok := ps.Responses[buf.String()]; ok {
			w.Header().Set("Duration-Microseconds", strconv.FormatInt(val.E2ELatency.Microseconds(), 10))
			_ = requests.FillResponseWithBufferedResponse(w, val)
		} else {
			_, _ = w.Write([]byte("Response with the provided ID was not found."))
			w.WriteHeader(http.StatusBadRequest)
		}
	}
}

func (ps *AsyncProxyingService) startGarbageCollector() {
	for {
		time.Sleep(15 * time.Minute)
		ps.garbageCollectorCycle()
	}
}

func (ps *AsyncProxyingService) garbageCollectorCycle() {
	ps.ResponsesLock.Lock()
	defer ps.ResponsesLock.Unlock()

	for key, response := range ps.Responses {
		if time.Now().Sub(response.Timestamp) > time.Minute {
			delete(ps.Responses, key)
		}
	}
}

package proxy

import (
	"bytes"
	common "cluster_manager/internal/data_plane/function_metadata"
	"cluster_manager/internal/data_plane/proxy/load_balancing"
	"cluster_manager/internal/data_plane/proxy/metrics_collection"
	rp "cluster_manager/internal/data_plane/proxy/persistence"
	"cluster_manager/internal/data_plane/proxy/requests"
	"cluster_manager/internal/data_plane/proxy/reverse_proxy"
	"cluster_manager/pkg/config"
	"cluster_manager/pkg/tracing"
	"cluster_manager/pkg/utils"
	"cluster_manager/proto"
	"context"
	"github.com/sirupsen/logrus"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"time"
)

const AsyncRequestBufferSize = 1000

type AsyncProxyingService struct {
	ProxyingService

	PortRead string

	RequestChannel chan *requests.BufferedRequest
	Responses      *sync.Map

	Persistence    rp.RequestPersistence
	AllowedRetries int
}

func (ps *AsyncProxyingService) GetCpApiServer() proto.CpiInterfaceClient {
	return ps.Context.cpInterface
}

func (ps *AsyncProxyingService) SetCpApiServer(client proto.CpiInterfaceClient) {
	ps.Context.cpInterface = client
}

func NewAsyncProxyingService(cfg config.DataPlaneConfig, cache *common.Deployments, cp proto.CpiInterfaceClient, outputFile string, loadBalancingPolicy load_balancing.LoadBalancingPolicy) *AsyncProxyingService {
	var persistenceLayer rp.RequestPersistence
	var err error

	if cfg.PersistRequests {
		persistenceLayer, err = rp.CreateRequestRedisClient(context.Background(), cfg.RedisConf)
		if err != nil {
			logrus.Fatalf("Failed to connect to redis client %s", err.Error())
		}
	} else {
		persistenceLayer = rp.CreateEmptyRequestPersistence()
	}

	incomingRequestChannel, doneRequestChannel := metrics_collection.NewMetricsCollector(cfg.ControlPlaneNotifyIntervalInMinutes, cp)

	proxy := &AsyncProxyingService{
		ProxyingService: ProxyingService{
			Host: utils.Localhost,
			Port: cfg.PortProxy,
			Context: proxyContext{
				cache:               cache,
				cpInterface:         cp,
				loadBalancingPolicy: loadBalancingPolicy,

				tracing: tracing.NewProxyTracingService(outputFile),

				incomingRequestChannel: incomingRequestChannel,
				doneRequestChannel:     doneRequestChannel,
			},
		},

		PortRead: cfg.PortProxyRead,

		RequestChannel: make(chan *requests.BufferedRequest, AsyncRequestBufferSize),
		Responses:      &sync.Map{},

		Persistence:    persistenceLayer,
		AllowedRetries: cfg.NumberRetries,
	}

	go proxy.startGarbageCollector()

	return proxy
}

func (ps *AsyncProxyingService) populateResponseStructures() {
	// Load responses and requests and resend requests without response
	responses, err := ps.Persistence.ScanBufferedResponses(context.Background())
	if err != nil {
		logrus.Fatalf("Failed to recover responses : error %s", err.Error())
	}

	for _, response := range responses {
		// thread-safe as it executed before all handlers are started
		ps.Responses.Store(response.UniqueCodeIdentifier, response)
	}
}

func (ps *AsyncProxyingService) submitBufferedRequests() {
	bufferedRequests, err := ps.Persistence.ScanBufferedRequests(context.Background())
	if err != nil {
		logrus.Fatalf("Failed to recover requests : error - %s", err.Error())
	}

	for _, request := range bufferedRequests {
		if _, ok := ps.Responses.Load(request.Code); !ok {
			ps.RequestChannel <- request
		}
	}
}

func (ps *AsyncProxyingService) StartProxyServer() {
	ps.populateResponseStructures()

	go startProxy(ps.createAsyncInvocationHandler(), ps.Context, ps.Host, ps.Port)
	go startProxy(ps.createAsyncResponseHandler(), ps.Context, ps.Host, ps.PortRead)
	go ps.asyncRequestHandler()

	ps.submitBufferedRequests()
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
			logrus.Errorf("Error while persistint async request to a buffer - %v", err)

			_, err = io.Copy(w, strings.NewReader(err.Error()))
			if err != nil {
				logrus.Errorf("Error composing response body while persisting async request - %v", err)
			}

			return
		}

		ps.RequestChannel <- bufferedRequest

		logrus.Tracef("[reverse proxy server] received request and generated code :%s\n", code)

		w.WriteHeader(http.StatusOK)
		_, err = io.Copy(w, strings.NewReader(code))
		if err != nil {
			logrus.Errorf("Error composing response body with async request ID - %v", err)
		}
	}
}

func (ps *AsyncProxyingService) declareResponseAsFailed(bufferedRequest *requests.BufferedRequest) {
	ps.Responses.Store(bufferedRequest.Code, &requests.BufferedResponse{
		StatusCode: http.StatusInternalServerError,
		Body:       "Too many retries. Request failed.",
	})
}

func (ps *AsyncProxyingService) declareResponseAsSuccessful(bufferedRequest *requests.BufferedRequest, response *requests.BufferedResponse) {
	response.Timestamp = time.Now()
	response.UniqueCodeIdentifier = bufferedRequest.Code

	if err := ps.Persistence.DeleteBufferedRequest(context.Background(), bufferedRequest.Code); err != nil {
		logrus.Errorf("Failed to delete buffered request, error : %s", err.Error())
	}

	response.E2ELatency = time.Since(bufferedRequest.Start)
	if err := ps.Persistence.PersistBufferedResponse(context.Background(), response); err != nil {
		logrus.Errorf("Failed to buffer response, error : %s", err.Error())
	}

	ps.Responses.Store(bufferedRequest.Code, response)
}

func (ps *AsyncProxyingService) retryRequest(bufferedRequest *requests.BufferedRequest) {
	bufferedRequest.NumberTries++
	ps.RequestChannel <- bufferedRequest
}

func (ps *AsyncProxyingService) asyncRequestHandler() {
	for {
		bufferedRequest := <-ps.RequestChannel

		go func(bufferedRequest *requests.BufferedRequest) {
			logrus.Tracef("[reverse proxy server] firing a request")

			if bufferedRequest.NumberTries >= ps.AllowedRetries {
				ps.declareResponseAsFailed(bufferedRequest)
			} else {
				response := ps.executeRequest(bufferedRequest)

				switch response.StatusCode {
				case http.StatusOK, http.StatusBadRequest:
					ps.declareResponseAsSuccessful(bufferedRequest, response)
				default:
					ps.retryRequest(bufferedRequest)
				}
			}
		}(bufferedRequest)
	}
}

func (ps *AsyncProxyingService) executeRequest(bufferedRequest *requests.BufferedRequest) *requests.BufferedResponse {
	httpResponse := httptest.NewRecorder()

	bufferedResponse := proxyHandler(
		reverse_proxy.CreateReverseProxy(),
		httpResponse,
		&bufferedRequest.Request,
		requestMetadata{
			start:            bufferedRequest.Start,
			serialization:    bufferedRequest.SerializationDuration,
			persistenceLayer: bufferedRequest.PersistenceDuration,
		},
		ps.Context,
	)
	// copy response from the buffer
	bufferedResponse.Body = httpResponse.Body.String()

	return bufferedResponse
}

func (ps *AsyncProxyingService) createAsyncResponseHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		buf := new(bytes.Buffer)
		_, err := buf.ReadFrom(r.Body)
		if err != nil {
			logrus.Errorf("[reverse proxy server] failed to read response body, error : %s", err.Error())
		}

		responseKey := buf.String()
		logrus.Tracef("[reverse proxy server] received code request with code : %s", responseKey)

		if v, ok := ps.Responses.Load(responseKey); ok {
			val := v.(*requests.BufferedResponse)

			elapsed := time.Since(start) + val.E2ELatency
			w.Header().Set("Duration-Microseconds", strconv.FormatInt(elapsed.Microseconds(), 10))
			_ = requests.FillResponseWithBufferedResponse(w, val)
		} else {
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
	ps.Responses.Range(func(key, value any) bool {
		if time.Now().Sub(value.(*requests.BufferedResponse).Timestamp) > time.Minute {
			ps.Responses.Delete(key)
		}

		return true
	})
}

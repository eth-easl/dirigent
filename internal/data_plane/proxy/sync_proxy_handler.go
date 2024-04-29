package proxy

import (
	common "cluster_manager/internal/data_plane/function_metadata"
	"cluster_manager/internal/data_plane/proxy/load_balancing"
	"cluster_manager/internal/data_plane/proxy/metrics_collection"
	"cluster_manager/pkg/config"
	"cluster_manager/pkg/tracing"
	"cluster_manager/pkg/utils"
	"cluster_manager/proto"
	"github.com/sirupsen/logrus"
	"net/http"
	"time"
)

type ProxyingService struct {
	Host    string
	Port    string
	Context proxyContext
}

func (ps *ProxyingService) GetCpApiServer() proto.CpiInterfaceClient {
	return ps.Context.cpInterface
}

func (ps *ProxyingService) SetCpApiServer(client proto.CpiInterfaceClient) {
	ps.Context.cpInterface = client
}

func NewProxyingService(cfg config.DataPlaneConfig, cache *common.Deployments, cp proto.CpiInterfaceClient, outputFile string, loadBalancingPolicy load_balancing.LoadBalancingPolicy) *ProxyingService {
	incomingRequestChannel, doneRequestChannel := metrics_collection.NewMetricsCollector(cfg.ControlPlaneNotifyIntervalInMinutes, cp)

	return &ProxyingService{
		Host: utils.Localhost,
		Port: cfg.PortProxy,
		Context: proxyContext{
			cache:               cache,
			cpInterface:         cp,
			loadBalancingPolicy: loadBalancingPolicy,

			tracing:                tracing.NewProxyTracingService(outputFile),
			incomingRequestChannel: incomingRequestChannel,
			doneRequestChannel:     doneRequestChannel,
		},
	}
}

func (ps *ProxyingService) StartProxyServer() {
	startProxy(ps.createInvocationHandler(), ps.Context, ps.Host, ps.Port)
}

func (ps *ProxyingService) StartTracingService() {
	ps.Context.tracing.StartTracingService()
}

func (ps *ProxyingService) createInvocationHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, request *http.Request) {
		bufferedResponse := proxyHandler(request, requestMetadata{
			start:            time.Now(),
			serialization:    0,
			persistenceLayer: 0,
		}, ps.Context)

		w.WriteHeader(bufferedResponse.StatusCode)

		if bufferedResponse.StatusCode == http.StatusOK {
			w.Write([]byte(bufferedResponse.Body))
		} else {
			logrus.Warnf(bufferedResponse.Body)
		}
	}
}

package proxy

import (
	"cluster_manager/internal/data_plane/proxy/load_balancing"
	"cluster_manager/internal/data_plane/proxy/metrics_collection"
	"cluster_manager/internal/data_plane/proxy/reverse_proxy"
	"cluster_manager/internal/data_plane/service_metadata"
	"cluster_manager/pkg/config"
	"cluster_manager/pkg/tracing"
	"cluster_manager/pkg/utils"
	"cluster_manager/proto"
	log "github.com/sirupsen/logrus"
	"net/http"
	"time"
)

type SyncProxyingService struct {
	ProxyingService
}

func NewProxyingService(cfg config.DataPlaneConfig, cache *service_metadata.Deployments, cp proto.CpiInterfaceClient, outputFile string, loadBalancingPolicy load_balancing.LoadBalancingPolicy) *SyncProxyingService {
	incomingRequestChannel, doneRequestChannel := metrics_collection.NewMetricsCollector(cfg.ControlPlaneNotifyIntervalInMinutes, cp)

	return &SyncProxyingService{
		ProxyingService: ProxyingService{
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
			reverseProxy: reverse_proxy.CreateReverseProxy(),
			dpConfig:     &cfg,
		},
	}
}

func (sps *SyncProxyingService) GetCpApiServer() proto.CpiInterfaceClient {
	return sps.Context.cpInterface
}

func (sps *SyncProxyingService) SetCpApiServer(client proto.CpiInterfaceClient) {
	sps.Context.cpInterface = client
}

func (sps *SyncProxyingService) StartProxyServer() {
	startProxy(sps.createInvocationHandler(), sps.Context, sps.Host, sps.Port)
}

func (sps *SyncProxyingService) StartTracingService() {
	sps.Context.tracing.StartTracingService()
}

func (sps *SyncProxyingService) createInvocationHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, request *http.Request) {
		resp := proxyHandler(sps.reverseProxy, w, request, requestMetadata{
			start:            time.Now(),
			serialization:    0,
			persistenceLayer: 0,
		}, sps.Context, sps.dpConfig)
		if resp.StatusCode != http.StatusOK {
			w.WriteHeader(resp.StatusCode)
			_, err := w.Write([]byte(resp.Body))
			if err != nil {
				log.Errorf("Failed to write failure message in response: %v", err)
			}
		}
	}
}

package proxy

import (
	"cluster_manager/pkg/config"
	"cluster_manager/proto"
	"net/http/httputil"
)

type Proxy interface {
	StartProxyServer()
	StartProxyTracingService()
	StartTaskTracingService()
	StartWorkflowTracingService()

	GetCpApiServer() proto.CpiInterfaceClient
	SetCpApiServer(client proto.CpiInterfaceClient)
}

type ProxyingService struct {
	Host    string
	Port    string
	Context proxyContext

	reverseProxy *httputil.ReverseProxy
	dpConfig     *config.DataPlaneConfig
}

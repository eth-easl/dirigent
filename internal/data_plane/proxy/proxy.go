package proxy

import (
	"cluster_manager/proto"
	"net/http/httputil"
)

type Proxy interface {
	StartProxyServer()
	StartTracingService()

	GetCpApiServer() proto.CpiInterfaceClient
	SetCpApiServer(client proto.CpiInterfaceClient)
}

type ProxyingService struct {
	Host    string
	Port    string
	Context proxyContext

	reverseProxy *httputil.ReverseProxy
}

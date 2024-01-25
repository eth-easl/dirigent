package proxy

type Proxy interface {
	StartProxyServer()
	StartTracingService()
}

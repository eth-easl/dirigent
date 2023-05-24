package proxy

import (
	"cluster_manager/common"
	net2 "cluster_manager/proxy/net"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"net"
	"net/http"
)

func CreateProxyServer(host string, port string, cache *common.Deployments) {
	proxy := net2.NewProxy()

	// function composition <=> [cold start handler -> proxy handler -> forwarded shim handler -> proxy]
	var composedHandler http.Handler = proxy
	composedHandler = ForwardedShimHandler(composedHandler)
	composedHandler = LoadBalancingHandler(composedHandler, cache)
	composedHandler = ColdStartHandler(composedHandler, cache)

	proxyRxAddress := net.JoinHostPort(host, port)
	logrus.Info("Creating a proxy server at ", proxyRxAddress)

	server := &http.Server{
		Addr:    proxyRxAddress,
		Handler: h2c.NewHandler(composedHandler, &http2.Server{}),
	}

	err := server.ListenAndServe()
	if err != nil {
		logrus.Fatal("Failed to create a proxy server.")
	}
}

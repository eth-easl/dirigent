package proxy

import (
	"cluster_manager/api/proto"
	"cluster_manager/common"
	net2 "cluster_manager/proxy/net"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"net"
	"net/http"
)

func CreateProxyServer(host string, port string, cache *common.Deployments, cp *proto.CpiInterfaceClient) {
	proxy := net2.NewProxy()

	// function composition <=> [cold start handler -> forwarded shim handler -> proxy]
	var composedHandler http.Handler = proxy
	//composedHandler = ForwardedShimHandler(composedHandler)
	composedHandler = InvocationHandler(composedHandler, cache, cp)

	proxyRxAddress := net.JoinHostPort(host, port)
	logrus.Info("Creating a proxy server at ", proxyRxAddress)

	server := &http.Server{
		Addr:    proxyRxAddress,
		Handler: h2c.NewHandler(composedHandler, &http2.Server{}),
		// TODO: add timeout for each request on the gateway side
	}

	err := server.ListenAndServe()
	if err != nil {
		logrus.Fatal("Failed to create a proxy server.")
	}
}

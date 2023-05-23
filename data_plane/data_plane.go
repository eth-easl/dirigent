package main

import (
	"cluster_manager/api"
	"cluster_manager/common"
	proxy2 "cluster_manager/proxy"
	net2 "cluster_manager/proxy/net"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"net"
	"net/http"
)

const (
	Host      = "localhost"
	ProxyPort = "8080"
	ApiPort   = "8081"
)

func prepopulate(cache *common.Deployments) {
	logrus.Debug("Cache prepopulation")
	cache.AddDeployment("/faas.Executor/Execute")
}

func main() {
	logrus.SetLevel(logrus.DebugLevel)

	stop := make(chan struct{})

	proxy := net2.NewProxy()
	deploymentCache := common.NewDeploymentList()

	prepopulate(deploymentCache)

	// function composition <=> [cold start handler -> proxy handler -> forwarded shim handler -> proxy]
	var composedHandler http.Handler = proxy
	composedHandler = proxy2.ForwardedShimHandler(composedHandler)
	composedHandler = proxy2.LoadBalancingHandler(composedHandler)
	composedHandler = proxy2.ColdStartHandler(composedHandler, deploymentCache)

	proxyRxAddress := net.JoinHostPort(Host, ProxyPort)
	logrus.Info("Creating a proxy server at ", proxyRxAddress)

	server := &http.Server{
		Addr:    proxyRxAddress,
		Handler: h2c.NewHandler(composedHandler, &http2.Server{}),
	}

	setupAPIServer(stop, deploymentCache)

	err := server.ListenAndServe()
	if err != nil {
		logrus.Fatal("Failed to create a proxy server.")
	}
}

func setupAPIServer(stop chan struct{}, cache *common.Deployments) {
	go func() {
		dpAPIAddress := net.JoinHostPort(Host, ApiPort)

		api.CreateDataPlaneAPIServer(dpAPIAddress, cache)
		<-stop
	}()
}

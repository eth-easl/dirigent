package main

import (
	"cluster_manager/api"
	"cluster_manager/common"
	"cluster_manager/proxy"
	"github.com/sirupsen/logrus"
)

const (
	Host      = "localhost"
	ProxyPort = "8080"
	ApiPort   = "8081"
)

func prepopulate(cache *common.Deployments) {
	logrus.Debug("Cache prepopulation")

	name := "/faas.Executor/Execute"
	cache.AddDeployment(name)
}

func main() {
	logrus.SetLevel(logrus.DebugLevel)

	cache := common.NewDeploymentList()

	prepopulate(cache)
	go api.CreateDataPlaneAPIServer(Host, ApiPort, cache)

	proxy.CreateProxyServer(Host, ProxyPort, cache)
}

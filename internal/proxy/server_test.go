package proxy

import (
	common "cluster_manager/internal/common"
	testserver "cluster_manager/tests"
	"cluster_manager/tests/proto"
	"fmt"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"
	"io"
	"net/http"
	"testing"
	"time"
)

// uses ports 9000 and 9001
func TestE2E_HTTP_H2C_NoColdStart(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)

	//host, dpPort := "localhost", "9000"
	endpointPort := "9001"

	cache := common.NewDeploymentList()
	//go CreateProxyServer(host, dpPort, cache)
	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/test", func(writer http.ResponseWriter, request *http.Request) {
			_, _ = io.WriteString(writer, fmt.Sprint("Hello"))
		})

		srv := &http.Server{
			Addr:    ":" + endpointPort,
			Handler: h2c.NewHandler(mux, &http2.Server{}),
		}

		err := srv.ListenAndServe()
		if err != nil {
			t.Error("Failed to create a http2 server.")
		}
	}()

	// http and proxy server setup may take some time
	time.Sleep(5 * time.Second)

	if !cache.AddDeployment("/test") {
		t.Error("Failed to add deployment to cache.")
	}
	fx := cache.GetDeployment("/test")
	fx.SetUpstreamURLs([]string{"localhost:" + endpointPort})

	client := &http.Client{}
	req, _ := http.NewRequest("GET", "http://localhost:9000/test", nil)
	res, err := client.Do(req)

	if err != nil || res.StatusCode != http.StatusOK {
		t.Error("Failed to proxy HTTP request.")
	}
}

// uses ports 9002 and 9003
func TestE2E_gRPC_H2C_NoColdStart(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)

	host := "localhost"
	proxyPort := "9002"
	sandboxPort := "9003"

	cache := common.NewDeploymentList()

	if !cache.AddDeployment("/faas.Executor/Execute") {
		t.Error("Failed to add deployment to cache.")
	}
	fx := cache.GetDeployment("/faas.Executor/Execute")
	fx.SetUpstreamURLs([]string{"localhost:" + fmt.Sprintf("%s", sandboxPort)})

	// proxy
	//go CreateProxyServer(host, proxyPort, cache)
	// endpoint
	go common.CreateGRPCServer(host, sandboxPort, func(sr grpc.ServiceRegistrar) {
		proto.RegisterExecutorServer(sr, &testserver.TestServer{})
	})

	time.Sleep(5 * time.Second)

	// invocation
	err := testserver.FireInvocation(t, host, proxyPort)
	if err != nil {
		t.Error(fmt.Sprintf("Invocation failed - %s", err))
	}
}

// uses ports 9004, 9005, 9006
func TestE2E_ColdStart_WithResolution(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)

	host := "localhost"
	proxyPort := "9004"
	sandboxPort := "9005"
	apiServerPort := "9006"

	cache := common.NewDeploymentList()
	if !cache.AddDeployment("/faas.Executor/Execute") {
		t.Error("Failed to add deployment to cache.")
	}

	// api server
	//go api.CreateDataPlaneAPIServer(host, apiServerPort, cache)
	// proxy
	//go CreateProxyServer(host, proxyPort, cache)
	// endpoint
	go common.CreateGRPCServer(host, sandboxPort, func(sr grpc.ServiceRegistrar) {
		proto.RegisterExecutorServer(sr, &testserver.TestServer{})
	})

	time.Sleep(5 * time.Second)

	// invocation to experience cold start
	result := make(chan interface{})
	go func() {
		err := testserver.FireInvocation(t, host, proxyPort)
		result <- err
	}()

	time.Sleep(3 * time.Second)

	testserver.UpdateEndpointList(t, host, apiServerPort, []string{host + ":" + sandboxPort})

	time.Sleep(2 * time.Second)

	msg := <-result
	if msg != nil {
		t.Error("Failed to take the request of the cold start buffer.")
	}
}

// uses ports 9007 and 9008
func TestE2E_gRPC_H2C_NoDeployment(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)

	host := "localhost"
	proxyPort := "9007"
	sandboxPort := "9008"

	//cache := common.NewDeploymentList()

	// proxy
	//go CreateProxyServer(host, proxyPort, cache)
	// endpoint
	go common.CreateGRPCServer(host, sandboxPort, func(sr grpc.ServiceRegistrar) {
		proto.RegisterExecutorServer(sr, &testserver.TestServer{})
	})

	time.Sleep(5 * time.Second)

	// invocation
	err := testserver.FireInvocation(t, host, proxyPort)
	if err == nil {
		t.Error(fmt.Sprintf("Invocation failed - %s", err))
	}
}

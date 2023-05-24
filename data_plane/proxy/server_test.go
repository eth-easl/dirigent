package proxy

import (
	"cluster_manager/common"
	testserver "cluster_manager/tests"
	"cluster_manager/tests/proto"
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"io"
	"net"
	"net/http"
	"testing"
	"time"
)

// uses ports 9000 and 9001
func TestE2E_HTTP_H2C_NoColdStart(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)

	host, dpPort := "localhost", "9000"
	endpointPort := "9001"

	cache := common.NewDeploymentList()
	go CreateProxyServer(host, dpPort, cache)
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

	err := cache.AddDeployment("/test")
	if err != nil {
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
	dpPort := "9002"
	endpointPort := 9003

	cache := common.NewDeploymentList()

	err := cache.AddDeployment("/faas.Executor/Execute")
	if err != nil {
		t.Error("Failed to add deployment to cache.")
	}
	fx := cache.GetDeployment("/faas.Executor/Execute")
	fx.SetUpstreamURLs([]string{"localhost:" + fmt.Sprintf("%d", endpointPort)})

	go CreateProxyServer(host, dpPort, cache)
	go testserver.StartGRPCServer(host, endpointPort)

	time.Sleep(5 * time.Second)

	conn, err := testserver.EstablishConnection(net.JoinHostPort(host, dpPort))
	defer testserver.GRPCConnectionClose(conn)
	if err != nil {
		t.Errorf("gRPC connection timeout - %v\n", err)
	}

	executionCxt, cancelExecution := context.WithTimeout(context.Background(), testserver.GRPCFunctionTimeout)
	defer cancelExecution()

	_, err = proto.NewExecutorClient(conn).Execute(executionCxt, &proto.FaasRequest{
		Message:           "nothing",
		RuntimeInMilliSec: uint32(1000),
		MemoryInMebiBytes: uint32(2048),
	})
	if err != nil {
		t.Error(fmt.Sprintf("Function timeout - %s", err))
	}
}

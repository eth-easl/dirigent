package proxy

import (
	proto2 "cluster_manager/api/proto"
	common "cluster_manager/internal/data_plane/function_metadata"
	"cluster_manager/pkg/grpc_helpers"
	"cluster_manager/pkg/utils"
	"cluster_manager/tools/proto"
	testserver "cluster_manager/tools/utils"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"
)

const localhost string = "localhost"

// uses ports 9000 and 9001.
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
		assert.NoError(t, err, "Failed to create a http2 server.")
	}()

	// http and proxy server setup may take some time
	time.Sleep(5 * time.Second)
	assert.True(t, cache.AddDeployment("/test"), "Failed to add deployment to cache.")

	fx, _ := cache.GetDeployment("/test")
	err := fx.SetUpstreamURLs([]*proto2.EndpointInfo{{
		ID:  "mockId",
		URL: "localhost:" + endpointPort},
	})

	assert.NoError(t, err, "Failed to set upstream urls")

	client := &http.Client{}
	req, _ := http.NewRequest(http.MethodGet, "http://localhost:9000/test", nil)
	res, err := client.Do(req)

	assert.True(t, err == nil && res.StatusCode == http.StatusOK, "Failed to proxy HTTP request.")
}

// uses ports 9002 and 9003.
func TestE2E_gRPC_H2C_NoColdStart(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)

	host := utils.Localhost
	proxyPort := "9002"
	sandboxPort := "9003"

	cache := common.NewDeploymentList()

	if !cache.AddDeployment("/faas.Executor/Execute") {
		t.Error("Failed to add deployment to cache.")
	}

	fx, _ := cache.GetDeployment("/faas.Executor/Execute")
	err := fx.SetUpstreamURLs([]*proto2.EndpointInfo{{
		ID:  "mockId",
		URL: utils.Localhost + ":" + fmt.Sprintf("%s", sandboxPort)},
	})

	assert.NoError(t, err, "Failed to set upstream URLs")

	// proxy
	//go CreateProxyServer(host, proxyPort, cache)
	// endpoint
	go grpc_helpers.CreateGRPCServer(host, sandboxPort, func(sr grpc.ServiceRegistrar) {
		proto.RegisterExecutorServer(sr, &testserver.TestServer{})
	})

	time.Sleep(5 * time.Second)

	conn := grpc_helpers.EstablishGRPCConnectionPoll(host, proxyPort)
	assert.NotNil(t, conn, "Failed to establish gRPC connection with the data plane")
	executorClient := proto.NewExecutorClient(conn)

	// invocation
	err = testserver.FireInvocation(executorClient)
	assert.NoErrorf(t, err, "Invocation failed - %s", err)
}

// uses ports 9004, 9005, 9006.
func TestE2E_ColdStart_WithResolution(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)

	host := "localhost"
	proxyPort := "9004"
	sandboxPort := "9005"
	apiServerPort := "9006"

	cache := common.NewDeploymentList()
	assert.True(t, cache.AddDeployment("/faas.Executor/Execute"), "Failed to add deployment to cache.")
	// api server
	//go api.CreateDataPlaneAPIServer(host, apiServerPort, cache)
	// proxy
	//go CreateProxyServer(host, proxyPort, cache)
	// endpoint
	go grpc_helpers.CreateGRPCServer(host, sandboxPort, func(sr grpc.ServiceRegistrar) {
		proto.RegisterExecutorServer(sr, &testserver.TestServer{})
	})

	time.Sleep(5 * time.Second)

	conn := grpc_helpers.EstablishGRPCConnectionPoll(host, proxyPort)
	assert.NotNil(t, conn, "Failed to establish gRPC connection with the data plane")
	executorClient := proto.NewExecutorClient(conn)

	// invocation to experience cold start
	result := make(chan interface{})
	go func() {
		err := testserver.FireInvocation(executorClient)
		result <- err
	}()

	time.Sleep(3 * time.Second)

	testserver.UpdateEndpointList(t, host, apiServerPort, []*proto2.EndpointInfo{{
		ID:  "id",
		URL: host + ":" + sandboxPort,
	}})

	time.Sleep(2 * time.Second)

	msg := <-result
	assert.NotNil(t, msg, "Failed to take the request of the cold start buffer.")
}

// uses ports 9007 and 9008.
func TestE2E_gRPC_H2C_NoDeployment(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)

	host := "localhost"
	proxyPort := "9007"
	sandboxPort := "9008"

	//cache := common.NewDeploymentList()

	// proxy
	//go CreateProxyServer(host, proxyPort, cache)
	// endpoint
	go grpc_helpers.CreateGRPCServer(host, sandboxPort, func(sr grpc.ServiceRegistrar) {
		proto.RegisterExecutorServer(sr, &testserver.TestServer{})
	})

	time.Sleep(5 * time.Second)

	conn := grpc_helpers.EstablishGRPCConnectionPoll(host, proxyPort)
	assert.NotNil(t, conn, "Failed to establish gRPC connection with the data plane")
	executorClient := proto.NewExecutorClient(conn)

	// invocation
	err := testserver.FireInvocation(executorClient)
	assert.NoErrorf(t, err, "Invocation failed - %s", err)
}

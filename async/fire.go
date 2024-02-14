package main

import (
	proto2 "cluster_manager/api/proto"
	"cluster_manager/internal/control_plane/autoscaling"
	common "cluster_manager/pkg/grpc_helpers"
	"cluster_manager/pkg/utils"
	"context"
	"crypto/tls"
	"fmt"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/http2"
	"io"
	"net"
	"net/http"
	"strconv"
	"time"
)

func Deployservice() {
	cpApi, err := common.InitializeControlPlaneConnection(
		[]string{"localhost", utils.DefaultControlPlanePort},
		"",
		-1,
		-1,
	)
	if err != nil {
		logrus.Fatalf("Failed to start control plane connection (error %s)", err.Error())
	}

	ctx, cancel := context.WithTimeout(context.Background(), utils.GRPCFunctionTimeout)
	defer cancel()

	autoscalingConfig := autoscaling.NewDefaultAutoscalingMetadata()
	autoscalingConfig.ScalingUpperBound = 1

	resp, err := cpApi.RegisterService(ctx, &proto2.ServiceInfo{
		Name:  "test",
		Image: "docker.io/cvetkovic/dirigent_trace_function:latest",
		PortForwarding: &proto2.PortMapping{
			GuestPort: 80,
			Protocol:  proto2.L4Protocol_TCP,
		},
		AutoscalingConfig: autoscalingConfig,
	})

	if err != nil || !resp.Success {
		logrus.Errorf("Failed to deploy service")
	}
}

func Fire() {

	logrus.Info("Invocating async function")

	////////////////////////////////////
	// INVOKE FUNCTION
	////////////////////////////////////

	client := http.Client{
		Timeout: 5 * time.Second,
		Transport: &http2.Transport{
			AllowHTTP: true,
			DialTLSContext: func(ctx context.Context, network, addr string, cfg *tls.Config) (net.Conn, error) {
				return net.Dial(network, addr)
			},
			DisableCompression: true,
		},
	}

	req, err := http.NewRequest("GET", "http://localhost:8080", nil)
	if err != nil {
		panic(err.Error())
	}

	req.Host = "test"

	req.Header.Set("workload", "trace")
	req.Header.Set("requested_cpu", strconv.Itoa(1))
	req.Header.Set("requested_memory", strconv.Itoa(1))
	req.Header.Set("multiplier", strconv.Itoa(80))

	resp, err := client.Do(req)
	if err != nil {
		logrus.Fatalf("Failed to establish a HTTP connection - %s\n", err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil || resp == nil || resp.StatusCode != http.StatusOK {
		msg := ""
		if err != nil {
			msg = err.Error()
		} else if resp == nil {
			msg = fmt.Sprintf("empty response - status code: %d", resp.StatusCode)
		} else if resp.StatusCode != http.StatusOK {
			msg = fmt.Sprintf("%s - status code: %d", body, resp.StatusCode)
		}
		logrus.Debugf("HTTP timeout for function")
		logrus.Fatalf(msg)
	}

	fmt.Println(string(body[:]))

	/*code := string(body[:])

	time.Sleep(5 * time.Second)

	req, err = http.NewRequest("GET", "http://localhost:8082", strings.NewReader(code))
	if err != nil {
		logrus.Fatalf(err.Error())
	}

	resp, err = client.Do(req)
	if err != nil {
		logrus.Fatalf(err.Error())
	}

	responseInBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		logrus.Fatalf(err.Error())
	}

	logrus.Info("Final result")
	logrus.Infof(string(responseInBytes[:]))*/
}

func main() {
	Deployservice()

	time.Sleep(200 * time.Millisecond)

	for i := 0; i < 100; i++ {
		Fire()
		time.Sleep(1000 * time.Millisecond)
	}

}

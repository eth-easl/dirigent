package shared

import (
	proto2 "cluster_manager/api/proto"
	"cluster_manager/internal/control_plane/autoscaling"
	common "cluster_manager/pkg/grpc_helpers"
	"cluster_manager/pkg/utils"
	"context"
	"crypto/tls"
	"fmt"
	"golang.org/x/net/http2"
	"io"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	DeployedFunctionName  string = "test"
	ControlPlaneAddress   string = "localhost"
	DataPlaneAddress      string = "localhost"
	FunctionNameFormatter string = "%s%d"
)

func DeployService(nbDeploys, offset int) {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.TextFormatter{TimestampFormat: time.StampMilli, FullTimestamp: true})

	cpApi, err := common.InitializeControlPlaneConnection(ControlPlaneAddress, utils.DefaultControlPlanePort, "", -1, -1)
	if err != nil {
		logrus.Fatalf("Failed to start control plane connection (error %s)", err.Error())
	}

	ctx, cancel := context.WithTimeout(context.Background(), utils.GRPCFunctionTimeout)
	defer cancel()

	autoscalingConfig := autoscaling.NewDefaultAutoscalingMetadata()
	autoscalingConfig.ScalingUpperBound = 1
	//autoscalingConfig.ScalingLowerBound = 1

	for i := 0; i < nbDeploys; i++ {
		resp, err := cpApi.RegisterService(ctx, &proto2.ServiceInfo{
			Name:  fmt.Sprintf(FunctionNameFormatter, DeployedFunctionName, i+offset),
			Image: "docker.io/cvetkovic/empty_function:latest",
			PortForwarding: &proto2.PortMapping{
				GuestPort: 80,
				Protocol:  proto2.L4Protocol_TCP,
			},
			AutoscalingConfig: autoscalingConfig,
		})

		if err != nil || !resp.Success {
			logrus.Error("Failed to deploy service")
		}
	}
}

func DeployServiceMultiThread(nbDeploys, offset int) {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.TextFormatter{TimestampFormat: time.StampMilli, FullTimestamp: true})

	cpApi, err := common.InitializeControlPlaneConnection(ControlPlaneAddress, utils.DefaultControlPlanePort, "", -1, -1)
	if err != nil {
		logrus.Fatalf("Failed to start control plane connection (error %s)", err.Error())
	}

	ctx, cancel := context.WithTimeout(context.Background(), utils.GRPCFunctionTimeout)
	defer cancel()

	autoscalingConfig := autoscaling.NewDefaultAutoscalingMetadata()
	autoscalingConfig.ScalingUpperBound = 1
	//autoscalingConfig.ScalingLowerBound = 1

	wg := sync.WaitGroup{}
	wg.Add(nbDeploys)

	for i := 0; i < nbDeploys; i++ {
		go func(i int, offset int) {
			resp, err := cpApi.RegisterService(ctx, &proto2.ServiceInfo{
				Name:  fmt.Sprintf("%s&%d", DeployedFunctionName, i+offset),
				Image: "docker.io/cvetkovic/empty_function:latest",
				PortForwarding: &proto2.PortMapping{
					GuestPort: 80,
					Protocol:  proto2.L4Protocol_TCP,
				},
				AutoscalingConfig: autoscalingConfig,
			})

			if err != nil || !resp.Success {
				logrus.Error("Failed to deploy service")
			}

			wg.Done()
		}(i, offset)
	}

	wg.Wait()
}

func FireColdstartMultiThread(nbDeploys, offset int, requests int32) {

	cpApi, err := common.InitializeControlPlaneConnection(ControlPlaneAddress, utils.DefaultControlPlanePort, "", -1, -1)
	if err != nil {
		logrus.Fatalf("Failed to start control plane connection (error %s)", err.Error())
	}

	ctx, cancel := context.WithTimeout(context.Background(), utils.GRPCFunctionTimeout)
	defer cancel()

	wg := sync.WaitGroup{}
	wg.Add(nbDeploys)

	for i := 0; i < nbDeploys; i++ {
		go func(i int, offset int) {
			resp, err := cpApi.OnMetricsReceive(ctx, &proto2.AutoscalingMetric{
				ServiceName:      fmt.Sprintf("%s&%d", DeployedFunctionName, i+offset),
				DataplaneName:    "mockDataplane",
				InflightRequests: requests,
			})

			if err != nil || !resp.Success {
				text := ""
				if err != nil {
					text += err.Error()
				}
				logrus.Errorf("Failed to simulate cold start service : (error %s)", text)
			}

			wg.Done()
		}(i, offset)
	}

	wg.Wait()
}

func Deregister(nbDeploys, offset int) {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.TextFormatter{TimestampFormat: time.StampMilli, FullTimestamp: true})

	cpApi, err := common.InitializeControlPlaneConnection(ControlPlaneAddress, utils.DefaultControlPlanePort, "", -1, -1)
	if err != nil {
		logrus.Fatalf("Failed to start control plane connection (error %s)", err.Error())
	}

	ctx, cancel := context.WithTimeout(context.Background(), utils.GRPCFunctionTimeout)
	defer cancel()

	autoscalingConfig := autoscaling.NewDefaultAutoscalingMetadata()
	autoscalingConfig.ScalingUpperBound = 1
	//autoscalingConfig.ScalingLowerBound = 1

	wg := sync.WaitGroup{}
	wg.Add(nbDeploys)

	for i := 0; i < nbDeploys; i++ {
		go func(i int, offset int) {
			resp, err := cpApi.DeregisterService(ctx, &proto2.ServiceInfo{
				Name:  fmt.Sprintf("%s&%d", DeployedFunctionName, i+offset),
				Image: "docker.io/cvetkovic/empty_function:latest",
				PortForwarding: &proto2.PortMapping{
					GuestPort: 80,
					Protocol:  proto2.L4Protocol_TCP,
				},
				AutoscalingConfig: autoscalingConfig,
			})

			if err != nil || !resp.Success {
				text := ""
				if err != nil {
					text += err.Error()
				}
				logrus.Errorf("Failed to deploy service : %s", text)
			}

			wg.Done()
		}(i, offset)
	}

	wg.Wait()
}

func DeployServiceTime(nbDeploys, offset int) time.Duration {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.TextFormatter{TimestampFormat: time.StampMilli, FullTimestamp: true})

	cpApi, err := common.InitializeControlPlaneConnection(ControlPlaneAddress, utils.DefaultControlPlanePort, "", -1, -1)
	if err != nil {
		logrus.Fatalf("Failed to start control plane connection (error %s)", err.Error())
	}

	ctx, cancel := context.WithTimeout(context.Background(), utils.GRPCFunctionTimeout)
	defer cancel()

	autoscalingConfig := autoscaling.NewDefaultAutoscalingMetadata()
	autoscalingConfig.ScalingUpperBound = 1
	//autoscalingConfig.ScalingLowerBound = 1

	start := time.Now()
	for i := 0; i < nbDeploys; i++ {
		resp, err := cpApi.RegisterService(ctx, &proto2.ServiceInfo{
			Name:  fmt.Sprintf(FunctionNameFormatter, DeployedFunctionName, i+offset),
			Image: "docker.io/cvetkovic/empty_function:latest",
			PortForwarding: &proto2.PortMapping{
				GuestPort: 80,
				Protocol:  proto2.L4Protocol_TCP,
			},
			AutoscalingConfig: autoscalingConfig,
		})

		if err != nil || !resp.Success {
			text := ""
			if err != nil {
				text += err.Error()
			}
			logrus.Errorf("Failed to deploy service : %s", text)
		}
	}

	return time.Since(start)
}

func PerformXInvocations(nbInvocations, offset int) {
	wg := sync.WaitGroup{}
	wg.Add(nbInvocations)

	for i := 0; i < nbInvocations; i++ {
		functionName := fmt.Sprintf(FunctionNameFormatter, DeployedFunctionName, i+offset)

		go func(i int) {
			defer wg.Done()

			start := time.Now()

			client := http.Client{
				Timeout: 10 * time.Second,
				Transport: &http2.Transport{
					AllowHTTP: true,
					DialTLSContext: func(ctx context.Context, network, addr string, cfg *tls.Config) (net.Conn, error) {
						return net.Dial(network, addr)
					},
					DisableCompression: true,
				},
			}

			req, err := http.NewRequest("GET", "http://"+DataPlaneAddress+":"+"8080", nil)
			req.Host = functionName

			resp, err := client.Do(req)
			if err != nil {
				logrus.Debugf("Failed to establish a HTTP connection - %v\n", err.Error())
				return
			}

			body, err := io.ReadAll(resp.Body)
			if err != nil || resp == nil || resp.StatusCode != http.StatusOK {
				logrus.Debugf("HTTP timeout for function %s", functionName)
				return
			}

			logrus.Info(fmt.Sprintf("%s - %d ms", string(body), time.Since(start).Milliseconds()))
		}(i)
	}

	wg.Wait()
}

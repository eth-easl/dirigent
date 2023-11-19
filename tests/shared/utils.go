package shared

import (
	proto2 "cluster_manager/api/proto"
	"cluster_manager/internal/control_plane/autoscaling"
	common "cluster_manager/pkg/grpc_helpers"
	"cluster_manager/pkg/utils"
	"cluster_manager/tests/proto"
	"context"
	"crypto/tls"
	"fmt"
	"golang.org/x/net/http2"
	"google.golang.org/grpc"
	"io"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	DeployedFunctionName  string = "test"
	ControlPlaneAddress   string = "10.0.1.2"
	DataPlaneAddress      string = "10.0.1.3"
	FunctionNameFormatter string = "%s%d"
)

func DeployServiceMultiThread(nbDeploys, offset int) {
	cpApi, err := common.InitializeControlPlaneConnection(ControlPlaneAddress, utils.DefaultControlPlanePort, "", -1, -1)
	if err != nil {
		logrus.Fatalf("Failed to start control plane connection (error %s)", err.Error())
	}

	ctx, cancel := context.WithTimeout(context.Background(), utils.GRPCFunctionTimeout)
	defer cancel()

	autoscalingConfig := autoscaling.NewDefaultAutoscalingMetadata()
	autoscalingConfig.ScalingUpperBound = 1

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

func DeregisterMultiThread(nbDeploys, offset int) {
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

func PerformXInvocationsContainerd(nbInvocations, offset int) {
	wg := sync.WaitGroup{}
	wg.Add(nbInvocations)

	for i := 0; i < nbInvocations; i++ {
		go func(i int) {
			conn := common.EstablishGRPCConnectionPoll(DataPlaneAddress, "8080", grpc.WithAuthority(fmt.Sprintf("%s&%d", DeployedFunctionName, i+offset)))
			if conn == nil {
				logrus.Fatal("Failed to establish gRPC connection with the data plane")
			}

			logrus.Info("Connection with the gRPC server has been established")

			executorClient := proto.NewExecutorClient(conn)
			err := FireInvocation(executorClient)

			if err != nil {
				logrus.Errorf("Invocation failed - %s", err)
			}

			wg.Done()
		}(i)
	}

	wg.Wait()
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
			defer handleBodyClosing(resp.Body)

			if err != nil || resp == nil || resp.StatusCode != http.StatusOK {
				logrus.Debugf("HTTP timeout for function %s", functionName)
				return
			}

			logrus.Info(fmt.Sprintf("%s - %d ms", string(body), time.Since(start).Milliseconds()))
		}(i)
	}

	wg.Wait()
}

func handleBodyClosing(Body io.ReadCloser) {
	err := Body.Close()
	if err != nil {
		logrus.Errorf("Error closing response body - %v", err)
	}
}

func DeployDataplanes(nbDeploys, offset int) {
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
			id := fmt.Sprintf("%s %d %d", "mockIp", i, offset)
			resp, err := cpApi.RegisterDataplane(ctx, &proto2.DataplaneInfo{
				IP:        id,
				APIPort:   0,
				ProxyPort: 0,
			})

			if err != nil || !resp.Success {
				errText := ""
				if err != nil {
					errText = err.Error()
				}
				logrus.Errorf("Failed to deploy dataplane : (error %s)", errText)
			}

			wg.Done()
		}(i, offset)
	}

	wg.Wait()
}

func DeployWorkers(nbDeploys, offset int) {
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
			id := fmt.Sprintf("%s %d %d", "mockIP", i, offset)
			resp, err := cpApi.RegisterNode(ctx, &proto2.NodeInfo{
				NodeID:     id, // Unique id while registering
				IP:         id,
				Port:       0,
				CpuCores:   0,
				MemorySize: 0,
			})

			if err != nil || !resp.Success {
				errText := "response is not successful"
				if err != nil {
					errText = err.Error()
				}
				logrus.Errorf("Failed to deploy worker : (error %s)", errText)
			}

			wg.Done()
		}(i, offset)
	}

	wg.Wait()
}

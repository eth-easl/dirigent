package dandelion

import (
	"bytes"
	"cluster_manager/internal/worker_node/managers"
	"cluster_manager/pkg/config"
	"cluster_manager/proto"
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/emptypb"
	"net/http"
	"os"
	"sync"
	"time"
)

type registeredFunctions struct {
	data map[string]bool
	sync.RWMutex
}

type Runtime struct {
	cpApi               proto.CpiInterfaceClient
	SandboxManager      *managers.SandboxManager
	registeredFunctions *registeredFunctions
	dandelionConfig     *config.DandelionConfig
	httpClient          *http.Client
}

func NewDandelionRuntime(cpApi proto.CpiInterfaceClient, sandboxManager *managers.SandboxManager, dandelionConfig *config.DandelionConfig) *Runtime {
	return &Runtime{
		cpApi:          cpApi,
		SandboxManager: sandboxManager,
		registeredFunctions: &registeredFunctions{
			data: make(map[string]bool),
		},
		dandelionConfig: dandelionConfig,
		httpClient: &http.Client{
			Timeout: 2500 * time.Millisecond,
			Transport: &http.Transport{
				IdleConnTimeout:     1 * time.Second,
				MaxIdleConns:        5,
				MaxIdleConnsPerHost: 5,
			},
		},
	}
}

// Patch for Rust and Golang bson libraries' different behavior
// rust: 1 byte -> 1 int32; golang: 1 byte -> 1 byte
func bytesToInts(binaryData []byte) []int32 {
	intData := make([]int32, len(binaryData))
	for i := 0; i < len(binaryData); i++ {
		intData[i] = int32(binaryData[i])
	}
	return intData
}

func getFailureStatus() *proto.SandboxCreationStatus {
	return &proto.SandboxCreationStatus{
		Success:          false,
		ID:               "-1",
		LatencyBreakdown: &proto.SandboxCreationBreakdown{},
	}
}

func (dr *Runtime) CreateSandbox(_ context.Context, in *proto.ServiceInfo) (*proto.SandboxCreationStatus, error) {
	start := time.Now()
	logrus.Debug("Create sandbox for service = '", in.Name, "'")

	dr.registeredFunctions.RLock()
	_, ok := dr.registeredFunctions.data[in.Name]
	dr.registeredFunctions.RUnlock()

	if !ok {
		// send register request to dandelion daemon
		binaryData, err := os.ReadFile(dr.dandelionConfig.BinaryPath)
		logrus.Infof("Using binary file %s (len: %d)", dr.dandelionConfig.BinaryPath, len(binaryData))
		if err != nil {
			logrus.Errorf("Error reading binary file - %v", err)
			return getFailureStatus(), nil
		}

		registerRequest := bson.D{
			{Key: "name", Value: in.Name},
			{Key: "context_size", Value: 0x8020000},
			{Key: "engine_type", Value: dr.dandelionConfig.EngineType},
			{Key: "binary", Value: bytesToInts(binaryData)},
			{Key: "input_sets", Value: bson.A{bson.A{"input", nil}, bson.A{"input", nil}}}, // names do not matter so far
			{Key: "output_sets", Value: []string{"output"}},                                // add "stdio" for debugging purpose
		}

		registerRequestBody, err := bson.Marshal(registerRequest)
		if err != nil {
			logrus.Errorf("Error marshalling function binary to BSON - %v", err)
			return getFailureStatus(), nil
		}

		registrationURL := fmt.Sprintf("http://localhost:%d/register/function", dr.dandelionConfig.DaemonPort)
		req, err := http.NewRequest("POST", registrationURL, bytes.NewBuffer(registerRequestBody))
		if err != nil {
			logrus.Errorf("Error creating Dandelion function registration request - %v", err)
			return getFailureStatus(), nil
		}

		resp, err := dr.httpClient.Do(req)
		if err != nil {
			logrus.Debugf("Failed to register function with Dandelion - %v", err)
			return getFailureStatus(), nil
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			logrus.Debugf("Successfully registered function %s", in.Name)
		} else {
			logrus.Debugf("Failed to register function %s with Dandelion (status code: %d)", in.Name, resp.StatusCode)
			return getFailureStatus(), nil
		}

		// Although someone may have registered function in the meantime, this is still fine
		// as a function can be registered with Dandelion only once
		dr.registeredFunctions.Lock()
		dr.registeredFunctions.data[in.Name] = true
		dr.registeredFunctions.Unlock()
	}

	logrus.Debug("Sandbox creation took ", time.Since(start).Microseconds(), " μs")
	sandboxCreationDuration := time.Since(start)

	return &proto.SandboxCreationStatus{
		Success: true,
		ID:      uuid.New().String(),
		PortMappings: &proto.PortMapping{
			HostPort:  int32(dr.dandelionConfig.DaemonPort),
			GuestPort: in.PortForwarding.GuestPort,
			Protocol:  in.PortForwarding.Protocol,
		},
		LatencyBreakdown: &proto.SandboxCreationBreakdown{
			Total:         durationpb.New(sandboxCreationDuration),
			SandboxCreate: durationpb.New(sandboxCreationDuration),
		},
	}, nil
}

func (dr *Runtime) DeleteSandbox(_ context.Context, _ *proto.SandboxID) (*proto.ActionStatus, error) {
	return &proto.ActionStatus{Success: true}, nil
}

func (dr *Runtime) ListEndpoints(_ context.Context, _ *emptypb.Empty) (*proto.EndpointsList, error) {
	return dr.SandboxManager.ListEndpoints()
}

func (dr *Runtime) ValidateHostConfig() bool {
	return true
}

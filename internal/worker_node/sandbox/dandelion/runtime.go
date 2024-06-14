package dandelion

import (
	"bytes"
	"cluster_manager/internal/worker_node/managers"
	"cluster_manager/internal/worker_node/sandbox"
	"cluster_manager/pkg/config"
	"cluster_manager/proto"
	"context"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/emptypb"
	"io/ioutil"
	"net/http"
	"sync"
	"time"
)

type registerFunction struct {
	Name        string  `bson:"name"`
	ContextSize uint64  `bson:"context_size"`
	EngineType  string  `bson:"engine_type"`
	Binary      []int32 `bson:"binary"`
}

type registeredFunctions struct {
	data map[string]bool
	sync.RWMutex
}

type DandelionRuntime struct {
	sandbox.RuntimeInterface
	cpiApi              proto.CpiInterfaceClient
	SandboxManager      *managers.SandboxManager
	registeredFunctions *registeredFunctions
	matmulBinaryPath    string
	httpClient          *http.Client
}

func NewDandelionRuntime(cpApi proto.CpiInterfaceClient, config config.WorkerNodeConfig, sandboxManager *managers.SandboxManager, matmulBinary string) *DandelionRuntime {
	return &DandelionRuntime{
		cpiApi:         cpApi,
		SandboxManager: sandboxManager,
		registeredFunctions: &registeredFunctions{
			data: make(map[string]bool),
		},
		matmulBinaryPath: matmulBinary,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				DisableCompression:  true,
				IdleConnTimeout:     30 * time.Second,
				MaxIdleConns:        3000,
				MaxIdleConnsPerHost: 3000,
			},
		},
	}
}

// Patch for rust and golang bson libraries' different behavior
// rust: 1 byte -> 1 int32; golang: 1 byte -> 1 byte
func bytesToInts(binaryData []byte) []int32 {
	intData := make([]int32, len(binaryData))
	for i := 0; i < len(binaryData); i++ {
		intData[i] = int32(binaryData[i])
	}
	return intData
}

func (cr *DandelionRuntime) CreateSandbox(grpcCtx context.Context, in *proto.ServiceInfo) (*proto.SandboxCreationStatus, error) {
	failure_status := &proto.SandboxCreationStatus{
		Success: false,
		ID:      "-1",
	}

	logrus.Debug("Create sandbox for service = '", in.Name, "'")

	start := time.Now()

	cr.registeredFunctions.RLock()
	_, ok := cr.registeredFunctions.data[in.Name]
	cr.registeredFunctions.RUnlock()

	if !ok {
		// send register request to dandelion daemon
		binaryData, err := ioutil.ReadFile(cr.matmulBinaryPath)
		if err != nil {
			logrus.Errorf("Error reading binary file: %v", err)
			return failure_status, nil
		}
		logrus.Debugf("binary file size = %v", len(binaryData))

		registerRequest := registerFunction{
			Name:        in.Name,
			ContextSize: 0x8020000,
			Binary:      bytesToInts(binaryData),
			EngineType:  "RWasm",
		}
		registerRequestBody, err := bson.Marshal(registerRequest)
		if err != nil {
			logrus.Errorf("Error encoding register request: %v", err)
			return failure_status, nil
		}

		logrus.Debugf("send register request for function %v", in.Name)

		req, err := http.NewRequest("POST", "http://localhost:8082/register/function", bytes.NewBuffer(registerRequestBody))
		if err != nil {
			logrus.Errorf("Error constructing registration request: %v", err)
			return failure_status, nil
		}

		resp, err := cr.httpClient.Do(req)
		if err != nil {
			logrus.Debugf("failed to register function to dandelion worker: %v", err)
			return failure_status, nil
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			logrus.Debugf("Registration to dandelion worker is successful!")
		} else {
			logrus.Debugf("Registration failed, status code=%v", resp.StatusCode)
			return failure_status, nil
		}

		cr.registeredFunctions.Lock()
		cr.registeredFunctions.data[in.Name] = true
		cr.registeredFunctions.Unlock()
	}

	logrus.Debug("Sandbox creation took ", time.Since(start).Microseconds(), " μs (")
	sandboxCreationDuration := time.Since(start)

	return &proto.SandboxCreationStatus{
		Success: true,
		ID:      uuid.New().String(),
		PortMappings: &proto.PortMapping{
			HostPort:  8082,
			GuestPort: in.PortForwarding.GuestPort,
			Protocol:  in.PortForwarding.Protocol,
		},
		LatencyBreakdown: &proto.SandboxCreationBreakdown{
			Total:         durationpb.New(sandboxCreationDuration),
			SandboxCreate: durationpb.New(sandboxCreationDuration),
		},
	}, nil
}

func (dr *DandelionRuntime) DeleteSandbox(grpcCtx context.Context, in *proto.SandboxID) (*proto.ActionStatus, error) {
	return &proto.ActionStatus{Success: true}, nil
}

func (dr *DandelionRuntime) ListEndpoints(_ context.Context, _ *emptypb.Empty) (*proto.EndpointsList, error) {
	return dr.SandboxManager.ListEndpoints()
}

func (cr *DandelionRuntime) ValidateHostConfig() bool {
	return true
}

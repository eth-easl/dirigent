package dandelion

import (
	"bytes"
	"cluster_manager/internal/worker_node/managers"
	"cluster_manager/pkg/config"
	"cluster_manager/proto"
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/emptypb"
	"slices"
)

type registeredServices struct {
	data map[string]bool
	sync.RWMutex
}

type Runtime struct {
	cpApi               proto.CpiInterfaceClient
	SandboxManager      *managers.SandboxManager
	registeredFunctions *registeredServices
	registeredTasks     *registeredServices
	dandelionConfig     *config.DandelionConfig
	httpClient          *http.Client
}

func NewDandelionRuntime(cpApi proto.CpiInterfaceClient, sandboxManager *managers.SandboxManager, dandelionConfig *config.DandelionConfig) *Runtime {
	return &Runtime{
		cpApi:          cpApi,
		SandboxManager: sandboxManager,
		registeredFunctions: &registeredServices{
			data: make(map[string]bool),
		},
		registeredTasks: &registeredServices{
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

func (dr *Runtime) ConfigureNetwork(string) {}

func (dr *Runtime) registerService(path string, reqBson *bson.D) error {
	registerRequestBody, err := bson.Marshal(reqBson)
	if err != nil {
		return fmt.Errorf("error marshalling request body to BSON - %v", err)
	}

	registrationURL := fmt.Sprintf("http://localhost:%d%s", dr.dandelionConfig.DaemonPort, path)
	req, err := http.NewRequest("POST", registrationURL, bytes.NewBuffer(registerRequestBody))
	if err != nil {
		return fmt.Errorf("error creating http registration request - %v", err)
	}

	resp, err := dr.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("call to dandelion failed - %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return nil
	} else {
		return fmt.Errorf("registration request failed (status code: %d)", resp.StatusCode)
	}
}

func shardingStr(s uint32) string {
	switch s {
	case 1:
		return ":keyed "
	case 2:
		return ":each "
	default:
		return "" // empty is equal to :all
	}
}
func exportFunctionComposition(task *proto.WorkflowTaskInfo) string {
	if task == nil || len(task.Functions) == 0 {
		return ""
	}

	// function declarations and function applications
	funDecls := ""
	funAppls := ""
	var funDeclared []string
	dfIdx := 0
	shIdx := 0
	for fIdx, f := range task.Functions {

		// function declaration
		if !slices.Contains(funDeclared, f) {
			declIn := ""
			for i := int32(0); i < task.FunctionInNum[fIdx]; i++ {
				declIn += fmt.Sprintf("in%d", i)
				if i != task.FunctionInNum[fIdx]-1 {
					declIn += " "
				}
			}
			declOut := ""
			for i := int32(0); i < task.FunctionOutNum[fIdx]; i++ {
				declOut += fmt.Sprintf("out%d", i)
				if i != task.FunctionOutNum[fIdx]-1 {
					declOut += " "
				}
			}
			funDecls += fmt.Sprintf("(:function %s (%s) -> (%s))", f, declIn, declOut)
			funDeclared = append(funDeclared, f)
		}

		// function application
		applIn := ""
		for inIdx := int32(0); inIdx < task.FunctionInNum[fIdx]; inIdx++ {
			srcFIdx := task.FunctionDataFlow[dfIdx]
			srcFDataIdx := task.FunctionDataFlow[dfIdx+1]
			srcSharding := shardingStr(task.FunctionInSharding[shIdx])
			dfIdx += 2
			shIdx++
			if srcFIdx == -1 {
				applIn += fmt.Sprintf("(%sin%d <- cIn%d)", srcSharding, inIdx, srcFDataIdx)
			} else {
				applIn += fmt.Sprintf("(%sin%d <- f%dd%d)", srcSharding, inIdx, srcFIdx, srcFDataIdx)
			}
			if inIdx != task.FunctionInNum[fIdx]-1 {
				applIn += " "
			}
		}
		applOut := ""
		for outIdx := int32(0); outIdx < task.FunctionOutNum[fIdx]; outIdx++ {
			applOut += fmt.Sprintf("(f%dd%d := out%d)", fIdx, outIdx, outIdx)
			if outIdx != task.FunctionOutNum[fIdx]-1 {
				applOut += " "
			}
		}
		funAppls += fmt.Sprintf("(%s (%s) => (%s))", f, applIn, applOut)

		if fIdx != len(task.Functions)-1 {
			funAppls += " "
		}
	}

	// composition input
	compIn := ""
	for inIdx := uint32(0); inIdx < task.NumIn; inIdx++ {
		compIn += fmt.Sprintf("cIn%d", inIdx)
		if inIdx != task.NumIn-1 {
			compIn += " "
		}
	}

	// composition output
	compOut := ""
	for outIdx := uint32(0); outIdx < task.NumOut; outIdx++ {
		srcFIdx := task.FunctionDataFlow[dfIdx]
		srcFDataIdx := task.FunctionDataFlow[dfIdx+1]
		dfIdx += 2
		compOut += fmt.Sprintf("f%dd%d", srcFIdx, srcFDataIdx)
		if outIdx != task.NumOut-1 {
			compOut += " "
		}
	}

	return fmt.Sprintf("%s(:composition %s (%s) -> (%s) (%s))", funDecls, task.Name, compIn, compOut, funAppls)
}

func (dr *Runtime) CreateSandbox(_ context.Context, in *proto.ServiceInfo) (*proto.SandboxCreationStatus, error) {
	start := time.Now()
	logrus.Debug("Create sandbox for service = '", in.Name, "'")

	dr.registeredFunctions.RLock()
	_, ok := dr.registeredFunctions.data[in.Name]
	dr.registeredFunctions.RUnlock()

	if !ok {
		// load function binary
		binaryData, err := os.ReadFile(in.Image)
		logrus.Infof("Using binary file %s (len: %d)", in.Image, len(binaryData))
		if err != nil {
			logrus.Errorf("Error reading binary file - %v", err)
			return getFailureStatus(), nil
		}

		// create registration request body
		inputSets := bson.A{}
		for i := uint32(0); i < in.NumArgs; i++ { // names do not matter so far
			inputSets = append(inputSets, bson.A{fmt.Sprintf("input%d", i), nil})
		}
		outputSets := bson.A{} // add "stdio" for debugging purpose
		for i := uint32(0); i < in.NumArgs; i++ {
			outputSets = append(outputSets, fmt.Sprintf("output%d", i))
		}
		registerRequest := bson.D{
			{Key: "name", Value: in.Name},
			{Key: "context_size", Value: 0x8020000},
			{Key: "engine_type", Value: dr.dandelionConfig.EngineType},
			{Key: "binary", Value: bytesToInts(binaryData)},
			{Key: "input_sets", Value: inputSets},
			{Key: "output_sets", Value: outputSets},
		}

		// send registration request to dandelion
		err = dr.registerService("/register/function", &registerRequest)
		if err != nil {
			logrus.Errorf("Failed to register function '%s' - %v", in.Name, err)
			return getFailureStatus(), nil
		}
		logrus.Debugf("Successfully registered function '%s'", in.Name)

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

func (dr *Runtime) CreateTaskSandbox(_ context.Context, task *proto.WorkflowTaskInfo) (*proto.SandboxCreationStatus, error) {
	start := time.Now()
	logrus.Debug("Create sandbox for task '", task.Name, "'")

	dr.registeredTasks.RLock()
	_, ok := dr.registeredTasks.data[task.Name]
	dr.registeredTasks.RUnlock()

	if !ok {
		// export composition as dandelion composition description and create registration request body
		dandelionComposition := exportFunctionComposition(task)
		registerRequest := bson.D{
			{Key: "composition", Value: dandelionComposition},
		}

		// send registration request
		err := dr.registerService("/register/composition", &registerRequest)
		if err != nil {
			logrus.Errorf("Failed to register composition '%s' - %v", task.Name, err)
			return getFailureStatus(), nil
		}
		logrus.Debugf("Created sandbox for task '%s", task.Name)

		// Although someone may have registered the task in the meantime, this is still fine
		// as a composition can be registered with Dandelion only once
		dr.registeredTasks.Lock()
		dr.registeredTasks.data[task.Name] = true
		dr.registeredTasks.Unlock()
	}

	logrus.Debug("Sandbox creation took ", time.Since(start).Microseconds(), " μs")
	sandboxCreationDuration := time.Since(start)

	return &proto.SandboxCreationStatus{
		Success: true,
		ID:      uuid.New().String(),
		PortMappings: &proto.PortMapping{
			HostPort: int32(dr.dandelionConfig.DaemonPort),
		},
		LatencyBreakdown: &proto.SandboxCreationBreakdown{
			Total:         durationpb.New(sandboxCreationDuration),
			SandboxCreate: durationpb.New(sandboxCreationDuration),
		},
	}, nil
}

func (dr *Runtime) ListEndpoints(_ context.Context, _ *emptypb.Empty) (*proto.EndpointsList, error) {
	return dr.SandboxManager.ListEndpoints()
}

func (dr *Runtime) PrepullImage(_ context.Context, _ *proto.ImageInfo) (*proto.ActionStatus, error) {
	// TODO: Implement Firecracker image fetching.
	return &proto.ActionStatus{
		Success: false,
		Message: "Dandelion runtime does not currently support prepulling images.",
	}, nil
}

func (dr *Runtime) GetImages(_ context.Context) ([]*proto.ImageInfo, error) {
	// TODO: Implement Firecracker image fetching.
	return []*proto.ImageInfo{}, errors.New("image pulling in Dandelion not implemented yet")
}

func (dr *Runtime) ValidateHostConfig() bool {
	return true
}

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

var dandelionOperators = []string{"aggregate", "csv_reader", "csv_writer", "fetch", "filter", "hash_join", "project", "order", "splitter"}

type registeredServices struct {
	data map[string]bool
	sync.RWMutex
}

type Runtime struct {
	cpApi          proto.CpiInterfaceClient
	ip             string
	SandboxManager *managers.SandboxManager

	registeredFunctions *registeredServices
	registeredTasks     *registeredServices

	dandelionConfig *config.DandelionConfig
	httpClient      *http.Client
}

func NewDandelionRuntime(ip string, cpApi proto.CpiInterfaceClient, sandboxManager *managers.SandboxManager, dandelionConfig *config.DandelionConfig) *Runtime {
	return &Runtime{
		cpApi:          cpApi,
		ip:             ip,
		SandboxManager: sandboxManager,

		registeredFunctions: &registeredServices{data: make(map[string]bool)},
		registeredTasks:     &registeredServices{data: make(map[string]bool)},

		dandelionConfig: dandelionConfig,
		httpClient: &http.Client{
			Timeout: 30 * time.Second, // bigger binaries can take a while to register
			Transport: &http.Transport{
				IdleConnTimeout:     10 * time.Second,
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
		return "keyed"
	case 2:
		return "each"
	default:
		return "all"
	}
}

// NOTE: exportFunctionComposition has the following assumptions when using AddOperatorLoadAndStore:
//  1. we assume the first and last functions in the task are dandelion operators
//  2. we assume only the last function is composition output
func (dr *Runtime) exportFunctionComposition(task *proto.WorkflowTaskInfo) string {
	if task == nil || len(task.Functions) == 0 {
		return ""
	}

	addLoadAndStore := false
	if dr.dandelionConfig.AddOperatorLoadAndStore && slices.Contains(dandelionOperators, task.Functions[0]) {
		addLoadAndStore = true
	}

	// function declarations and function applications
	funDecls := ""
	funAppls := ""
	var funDeclared []string
	dfIdx := 0
	shIdx := 0
	for fIdx, f := range task.Functions {

		// operator load
		if addLoadAndStore {
			// operators are stored as a fetch/store composition with the actual function having a '_op' suffix attached
			if slices.Contains(dandelionOperators, f) {
				f = fmt.Sprintf("%s_op", f)
			}

			// add loading part at the start
			if fIdx == 0 {
				funDecls += "function fetch_requests(url) => (getRequest);"
				funDecls += "function HTTP(request) => (response);"
				if f == "csv_reader_op" { // -> reader: 2nd input -> data
					srcDataIdx := task.FunctionDataFlow[3]
					task.FunctionDataFlow[2] = -2
					funAppls += fmt.Sprintf("fetch_requests(url = each cIn%d) => (inDataReq = getRequest); ", srcDataIdx)
					funAppls += fmt.Sprintf("HTTP(request = each inDataReq) => (fetched%d = response); ", srcDataIdx)
				} else { // -> any other data operator: 2nd input -> schema, 3rd input -> batches (+ 4th and 5th for hash_join)
					numDataInput := 2
					if f == "hash_join_op" {
						numDataInput = 4
					}
					for i := 1; i <= numDataInput; i++ { // -> start at 1 since index 0 is operator option set
						task.FunctionDataFlow[i*2] = -2
						srcIdx := task.FunctionDataFlow[i*2+1]
						funAppls += fmt.Sprintf("fetch_requests(url = each cIn%d) => (inReq%d = getRequest); ", srcIdx, srcIdx)
						funAppls += fmt.Sprintf("HTTP(request = each inReq%d) => (fetched%d = response); ", srcIdx, srcIdx)
					}
				}
			}
		}

		// function declaration
		if !slices.Contains(funDeclared, f) {
			declIn := ""
			for i := int32(0); i < task.FunctionInNum[fIdx]; i++ {
				declIn += fmt.Sprintf("in%d", i)
				if i != task.FunctionInNum[fIdx]-1 {
					declIn += ", "
				}
			}
			declOut := ""
			for i := int32(0); i < task.FunctionOutNum[fIdx]; i++ {
				declOut += fmt.Sprintf("out%d", i)
				if i != task.FunctionOutNum[fIdx]-1 {
					declOut += ", "
				}
			}
			funDecls += fmt.Sprintf("function %s (%s) => (%s);", f, declIn, declOut)
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
				applIn += fmt.Sprintf("in%d = %s cIn%d", inIdx, srcSharding, srcFDataIdx)
			} else if srcFIdx == -2 { // operator load
				applIn += fmt.Sprintf("in%d = %s fetched%d", inIdx, srcSharding, srcFDataIdx)
			} else {
				applIn += fmt.Sprintf("in%d = %s f%dd%d", inIdx, srcSharding, srcFIdx, srcFDataIdx)
			}
			if inIdx != task.FunctionInNum[fIdx]-1 {
				applIn += ", "
			}
		}
		applOut := ""
		for outIdx := int32(0); outIdx < task.FunctionOutNum[fIdx]; outIdx++ {
			applOut += fmt.Sprintf("f%dd%d = out%d", fIdx, outIdx, outIdx)
			if outIdx != task.FunctionOutNum[fIdx]-1 {
				applOut += ", "
			}
		}
		funAppls += fmt.Sprintf("%s (%s) => (%s);", f, applIn, applOut)

		if fIdx != len(task.Functions)-1 {
			funAppls += " "
		}

		// operator store
		if addLoadAndStore && fIdx == len(task.Functions)-1 {
			inUrlSet := fmt.Sprintf("cIn%d", task.FunctionDataFlow[5])
			if task.Functions[0] == "csv_reader" {
				inUrlSet = fmt.Sprintf("cIn%d", task.FunctionDataFlow[3])
			}

			if f == "csv_writer_op" {
				funDecls += "function store_requests_data(inUrl, data) => (putRequest, outUrl);"
				funAppls += fmt.Sprintf(" store_requests_data(inUrl = all %s, data = each f%dd0) => (outDataReq = putRequest, outDataUrl = outUrl); ", inUrlSet, fIdx)
				funAppls += "HTTP(request = each outDataReq) => (_0 = response);"
			} else {
				funDecls += "function store_requests_schema(inUrl, data) => (putRequest, outUrl);"
				funDecls += "function store_requests_batches(inUrl, data) => (putRequest, outUrl);"
				funAppls += fmt.Sprintf(" store_requests_schema(inUrl = all %s, data = each f%dd0) => (outSchemaReq = putRequest, outSchemaUrl = outUrl); ", inUrlSet, fIdx)
				funAppls += "HTTP(request = each outSchemaReq) => (_0 = response); "
				funAppls += fmt.Sprintf("store_requests_batches(inUrl = all %s, data = each f%dd1) => (outBatchesReq = putRequest, outBatchesUrl = outUrl); ", inUrlSet, fIdx)
				funAppls += "HTTP(request = each outBatchesReq) => (_1 = response);"
			}
		}
	}

	// composition input
	compIn := ""
	for inIdx := uint32(0); inIdx < task.NumIn; inIdx++ {
		compIn += fmt.Sprintf("cIn%d", inIdx)
		if inIdx != task.NumIn-1 {
			compIn += ", "
		}
	}

	// composition output
	compOut := ""
	if addLoadAndStore {
		if task.Functions[len(task.Functions)-1] == "csv_writer" {
			compOut = "outDataUrl"
		} else {
			compOut = "outSchemaUrl, outBatchesUrl"
		}
	} else {
		for outIdx := uint32(0); outIdx < task.NumOut; outIdx++ {
			srcFIdx := task.FunctionDataFlow[dfIdx]
			srcFDataIdx := task.FunctionDataFlow[dfIdx+1]
			dfIdx += 2
			compOut += fmt.Sprintf("f%dd%d", srcFIdx, srcFDataIdx)
			if outIdx != task.NumOut-1 {
				compOut += ", "
			}
		}
	}

	return fmt.Sprintf("%s composition %s (%s) => (%s) {%s}", funDecls, task.Name, compIn, compOut, funAppls)
}

func addOperatorReadStore(operator string, withStdio bool) (string, string) {
	stdioOpDecl := ""
	stdioOpCall := ""
	stdioCompDef := ""
	if withStdio {
		stdioOpDecl = ", stdio"
		stdioOpCall = ", stdioOut = stdio"
		stdioCompDef = ", stdioOut"
	}

	var outComposition string
	switch operator {
	case "csv_reader":
		outComposition = fmt.Sprintf(
			`function csv_reader_op(options, inData) => (outSchema, outBatches%s);
			 function fetch_requests(url) => (getRequest);
			 function store_requests_schema(inUrl, data) => (putRequest, outUrl);
			 function store_requests_batches(inUrl, data) => (putRequest, outUrl);
			 function HTTP(request) => (response);

			 composition csv_reader (opOptions, inDataUrl) => (outSchemaUrl, outBatchesUrl%s) {
				 fetch_requests(url = all inDataUrl) => (inDataReq = getRequest);
				 HTTP(request = all inDataReq) => (fetchedData = response);
				 
				 csv_reader_op (options = all opOptions, inData = all fetchedData)
				   => (outSchema = outSchema, outBatches = outBatches%s);
	
				 store_requests_schema(inUrl = all inDataUrl, data = all outSchema) 
				   => (outSchemaReq = putRequest, outSchemaUrl = outUrl);
				 HTTP(request = all outSchemaReq) => (_0 = response);
				 store_requests_batches(inUrl = all inDataUrl, data = all outBatches) 
				   => (outBatchesReq = putRequest, outBatchesUrl = outUrl);
				 HTTP(request = all outBatchesReq) => (_1 = response);
			 }`, stdioOpDecl, stdioCompDef, stdioOpCall)
	case "csv_writer":
		outComposition = fmt.Sprintf(
			`function csv_writer_op(options, inSchema, inBatches) => (outData%s);
			 function fetch_requests(url) => (getRequest);
			 function store_requests_data(inUrl, data) => (putRequest, outUrl);
			 function HTTP(request) => (response);

			 composition csv_writer (opOptions, inSchemaUrl, inBatchesUrl) => (outDataUrl%s) {
				 fetch_requests(url = all inSchemaUrl) => (inSchemaReq = getRequest);
				 HTTP(request = all inSchemaReq) => (fetchedSchema = response);
				 fetch_requests(url = all inBatchesUrl) => (inBatchesReq = getRequest);
				 HTTP(request = all inBatchesReq) => (fetchedBatches = response);

				 csv_writer_op(options = all opOptions, inSchema = all fetchedSchema, inBatches = all fetchedBatches) 
				   => (outData = outData%s);

				 store_requests_data(inUrl = all inBatchesUrl, data = all outData) 
				   => (outDataReq = putRequest, outDataUrl = outUrl);
				 HTTP(request = all outDataReq) => (_0 = response);
			 }`, stdioOpDecl, stdioCompDef, stdioOpCall)
	case "hash_join":
		outComposition = fmt.Sprintf(
			`function hash_join_op(options, inSchema, inBatches, inSchema2, inBatches2) => (outSchema, outBatches%s);
			 function fetch_requests(url) => (getRequest);
			 function store_requests_schema(inUrl, data) => (putRequest, outUrl);
			 function store_requests_batches(inUrl, data) => (putRequest, outUrl);
			 function HTTP(request) => (response);

			 composition hash_join (opOptions, inSchemaUrl, inBatchesUrl, inSchemaUrl2, inBatchesUrl2) => (outSchemaUrl, outBatchesUrl%s) {
				 fetch_requests(url = all inSchemaUrl) => (inSchemaReq = getRequest);
				 HTTP(request = all inSchemaReq) => (fetchedSchema = response);
				 fetch_requests(url = all inBatchesUrl) => (inBatchesReq = getRequest);
				 HTTP(request = all inBatchesReq) => (fetchedBatches = response);
				 fetch_requests(url = all inSchemaUrl2) => (inSchemaReq2 = getRequest);
				 HTTP(request = all inSchemaReq2) => (fetchedSchema2 = response);
				 fetch_requests(url = all inBatchesUrl2) => (inBatchesReq2 = getRequest);
				 HTTP(request = all inBatchesReq2) => (fetchedBatches2 = response);

				 hash_join_op(options = all opOptions, inSchema = all fetchedSchema, inBatches = all fetchedBatches, inSchema2 = all fetchedSchema2, inBatches2 = all fetchedBatches2) 
				   => (outSchema = outSchema, outBatches = outBatches%s);

				 store_requests_schema(inUrl = all inBatchesUrl, data = all outSchema) 
				   => (outSchemaReq = putRequest, outSchemaUrl = outUrl);
				 HTTP(request = all outSchemaReq) => (_0 = response);
				 store_requests_batches(inUrl = all inBatchesUrl, data = all outBatches) 
				   => (outBatchesReq = putRequest, outBatchesUrl = outUrl);
				 HTTP(request = all outBatchesReq) => (_1 = response);
			 }`, stdioOpDecl, stdioCompDef, stdioOpCall)
	default:
		outComposition = fmt.Sprintf(
			`function %s_op(options, inSchema, inBatches) => (outSchema, outBatches%s);
			 function fetch_requests(url) => (getRequest);
			 function store_requests_schema(inUrl, data) => (putRequest, outUrl);
			 function store_requests_batches(inUrl, data) => (putRequest, outUrl);
			 function HTTP(request) => (response);

			 composition %s (opOptions, inSchemaUrl, inBatchesUrl) => (outSchemaUrl, outBatchesUrl%s) {
				 fetch_requests(url = all inSchemaUrl) => (inSchemaReq = getRequest);
				 HTTP(request = all inSchemaReq) => (fetchedSchema = response);
				 fetch_requests(url = all inBatchesUrl) => (inBatchesReq = getRequest);
				 HTTP(request = all inBatchesReq) => (fetchedBatches = response);

				 %s_op(options = all opOptions, inSchema = all fetchedSchema, inBatches = all fetchedBatches) 
				   => (outSchema = outSchema, outBatches = outBatches%s);

				 store_requests_schema(inUrl = all inBatchesUrl, data = all outSchema) 
				   => (outSchemaReq = putRequest, outSchemaUrl = outUrl);
				 HTTP(request = all outSchemaReq) => (_0 = response);
				 store_requests_batches(inUrl = all inBatchesUrl, data = all outBatches) 
				   => (outBatchesReq = putRequest, outBatchesUrl = outUrl);
				 HTTP(request = all outBatchesReq) => (_1 = response);
			 }`, operator, stdioOpDecl, operator, stdioCompDef, operator, stdioOpCall)
	}

	// return new name for the operator function + operator composition
	return fmt.Sprintf("%s_op", operator), outComposition
}

func (dr *Runtime) registerHelpers(ctx context.Context) (*proto.SandboxCreationStatus, error) {
	dr.registeredFunctions.RLock()
	_, ok := dr.registeredFunctions.data["fetch_requests"]
	dr.registeredFunctions.RUnlock()
	if ok {
		return nil, nil
	}

	status, err := dr.CreateSandbox(ctx, &proto.ServiceInfo{
		Name:    "fetch_requests",
		Image:   "/users/tstocker/operators/ops_export/fetch_requests",
		NumArgs: 1,
		NumRets: 1,
	})
	if err != nil {
		return status, err
	}

	status, err = dr.CreateSandbox(ctx, &proto.ServiceInfo{
		Name:    "store_requests_schema",
		Image:   "/users/tstocker/operators/ops_export/store_requests_schema",
		NumArgs: 2,
		NumRets: 2,
	})
	if err != nil {
		return status, err
	}

	status, err = dr.CreateSandbox(ctx, &proto.ServiceInfo{
		Name:    "store_requests_batches",
		Image:   "/users/tstocker/operators/ops_export/store_requests_batches",
		NumArgs: 2,
		NumRets: 2,
	})
	if err != nil {
		return status, err
	}

	status, err = dr.CreateSandbox(ctx, &proto.ServiceInfo{
		Name:    "store_requests_data",
		Image:   "/users/tstocker/operators/ops_export/store_requests_data",
		NumArgs: 2,
		NumRets: 2,
	})
	if err != nil {
		return status, err
	}

	return status, nil
}

func (dr *Runtime) CreateSandbox(ctx context.Context, in *proto.ServiceInfo) (*proto.SandboxCreationStatus, error) {
	start := time.Now()
	logrus.Debug("Create sandbox for service = '", in.Name, "'")

	dr.registeredFunctions.RLock()
	_, ok := dr.registeredFunctions.data[in.Name]
	dr.registeredFunctions.RUnlock()

	if !ok {
		// check if binary exists
		binaryInfo, err := os.Stat(in.Image)
		if err != nil {
			logrus.Errorf("Error validating binary file - %v", err)
			return getFailureStatus(), nil
		}
		if binaryInfo.IsDir() {
			logrus.Errorf("Error validating binary file - file is a directory")
			return getFailureStatus(), nil
		}
		logrus.Infof("Registering binary file %s (size=%d)", in.Image, binaryInfo.Size())

		// operator load/store
		fName := in.Name
		var opComposition string
		if dr.dandelionConfig.AddOperatorLoadAndStore && slices.Contains(dandelionOperators, fName) {
			s, e := dr.registerHelpers(ctx)
			if e != nil {
				return s, e
			}
			fName, opComposition = addOperatorReadStore(in.Name, dr.dandelionConfig.LogFunctionStdioOutset)
		}

		// create registration request body
		inputSets := bson.A{}
		for i := uint32(0); i < in.NumArgs; i++ { // names do not matter so far
			inputSets = append(inputSets, bson.A{fmt.Sprintf("input%d", i), nil})
		}
		outputSets := bson.A{}
		for i := uint32(0); i < in.NumRets; i++ {
			outputSets = append(outputSets, fmt.Sprintf("output%d", i))
		}
		if dr.dandelionConfig.LogFunctionStdioOutset { // add "stdio" to get the stdio output from dandelion
			outputSets = append(outputSets, "stdio")
		}
		registerRequest := bson.D{
			{Key: "name", Value: fName},
			{Key: "context_size", Value: 0x8020000},
			{Key: "engine_type", Value: dr.dandelionConfig.EngineType},
			{Key: "local_path", Value: in.Image},
			{Key: "binary", Value: bytesToInts([]byte{0})},
			{Key: "input_sets", Value: inputSets},
			{Key: "output_sets", Value: outputSets},
		}

		// send registration request to dandelion
		err = dr.registerService("/register/function", &registerRequest)
		if err != nil {
			logrus.Errorf("Failed to register function '%s' - %v", fName, err)
			return getFailureStatus(), nil
		}
		logrus.Debugf("Successfully registered function '%s'", fName)

		// Although someone may have registered function in the meantime, this is still fine
		// as a function can be registered with Dandelion only once
		dr.registeredFunctions.Lock()
		dr.registeredFunctions.data[in.Name] = true
		dr.registeredFunctions.Unlock()

		// operator load/store
		if dr.dandelionConfig.AddOperatorLoadAndStore && slices.Contains(dandelionOperators, in.Name) {
			compRegistrationRequest := bson.D{
				{Key: "composition", Value: opComposition},
			}

			// send registration request
			err := dr.registerService("/register/composition", &compRegistrationRequest)
			if err != nil {
				logrus.Errorf("Failed to register composition '%s' - %v", in.Name, err)
				return getFailureStatus(), nil
			}
			logrus.Debugf("Created composition for task '%s", in.Name)
		}
	}

	logrus.Debug("Sandbox creation took ", time.Since(start).Microseconds(), " μs")
	sandboxCreationDuration := time.Since(start)

	return &proto.SandboxCreationStatus{
		Success: true,
		ID:      uuid.New().String(),
		URL:     fmt.Sprintf("%s:%d", dr.ip, dr.dandelionConfig.DaemonPort),
		LatencyBreakdown: &proto.SandboxCreationBreakdown{
			Total:         durationpb.New(sandboxCreationDuration),
			SandboxCreate: durationpb.New(sandboxCreationDuration),
		},
	}, nil
}

func (dr *Runtime) DeleteSandbox(_ context.Context, _ *proto.SandboxID) (*proto.ActionStatus, error) {
	return &proto.ActionStatus{Success: true}, nil
}

func (dr *Runtime) CreateTaskSandbox(ctx context.Context, task *proto.WorkflowTaskInfo) (*proto.SandboxCreationStatus, error) {
	start := time.Now()
	logrus.Debug("Create sandbox for task '", task.Name, "'")

	dr.registeredTasks.RLock()
	_, ok := dr.registeredTasks.data[task.Name]
	dr.registeredTasks.RUnlock()

	if !ok {
		// check that all composition functions are already registered
		dr.registeredTasks.RLock()
		for _, funcName := range task.Functions {
			if _, isRegistered := dr.registeredFunctions.data[funcName]; !isRegistered {
				logrus.Errorf("Could not register composition '%s' as function '%s' is not yet registered in runtime", task.Name, funcName)
				return getFailureStatus(), nil
			}
		}
		dr.registeredTasks.RUnlock()

		// operator load/store
		if dr.dandelionConfig.AddOperatorLoadAndStore {
			firstIsOperator := slices.Contains(dandelionOperators, task.Functions[0])
			lastIsOperator := slices.Contains(dandelionOperators, task.Functions[len(task.Functions)-1])
			if firstIsOperator && lastIsOperator {
				s, e := dr.registerHelpers(ctx)
				if e != nil {
					return s, e
				}
			} else if firstIsOperator || lastIsOperator {
				logrus.Errorf("Cannot mix operators and non-operators in first and last function of composition.")
				return getFailureStatus(), nil
			}
		}

		// export composition as dandelion composition description and create registration request body
		dandelionComposition := dr.exportFunctionComposition(task)
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
		URL:     fmt.Sprintf("%s:%d", dr.ip, dr.dandelionConfig.DaemonPort),
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

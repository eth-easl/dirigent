package proxy

import (
	"bufio"
	"bytes"
	common "cluster_manager/internal/data_plane/function_metadata"
	"cluster_manager/internal/data_plane/proxy/load_balancing"
	"cluster_manager/internal/data_plane/proxy/metrics_collection"
	"cluster_manager/internal/data_plane/proxy/requests"
	"cluster_manager/internal/data_plane/workflow"
	"cluster_manager/internal/data_plane/workflow/scheduler"
	"cluster_manager/pkg/tracing"
	"cluster_manager/proto"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

type proxyContext struct {
	cache               *common.Deployments
	cpInterface         proto.CpiInterfaceClient
	loadBalancingPolicy load_balancing.LoadBalancingPolicy

	tracing *tracing.TracingService[tracing.ProxyLogEntry]

	incomingRequestChannel chan string
	doneRequestChannel     chan metrics_collection.DurationInvocation
}

type requestMetadata struct {
	start            time.Time
	serialization    time.Duration
	persistenceLayer time.Duration
}

func startProxy(handler http.HandlerFunc, wfHandler http.HandlerFunc, context proxyContext, host string, port string) {
	mux := http.NewServeMux()
	mux.Handle("/", handler)
	mux.HandleFunc("/health", HealthHandler)
	mux.HandleFunc("/metrics", CreateMetricsHandler(context.cache))
	mux.HandleFunc("/workflow", wfHandler)

	proxyRxAddress := net.JoinHostPort(host, port)
	logrus.Info("Creating a proxy server at ", proxyRxAddress)

	server := &http.Server{
		Addr:    proxyRxAddress,
		Handler: h2c.NewHandler(mux, &http2.Server{}),
	}

	err := server.ListenAndServe()
	if err != nil {
		logrus.Fatalf("Failed to create a proxy server : (err : %s)", err.Error())
	}
}

func proxyHandler(proxy *httputil.ReverseProxy, writer http.ResponseWriter, request *http.Request, requestMetadata requestMetadata, proxyContext proxyContext) *requests.BufferedResponse {
	///////////////////////////////////////////////
	// METADATA FETCHING
	///////////////////////////////////////////////
	// TODO: why use Host attribute of request and not the body?
	serviceName := GetServiceName(request)
	metadata, durationGetDeployment := proxyContext.cache.GetDeployment(serviceName)
	if metadata == nil {
		logrus.Trace("Invocation for non-existing service ", serviceName, " has been dumped.")
		return &requests.BufferedResponse{
			StatusCode: http.StatusBadRequest,
			Body:       "Invocation for non-existing service " + serviceName + " has been dumped.",
			E2ELatency: time.Since(requestMetadata.start),
		}
	}

	// Notify metric collector we got a request
	proxyContext.incomingRequestChannel <- serviceName

	logrus.Trace("Invocation for service ", serviceName, " has been received.")

	///////////////////////////////////////////////
	// COLD/WARM START
	///////////////////////////////////////////////
	coldStartChannel, durationColdStart := metadata.TryWarmStart(proxyContext.cpInterface)
	addDeploymentDuration := time.Duration(0)

	defer metadata.GetStatistics().DecrementInflight()

	// unblock from cold start if context gets cancelled
	go contextTerminationHandler(request, coldStartChannel)

	if coldStartChannel != nil {
		logrus.Debug("Enqueued invocation for ", serviceName)

		// wait until a cold start is resolved
		coldStartWaitTime := time.Now()
		waitOutcome := <-coldStartChannel
		metadata.GetStatistics().DecrementQueueDepth()
		durationColdStart = time.Since(coldStartWaitTime) - waitOutcome.AddEndpointDuration
		addDeploymentDuration = waitOutcome.AddEndpointDuration

		// TODO: Resend the request in the channel & execute until it works
		if waitOutcome.Outcome == common.CanceledColdStart {
			return &requests.BufferedResponse{
				StatusCode: http.StatusGatewayTimeout,
				Body:       fmt.Sprintf("Invocation context canceled for %s.", serviceName),
				E2ELatency: time.Since(requestMetadata.start),
			}
		}
	}

	///////////////////////////////////////////////
	// LOAD BALANCING AND ROUTING
	///////////////////////////////////////////////
	endpoint, durationLB, durationCC := load_balancing.DoLoadBalancing(request, metadata, proxyContext.loadBalancingPolicy)
	if endpoint == nil {
		return &requests.BufferedResponse{
			StatusCode: http.StatusGone,
			Body:       fmt.Sprintf("Cold start passed, but no sandbox available for %s.", serviceName),
			E2ELatency: time.Since(requestMetadata.start),
		}
	}

	defer giveBackCCCapacity(endpoint)

	///////////////////////////////////////////////
	// PROXYING - LEAVING THE MACHINE
	///////////////////////////////////////////////
	startProxy := time.Now()

	proxy.ServeHTTP(writer, request)

	// Notify metric collector we got a response
	proxyContext.doneRequestChannel <- metrics_collection.DurationInvocation{
		Duration:    max(uint32(time.Since(startProxy).Seconds()), 1),
		ServiceName: serviceName,
	}

	///////////////////////////////////////////////
	// ON THE WAY BACK
	///////////////////////////////////////////////

	proxyContext.tracing.InputChannel <- tracing.ProxyLogEntry{
		ServiceName: serviceName,
		ContainerID: endpoint.ID,

		StartTime:     requestMetadata.start,
		Total:         time.Since(requestMetadata.start),
		GetMetadata:   durationGetDeployment,
		AddDeployment: addDeploymentDuration,
		ColdStart:     durationColdStart,
		LoadBalancing: durationLB,
		CCThrottling:  durationCC,
		Proxying:      time.Since(startProxy),

		Serialization:    requestMetadata.serialization,
		PersistenceLayer: requestMetadata.persistenceLayer,
	}
	metadata.GetStatistics().IncrementSuccessfulInvocations()

	return &requests.BufferedResponse{
		// Body is written by proxy.ServerHTTP()
		StatusCode: http.StatusOK,
		E2ELatency: time.Since(requestMetadata.start),
	}
}

// TODO: refactor this + implement metrics stuff
//
//	-> maybe create a function that can handle single function invocation + scheduling functions for workflows
func functionInvocationRequest(wfRequest *http.Request, proxyContext *proxyContext, serviceName string, reqBody []byte) (*http.Request, error) {
	metadata, _ := proxyContext.cache.GetDeployment(serviceName)
	proxyContext.incomingRequestChannel <- serviceName

	///////////////////////////////////////////////
	// COLD/WARM START
	///////////////////////////////////////////////
	coldStartChannel, _ := metadata.TryWarmStart(proxyContext.cpInterface)

	defer metadata.GetStatistics().DecrementInflight()

	// unblock from cold start if context gets cancelled
	go contextTerminationHandler(wfRequest, coldStartChannel)

	if coldStartChannel != nil {
		logrus.Debug("Enqueued invocation for ", serviceName)

		// wait until a cold start is resolved
		waitOutcome := <-coldStartChannel
		metadata.GetStatistics().DecrementQueueDepth()

		if waitOutcome.Outcome == common.CanceledColdStart {
			return nil, fmt.Errorf("cold start failed or got wfRequest got canceled")
		}
	}

	///////////////////////////////////////////////
	// CREATE REQUEST THAT IS SENT TO SANDBOX
	///////////////////////////////////////////////
	funcReq, err := http.NewRequest("GET", "http://localhost/hot/matmul", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create the function invocation request for the worker (%v)", err)
	}

	///////////////////////////////////////////////
	// LOAD BALANCING AND ROUTING
	///////////////////////////////////////////////
	endpoint, _, _ := load_balancing.DoLoadBalancing(funcReq, metadata, proxyContext.loadBalancingPolicy)
	if endpoint == nil {
		return nil, fmt.Errorf("cold start passed but no sandbox available for %s.", serviceName)
	}
	defer giveBackCCCapacity(endpoint)

	return funcReq, nil
}

// TODO: refactor (same as above)
func scheduleWorkflowFunction(httpClient *http.Client, wfRequest *http.Request, proxyContext *proxyContext) scheduler.ScheduleTaskFunc {
	return func(service *workflow.Statement) error {
		logrus.Tracef("Invoking workflow function '%s'", service.Name)

		// create the function invocation request
		funcReqBody, err := workflow.BusyLoopInvocationBody(service.Name, service.GetInData())
		if err != nil {
			return err
		}
		funcReq, err := functionInvocationRequest(wfRequest, proxyContext, service.Name, funcReqBody)
		if err != nil {
			return err
		}

		// execute the request
		resp, err := httpClient.Do(funcReq)
		if err != nil {
			return fmt.Errorf("HTTP client call for '%s' failed (%v)", service.Name, err)
		}
		logrus.Tracef("Workflow function invocation (%s) returned status code %d", service.Name, resp.StatusCode)

		defer resp.Body.Close()
		respBody, err := io.ReadAll(resp.Body)
		if err != nil || resp.StatusCode != http.StatusOK {
			if err != nil {
				return fmt.Errorf("HTTP request failed for '%s', got invalid response (cannot read body: %v)", service.Name, err)
			}
			if len(respBody) == 0 {
				return fmt.Errorf("HTTP request failed for '%s', got status code %s", service.Name, resp.StatusCode)
			}
			return fmt.Errorf("HTTP request failed for '%s', got status code: %s, response body: { %s }", service.Name, resp.Status, string(respBody))
		}

		// collect data and update the scheduler
		outData, err := workflow.DeserializeResponseToData(respBody)
		if err != nil {
			return err
		}
		err = service.SetOutData(outData)
		if err != nil {
			return err
		}

		return nil
	}
}

func workflowHandler(w http.ResponseWriter, r *http.Request, proxyContext proxyContext) {
	// TODO: currently uses a POST request containing a workflow name and description in the body
	// 		 -> in the future move to registration in control plane + invocation here using a GET request
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed.", http.StatusMethodNotAllowed)
		return
	}

	// Parse request
	logrus.Infof("Received workflow request.")
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Failed to parse workflow request.", http.StatusBadRequest)
		return
	}
	name := r.FormValue("name")
	if len(name) == 0 {
		logrus.Errorf("Invalid workflow request (empty workflow name).")
		http.Error(w, "Invalid workflow name.", http.StatusBadRequest)
		return
	}
	wfString := r.FormValue("workflow")
	if len(wfString) == 0 {
		logrus.Errorf("Invalid workflow request (empty workflow description).")
		http.Error(w, "Empty workflow description.", http.StatusBadRequest)
		return
	}
	input := r.FormValue("input")

	// Parse workflow
	wfParser := workflow.NewParser(bufio.NewReader(strings.NewReader(wfString)))
	wf, err := wfParser.Parse()
	if err != nil {
		logrus.Errorf("Failed to parse workflow '%s': %v", name, err)
		http.Error(w, "Invalid workflow.", http.StatusBadRequest)
		return
	}

	// Prepare workflow input
	inData, err := workflow.DeserializeRequestToData([]byte(input))
	if err != nil {
		logrus.Errorf("Invalid workflow request (failed to deserialize input data): %v).", err)
		http.Error(w, "Invalid workflow input.", http.StatusBadRequest)
		return
	}

	// Execute workflow
	logrus.Debugf("Executing workflow '%s'.", name)
	httpClient := &http.Client{Timeout: 10 * time.Second}
	wfScheduler := scheduler.NewScheduler(wf, scheduler.SequentialFifo)
	err = wfScheduler.Schedule(scheduleWorkflowFunction(httpClient, r, &proxyContext), inData)
	if err != nil {
		logrus.Errorf("Failed while executing workflow %s: %v", name, err)
		http.Error(w, "Failed to execute workflow.", http.StatusInternalServerError)
		return
	}

	// collect workflow output
	w.WriteHeader(http.StatusOK)
	outData := wf.Compositions[0].CollectOutData()
	outBody, err := workflow.GetResponseBody(outData)
	if err != nil {
		w.Write([]byte(fmt.Sprintf("Failed to add workflow output: %s", err)))
	} else {
		w.Write(outBody)
	}

}

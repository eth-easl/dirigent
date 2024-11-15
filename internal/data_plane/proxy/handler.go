package proxy

import (
	"bytes"
	"cluster_manager/internal/data_plane/proxy/load_balancing"
	"cluster_manager/internal/data_plane/proxy/metrics_collection"
	"cluster_manager/internal/data_plane/proxy/requests"
	"cluster_manager/internal/data_plane/service_metadata"
	common "cluster_manager/internal/data_plane/service_metadata"
	"cluster_manager/internal/data_plane/workflow"
	"cluster_manager/internal/data_plane/workflow/scheduler"
	"cluster_manager/pkg/config"
	"cluster_manager/pkg/tracing"
	"cluster_manager/proto"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
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

func startProxy(handler http.HandlerFunc, context proxyContext, host string, port string) {
	mux := http.NewServeMux()
	mux.Handle("/", handler)
	mux.HandleFunc("/health", HealthHandler)
	mux.HandleFunc("/metrics", CreateMetricsHandler(context.cache))

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

func handleFunction(proxy *httputil.ReverseProxy, writer http.ResponseWriter, request *http.Request, reqMetadata *service_metadata.ServiceMetadata, proxyContext proxyContext) (*tracing.ProxyLogEntry, *requests.BufferedResponse) {
	serviceName := reqMetadata.GetIdentifier()

	///////////////////////////////////////////////
	// COLD/WARM START
	///////////////////////////////////////////////
	coldStartChannel, durationColdStart := reqMetadata.TryWarmStart(proxyContext.cpInterface)
	addDeploymentDuration := time.Duration(0)

	defer reqMetadata.GetStatistics().DecrementInflight()

	// unblock from cold start if context gets cancelled
	go contextTerminationHandler(request.Context(), coldStartChannel)

	if coldStartChannel != nil {
		logrus.Debug("Enqueued invocation for ", serviceName)

		// wait until a cold start is resolved
		coldStartWaitTime := time.Now()
		waitOutcome := <-coldStartChannel
		reqMetadata.GetStatistics().DecrementQueueDepth()
		durationColdStart = time.Since(coldStartWaitTime) - waitOutcome.AddEndpointDuration
		addDeploymentDuration = waitOutcome.AddEndpointDuration

		// TODO: Resend the request in the channel & execute until it works
		if waitOutcome.Outcome == common.CanceledColdStart {
			return nil, &requests.BufferedResponse{
				StatusCode: http.StatusGatewayTimeout,
				Body:       fmt.Sprintf("Invocation context canceled for %s.", serviceName),
			}
		}
	}

	///////////////////////////////////////////////
	// LOAD BALANCING AND ROUTING
	///////////////////////////////////////////////
	endpoint, durationLB, durationCC := load_balancing.DoLoadBalancing(request, reqMetadata, proxyContext.loadBalancingPolicy)
	if endpoint == nil {
		return nil, &requests.BufferedResponse{
			StatusCode: http.StatusGone,
			Body:       fmt.Sprintf("Cold start passed, but no sandbox available for %s.", serviceName),
		}
	}

	defer giveBackCCCapacity(endpoint)

	///////////////////////////////////////////////
	// PROXYING - LEAVING THE MACHINE
	///////////////////////////////////////////////
	proxyStartTime := time.Now()

	proxy.ServeHTTP(writer, request)

	// Notify metric collector we got a response
	proxyContext.doneRequestChannel <- metrics_collection.DurationInvocation{
		Duration:    max(uint32(time.Since(proxyStartTime).Seconds()), 1),
		ServiceName: serviceName,
	}

	///////////////////////////////////////////////
	// ON THE WAY BACK
	///////////////////////////////////////////////

	logEntry := &tracing.ProxyLogEntry{
		ServiceName: serviceName,
		ContainerID: endpoint.ID,

		AddDeployment: addDeploymentDuration,
		ColdStart:     durationColdStart,
		LoadBalancing: durationLB,
		CCThrottling:  durationCC,
		Proxying:      time.Since(proxyStartTime),
	}
	reqMetadata.GetStatistics().IncrementSuccessfulInvocations()

	return logEntry, &requests.BufferedResponse{
		// Body is written by proxy.ServerHTTP()
		StatusCode: http.StatusOK,
	}
}

func scheduleWorkflowFunction(httpClient *http.Client, proxyCtx *proxyContext) scheduler.ScheduleTaskFunc {
	return func(orchestrator *workflow.TaskOrchestrator, schedulerTask *workflow.SchedulerTask, ctx context.Context) error {
		task := schedulerTask.GetTask()
		var serviceName string
		if len(task.Functions) > 1 {
			serviceName = task.Name
		} else {
			serviceName = task.Functions[0]
		}

		// metadata fetching
		metadata, _ := proxyCtx.cache.GetServiceMetadata(serviceName)
		if metadata == nil {
			return fmt.Errorf("no deployment found for function '%s'", serviceName)
		}

		// cold/warm start
		coldStartChannel, _ := metadata.TryWarmStart(proxyCtx.cpInterface)
		defer metadata.GetStatistics().DecrementInflight()
		go contextTerminationHandler(ctx, coldStartChannel)
		if coldStartChannel != nil {
			logrus.Debug("Enqueued invocation for ", serviceName)

			// wait until a cold start is resolved
			waitOutcome := <-coldStartChannel
			metadata.GetStatistics().DecrementQueueDepth()

			// TODO: Resend the request in the channel & execute until it works
			if waitOutcome.Outcome == common.CanceledColdStart {
				return fmt.Errorf("invocation context canceled for '%s' (workflow request canceled?)", serviceName)
			}
		}

		// create function invocation request
		funcReqBody, err := workflow.InvocationBody(serviceName, orchestrator.GetInData(schedulerTask))
		if err != nil {
			return fmt.Errorf("failed to get input data for task '%s': %v", serviceName, err)
		}
		funcReq, err := http.NewRequest("GET", "http://localhost/hot/matmul", bytes.NewBuffer(funcReqBody))
		if err != nil {
			return fmt.Errorf("failed to create invocation request for function '%s': %v", serviceName, err)
		}

		// load balancing and routing
		endpoint, _, _ := load_balancing.DoLoadBalancing(funcReq, metadata, proxyCtx.loadBalancingPolicy)
		if endpoint == nil {
			return fmt.Errorf("cold start passed, but no sandbox available for '%s'", serviceName)
		}

		defer giveBackCCCapacity(endpoint)

		// execute the request
		resp, err := httpClient.Do(funcReq)
		if err != nil {
			return fmt.Errorf("HTTP client call for '%s' failed (%v)", serviceName, err)
		}
		logrus.Tracef("Workflow function invocation (%s) returned status code %d", serviceName, resp.StatusCode)

		defer resp.Body.Close()
		respBody, err := io.ReadAll(resp.Body)
		if err != nil || resp.StatusCode != http.StatusOK {
			if err != nil {
				return fmt.Errorf("HTTP request failed for '%s', got invalid response (cannot read body: %v)", serviceName, err)
			}
			if len(respBody) == 0 {
				return fmt.Errorf("HTTP request failed for '%s', got status code %d", serviceName, resp.StatusCode)
			}
			return fmt.Errorf("HTTP request failed for '%s', got status code: %s, response body: { %s }", serviceName, resp.Status, string(respBody))
		}

		// collect data and update the task orchestrator
		outData, err := workflow.DeserializeResponseToData(respBody)
		if err != nil {
			return err
		}
		err = orchestrator.SetOutData(schedulerTask, outData)
		if err != nil {
			return err
		}

		metadata.GetStatistics().IncrementSuccessfulInvocations()

		return nil
	}
}

func handleWorkflow(writer http.ResponseWriter, request *http.Request, wf *workflow.Workflow, proxyCtx *proxyContext, dpConfig *config.DataPlaneConfig) *requests.BufferedResponse {
	// parse request
	err := request.ParseForm()
	if err != nil {
		logrus.Errorf("Error parsing request form: %v", err)
		return &requests.BufferedResponse{
			StatusCode: http.StatusBadRequest,
			Body:       "Failed to parse workflow request.",
		}
	}
	input := request.FormValue("input")
	reqSchedulerType := request.FormValue("schedulerType")

	// set scheduler type
	var schedulerType scheduler.SchedulerType
	if len(reqSchedulerType) > 0 {
		schedulerType = scheduler.SchedulerTypeFromString(reqSchedulerType)
		if schedulerType == scheduler.Invalid {
			logrus.Errorf("Invalid scheduler type '%s' in request.", reqSchedulerType)
			return &requests.BufferedResponse{
				StatusCode: http.StatusBadRequest,
				Body:       fmt.Sprintf("Invalid scheduler type '%s'.", reqSchedulerType),
			}
		}
	} else {
		schedulerType = scheduler.SchedulerTypeFromString(dpConfig.WorkflowDefaultScheduler)
		if schedulerType == scheduler.Invalid {
			logrus.Errorf("Invalid default scheduler type '%s'.", dpConfig.WorkflowDefaultScheduler)
			return &requests.BufferedResponse{
				StatusCode: http.StatusInternalServerError,
				Body:       "Invalid default scheduler type.",
			}
		}
	}

	// prepare workflow input
	inData, err := workflow.DeserializeRequestToData([]byte(input))
	if err != nil {
		logrus.Errorf("Invalid workflow request (failed to deserialize input data): %v.", err)
		return &requests.BufferedResponse{
			StatusCode: http.StatusBadRequest,
			Body:       fmt.Sprintf("Invalid workflow request (failed to deserialize input data): %v.", err),
		}
	}

	// schedule workflow
	httpClient := &http.Client{Timeout: 10 * time.Second}
	wfScheduler := scheduler.NewScheduler(wf, schedulerType)
	err = wfScheduler.Schedule(scheduleWorkflowFunction(httpClient, proxyCtx), inData, dpConfig, request.Context())
	if err != nil {
		logrus.Errorf("Failed while executing workflow %s: %v", wf.Name, err)
		return &requests.BufferedResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       "Failed to execute workflow.",
		}
	}

	// collect workflow output
	outData, err := wfScheduler.CollectOutput()
	if err != nil {
		logrus.Errorf("Failed to collect output of workflow %s: %v", wf.Name, err)
		return &requests.BufferedResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       "Failed to collect workflow output.",
		}
	}

	// write collected data in response
	outBody, err := workflow.GetResponseBody(outData)
	if err != nil {
		logrus.Errorf("Failed create response body for workflow %s: %v", wf.Name, err)
		return &requests.BufferedResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       "Failed create response body.",
		}
	}

	writer.WriteHeader(http.StatusOK)
	_, err = writer.Write(outBody)
	if err != nil {
		logrus.Errorf("Failed write response body for workflow %s: %v", wf.Name, err)
	}
	return &requests.BufferedResponse{
		StatusCode: http.StatusOK,
		Body:       string(outBody),
	}
}

func proxyHandler(proxy *httputil.ReverseProxy, writer http.ResponseWriter, request *http.Request, requestMetadata requestMetadata, proxyContext proxyContext, dpConfig *config.DataPlaneConfig) *requests.BufferedResponse {
	// fetch metadata
	serviceName := GetServiceName(request)
	deployment, durationGetDeployment := proxyContext.cache.GetDeployment(serviceName)
	if deployment == nil {
		logrus.Trace("Invocation for non-existing service ", serviceName, " has been dumped.")
		return &requests.BufferedResponse{
			StatusCode: http.StatusBadRequest,
			Body:       "Invocation for non-existing service " + serviceName + " has been dumped.",
			E2ELatency: time.Since(requestMetadata.start),
		}
	}

	// handle invocation
	var resp *requests.BufferedResponse
	switch deployment.GetType() {
	case service_metadata.Function:
		logrus.Tracef("Invocation for function '%s' has been received.", serviceName)

		// notify metric collector we got a request
		proxyContext.incomingRequestChannel <- serviceName

		// handle function invocation
		var logEntry *tracing.ProxyLogEntry
		logEntry, resp = handleFunction(proxy, writer, request, deployment.GetFunction(), proxyContext)

		// extend log entry and send it to the tracing service
		if resp.StatusCode == http.StatusOK {
			logEntry.GetMetadata = durationGetDeployment
			logEntry.StartTime = requestMetadata.start
			logEntry.Total = time.Since(requestMetadata.start)
			logEntry.Serialization = requestMetadata.serialization
			logEntry.PersistenceLayer = requestMetadata.persistenceLayer

			proxyContext.tracing.InputChannel <- *logEntry
		}
	case service_metadata.Workflow:
		logrus.Tracef("Invocation for workflow '%s' has been received.", serviceName)
		resp = handleWorkflow(writer, request, deployment.GetWorkflow(), &proxyContext, dpConfig)
	default:
		resp = &requests.BufferedResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       fmt.Sprintf("Service '%s' has unknown deployment type.", serviceName),
		}
	}

	resp.E2ELatency = time.Since(requestMetadata.start)
	return resp

}

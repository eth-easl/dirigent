package registration_server

import (
	"bufio"
	"cluster_manager/internal/control_plane/control_plane"
	"cluster_manager/internal/control_plane/control_plane/autoscalers"
	"cluster_manager/internal/control_plane/workflow"
	dandelion_workflow "cluster_manager/internal/control_plane/workflow/dandelion-workflow"
	"cluster_manager/proto"
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/sirupsen/logrus"
)

func getFormValue(w http.ResponseWriter, r *http.Request, key string) (string, error) {
	val := r.FormValue(key)
	if len(val) == 0 {
		msg := fmt.Sprintf("Invalid function %s.", key)

		http.Error(w, msg, http.StatusBadRequest)
		return "", errors.New(msg)
	}

	return val, nil
}

func parsePrepullConfig(mode string) proto.PrepullMode {
	switch mode {
	case "all_sync":
		return proto.PrepullMode_ALL_SYNC
	case "all_async":
		return proto.PrepullMode_ALL_ASYNC
	case "one_sync":
		return proto.PrepullMode_ONE_SYNC
	case "one_async":
		return proto.PrepullMode_ONE_ASYNC
	default:
		return proto.PrepullMode_NONE
	}
}

func functionRegistrationHandler(cpApi *control_plane.CpApiServer) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed.", http.StatusMethodNotAllowed)
			return
		}

		err := r.ParseForm()
		if err != nil {
			http.Error(w, "Invalid parsing service registration request.", http.StatusBadRequest)
			return
		}

		var name string
		if name, err = getFormValue(w, r, "name"); err != nil {
			return
		}

		var image string
		if image, err = getFormValue(w, r, "image"); err != nil {
			return
		}

		portForwarding := r.Form["port_forwarding"]
		if len(portForwarding) == 0 {
			http.Error(w, "No port forwarding specified. Function cannot communicate with the outside world.", http.StatusBadRequest)
			return
		}
		guestPort, err := strconv.Atoi(portForwarding[0])
		if err != nil {
			http.Error(w, "Invalid guest port specified.", http.StatusBadRequest)
			return
		}

		portMapping := &proto.PortMapping{
			GuestPort: int32(guestPort),
		}

		switch portForwarding[1] {
		case "tcp":
			portMapping.Protocol = proto.L4Protocol_TCP
		case "udp":
			portMapping.Protocol = proto.L4Protocol_UDP
		default:
			http.Error(w, "Invalid port forwarding protocol.", http.StatusBadRequest)
			return
		}

		cpu := 1
		runtimeSpec, ok := r.Form["requested_cpu"]
		if ok {
			cpu, err = strconv.Atoi(runtimeSpec[0])
			if err != nil {
				http.Error(w, "Invalid CPU value", http.StatusBadRequest)
				return
			}
		}

		memory := 1
		memSpec, ok := r.Form["requested_memory"]
		if ok {
			memory, err = strconv.Atoi(memSpec[0])
			if err != nil {
				http.Error(w, "Invalid memory value", http.StatusBadRequest)
				return
			}
		}

		gpu := 0
		gpuSpec, ok := r.Form["requested_gpu"]
		if ok {
			gpu, err = strconv.Atoi(gpuSpec[0])
			if err != nil {
				http.Error(w, "Invalid GPU value", http.StatusBadRequest)
				return
			}
		}

		prepullConfig := proto.PrepullMode_NONE
		prepullMode, ok := r.Form["prepull_mode"]
		if ok {
			prepullConfig = parsePrepullConfig(prepullMode[0])
		}
		isAsync := prepullConfig != proto.PrepullMode_NONE

		autoscalingConfig := autoscalers.NewDefaultAutoscalingConfiguration()

		if len(r.FormValue("scaling_upper_bound")) != 0 {
			upperBound, err := strconv.Atoi(r.FormValue("scaling_upper_bound"))
			if err != nil {
				http.Error(w, "Invalid scaling upper bound.", http.StatusBadRequest)
				return
			}

			autoscalingConfig.ScalingUpperBound = int32(upperBound)
		}
		if len(r.FormValue("scaling_lower_bound")) != 0 {
			lowerBound, err := strconv.Atoi(r.FormValue("scaling_lower_bound"))
			if err != nil {
				http.Error(w, "Invalid scaling lower bound.", http.StatusBadRequest)
				return
			}

			autoscalingConfig.ScalingLowerBound = int32(lowerBound)
		}

		iterationMultiplier := 102
		iterMul, ok := r.Form["iteration_multiplier"]
		if ok {
			iterationMultiplier, err = strconv.Atoi(iterMul[0])
			if err != nil {
				http.Error(w, "Invalid iteration multiplier.", http.StatusBadRequest)
				return
			}
		}

		coldStartBusyLoopMs := 0
		busyLoopMs, ok := r.Form["cold_start_busy_loop_ms"]
		if ok {
			coldStartBusyLoopMs, err = strconv.Atoi(busyLoopMs[0])
			if err != nil {
				http.Error(w, "Invalid cold start busy loop duration.", http.StatusBadRequest)
				return
			}
		}

		numArgs := 0
		bodyNumArgs, ok := r.Form["num_args"]
		if ok {
			numArgs, err = strconv.Atoi(bodyNumArgs[0])
			if err != nil {
				http.Error(w, "Invalid number of function arguments.", http.StatusBadRequest)
				return
			}
		}

		numRets := 0
		bodyNumRets, ok := r.Form["num_rets"]
		if ok {
			numRets, err = strconv.Atoi(bodyNumRets[0])
			if err != nil {
				http.Error(w, "Invalid number of function returns.", http.StatusBadRequest)
				return
			}
		}

		environmentVars := r.Form["env_vars"]
		programArgs := r.Form["program_args"]

		serviceInfo := &proto.ServiceInfo{
			Name:              name,
			Image:             image,
			RequestedCpu:      uint64(cpu),
			RequestedMemory:   uint64(memory),
			RequestedGpu:      uint64(gpu),
			PortForwarding:    portMapping,
			AutoscalingConfig: autoscalingConfig,
			SandboxConfiguration: &proto.SandboxConfiguration{
				IterationMultiplier: int32(iterationMultiplier),
				ColdStartBusyLoopMs: int32(coldStartBusyLoopMs),
			},
			NumArgs:              uint32(numArgs),
			NumRets:              uint32(numRets),
			EnvironmentVariables: environmentVars,
			ProgramArguments:     programArgs,
		}

		var service *proto.ActionStatus
		if isAsync {
			go cpApi.RegisterService(context.Background(), serviceInfo)
		} else {
			service, err = cpApi.RegisterService(r.Context(), serviceInfo)
		}

		if err == nil && service.Success {
			w.WriteHeader(http.StatusOK)
		} else {
			http.Error(w, "Failed to register service.", http.StatusConflict)
			return
		}

		_, err = w.Write([]byte(GetLoadBalancerAddress(cpApi)))
		if err != nil {
			http.Error(w, "Error writing endpoints.", http.StatusInternalServerError)
			return
		}
	}
}

func isFunctionRegisteredHandler(cpApi *control_plane.CpApiServer) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		name, ok := r.URL.Query()["name"]
		if !ok {
			http.Error(w, "Querystring argument 'name' has not been specified.", http.StatusBadRequest)
			return
		}
		result, err := cpApi.HasService(r.Context(), &proto.ServiceIdentifier{Name: name[0]})
		if err != nil {
			http.Error(w, "Error checking for service.", http.StatusInternalServerError)
			return
		}
		if result.HasService {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}
}

func workflowRegistrationHandler(cpApi *control_plane.CpApiServer) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed.", http.StatusMethodNotAllowed)
			return
		}

		// Parse request
		err := r.ParseForm()
		if err != nil {
			http.Error(w, "Failed to parse workflow request.", http.StatusBadRequest)
			return
		}
		name := r.FormValue("name") // required
		if len(name) == 0 {
			logrus.Errorf("Invalid workflow request (empty workflow name).")
			http.Error(w, "Invalid workflow name.", http.StatusBadRequest)
			return
		}
		wfString := r.FormValue("workflow") // required
		if len(wfString) == 0 {
			logrus.Errorf("Invalid workflow request (empty workflow description).")
			http.Error(w, "Empty workflow description.", http.StatusBadRequest)
			return
		}
		partMethodString := r.FormValue("partitionMethod") // optional (otherwise use default)

		// Set partition method
		var partitionMethod dandelion_workflow.PartitionMethod
		if len(partMethodString) > 0 {
			partitionMethod = dandelion_workflow.PartitionMethodFromString(partMethodString)
			if partitionMethod == dandelion_workflow.Invalid {
				logrus.Errorf("Invalid partition method '%s' in request.", partMethodString)
				http.Error(w, fmt.Sprintf("Invalid partition method '%s'.", partMethodString), http.StatusBadRequest)
				return
			}
		} else {
			partitionMethod = dandelion_workflow.PartitionMethodFromString(cpApi.ControlPlane.Config.DefaultWFPartitionMethod)
			if partitionMethod == dandelion_workflow.Invalid {
				logrus.Errorf("Invalid default partition method '%s'", cpApi.ControlPlane.Config.DefaultWFPartitionMethod)
				http.Error(w, "Invalid default partition method.", http.StatusInternalServerError)
				return
			}
		}

		// Parse workflow
		dwfParser := dandelion_workflow.NewParser(bufio.NewReader(strings.NewReader(wfString)))
		dwf, err := dwfParser.Parse()
		if err != nil {
			logrus.Errorf("Failed to parse workflow '%s': %v", name, err)
			http.Error(w, fmt.Sprintf("Invalid workflow (failed to parse: %s).", err), http.StatusBadRequest)
			return
		}
		dwf.Name = name

		// Check that workflow functions are registered
		var serviceList *proto.ServiceList
		if cpApi.LeaderElectionServer.IsLeader() {
			serviceList, err = cpApi.ListServices(r.Context(), &emptypb.Empty{})
		} else {
			serviceList, err = cpApi.LeaderElectionServer.GetLeader().ListServices(r.Context(), &emptypb.Empty{})
		}
		if err != nil {
			logrus.Errorf("Failed to get registered services: %v", err)
			http.Error(w, fmt.Sprintf("Failed to get registered services."), http.StatusInternalServerError)
			return
		}
		if !dwf.CheckFunctionDeclarations(serviceList.Service) {
			logrus.Errorf("Not all workflow functions are registered.")
			http.Error(w, fmt.Sprintf("Not all workflow functions are registered."), http.StatusBadRequest)
			return
		}

		// Process workflow
		wfs, wfTasks, err := dwf.ExportWorkflow(partitionMethod)
		if err != nil {
			logrus.Errorf("Failed to process workflow '%s': %v", name, err)
			http.Error(w, fmt.Sprintf("Invalid workflow (failed to process: %s).", err), http.StatusBadRequest)
			return
		}

		// Register workflow tasks
		var status *proto.ActionStatus
		var errIdx int
		for wfIdx, wf := range wfs {
			var taskInfoBatch []*proto.WorkflowTaskInfo
			for _, task := range wfTasks[wfIdx] {
				taskInfoBatch = append(taskInfoBatch, &proto.WorkflowTaskInfo{
					Name:               task.Name,
					NumIn:              task.NumIn,
					NumOut:             task.NumOut,
					InputSharding:      workflow.ShardingToUint32(task.InputSharding),
					Functions:          task.Functions,
					FunctionInNum:      task.FunctionInNum,
					FunctionOutNum:     task.FunctionOutNum,
					FunctionDataFlow:   task.FunctionDataFlow,
					FunctionInSharding: workflow.ShardingToUint32(task.FunctionInSharding),
					ConsumerTasks:      workflow.TaskToStr(task.ConsumerTasks),
					ConsumerDataSrcIdx: task.ConsumerDataSrcIdx,
					ConsumerDataDstIdx: task.ConsumerDataDstIdx,
				})
			}

			wfInfo := &proto.WorkflowInfo{
				Name:              wf.Name,
				NumIn:             wf.NumIn,
				NumOut:            wf.NumOut,
				InitialTasks:      workflow.TaskToStr(wf.InitialTasks),
				InitialDataSrcIdx: wf.InitialDataSrcIdx,
				InitialDataDstIdx: wf.InitialDataDstIdx,
				OutTasks:          workflow.TaskToStr(wf.OutTasks),
				OutDataSrcIdx:     wf.OutDataSrcIdx,
				Tasks:             taskInfoBatch,
			}

			if cpApi.LeaderElectionServer.IsLeader() {
				status, err = cpApi.RegisterWorkflow(r.Context(), wfInfo)
			} else {
				status, err = cpApi.LeaderElectionServer.GetLeader().RegisterWorkflow(r.Context(), wfInfo)
			}

			if !status.Success || err != nil {
				errIdx = wfIdx
				break
			}
		}

		if errIdx > 0 {
			wfStr := ""
			for wfIdx, wf := range wfs[:errIdx] {
				if wfIdx != len(wfs)-1 {
					wfStr += fmt.Sprintf("'%s', ", wf.Name)
				} else {
					wfStr += fmt.Sprintf("'%s'", wf.Name)
				}
			}
			logrus.Infof("Successfully registered workflow(s) [%s].", wfStr)
		}

		if !status.Success || err != nil {
			logrus.Errorf("Failed to register workflow '%s': %v", wfs[errIdx].Name, err)
			http.Error(w, fmt.Sprintf("Failed to register workflow '%s' (%s).", wfs[errIdx].Name, err), http.StatusConflict)
			return
		}

		// Write successfully registered workflows into response
		w.WriteHeader(http.StatusOK)
		wfStr := ""
		for wfIdx, wf := range wfs {
			if wfIdx != len(wfs)-1 {
				wfStr += fmt.Sprintf("%s;", wf.Name)
			} else {
				wfStr += fmt.Sprintf("%s", wf.Name)
			}
		}
		_, err = w.Write([]byte(wfStr))
		if err != nil {
			logrus.Errorf("Failed to write workfows into response: %v", err)
		}
		logrus.Infof("Successfully registered workflow(s) [%s].", wfStr)
	}
}

func GetLoadBalancerAddress(cpApi *control_plane.CpApiServer) string {
	if cpApi.HAProxyAPI.GetLoadBalancerAddress() == "" {
		if len(cpApi.LeaderElectionServer.GetPeers()) > 0 {
			logrus.Fatal("Load balancer should always be used in high-availability mode.")
		}

		endpointList := ""
		cnt := 0

		cpApi.ControlPlane.DataPlaneConnections.RLock()
		defer cpApi.ControlPlane.DataPlaneConnections.RUnlock()

		for _, conn := range cpApi.ControlPlane.DataPlaneConnections.GetMap() {
			setDelimiter := cnt != cpApi.ControlPlane.DataPlaneConnections.Len()-1
			delimiter := ""

			if setDelimiter {
				delimiter = ";"
			}

			endpointList += fmt.Sprintf("%s:%s%s", conn.GetIP(), conn.GetProxyPort(), delimiter)
			cnt++
		}

		return endpointList
	} else {
		return cpApi.HAProxyAPI.GetLoadBalancerAddress()
	}
}

func healthHandler(writer http.ResponseWriter, _ *http.Request) {
	writer.WriteHeader(http.StatusOK)
}

func StartServiceRegistrationServer(cpApi *control_plane.CpApiServer, registrationPort string) chan struct{} {
	mux := http.NewServeMux()
	mux.HandleFunc("/", functionRegistrationHandler(cpApi))
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/patch", patchHandler(cpApi))
	mux.HandleFunc("/workflow", workflowRegistrationHandler(cpApi))
	mux.HandleFunc("/check", isFunctionRegisteredHandler(cpApi))

	server := &http.Server{
		Addr:    fmt.Sprintf(":%s", registrationPort),
		Handler: mux,
	}
	stopCh := make(chan struct{})

	//////////////////////////////
	// SERVER START
	//////////////////////////////
	go func() {
		attempt := 0

		for {
			time.Sleep(50 * time.Millisecond)
			logrus.Infof("Starting service registration service - attempt #%d", attempt+1)

			err := server.ListenAndServe()
			if !errors.Is(err, http.ErrServerClosed) {
				logrus.Errorf("Failed to start service registration server (attempt: #%d, error: %s)", attempt+1, err.Error())
				attempt++
			} else {
				logrus.Infof("Succesfully closed function registration server")
				return
			}
		}
	}()

	//////////////////////////////
	// SERVER CLOSE
	//////////////////////////////
	go func() {
		defer close(stopCh)

		<-stopCh
		if err := server.Close(); err != nil {
			logrus.Errorf("Failed to shut down function registration server.")
		}
	}()

	return stopCh
}

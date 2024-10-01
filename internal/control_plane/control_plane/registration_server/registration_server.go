package registration_server

import (
	"bufio"
	"cluster_manager/internal/control_plane/control_plane"
	"cluster_manager/internal/control_plane/control_plane/autoscalers"
	"cluster_manager/internal/control_plane/workflow"
	"cluster_manager/internal/control_plane/workflow/dandelion-workflow"
	"cluster_manager/proto"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

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

		name := r.FormValue("name")
		if len(name) == 0 {
			http.Error(w, "Invalid function name.", http.StatusBadRequest)
			return
		}
		image := r.FormValue("image")
		if len(image) == 0 {
			http.Error(w, "Invalid function image.", http.StatusBadRequest)
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

		var service *proto.ActionStatus
		if cpApi.LeaderElectionServer.IsLeader() {
			service, err = cpApi.RegisterService(r.Context(), &proto.ServiceInfo{
				Name:              name,
				Image:             image,
				PortForwarding:    portMapping,
				AutoscalingConfig: autoscalingConfig,
				RequestedCpu:      uint64(cpu),
				RequestedMemory:   uint64(memory),
				SandboxConfiguration: &proto.SandboxConfiguration{
					IterationMultiplier: int32(iterationMultiplier),
					ColdStartBusyLoopMs: int32(coldStartBusyLoopMs),
				},
			})
		} else {
			service, err = cpApi.LeaderElectionServer.GetLeader().RegisterService(r.Context(), &proto.ServiceInfo{
				Name:              name,
				Image:             image,
				PortForwarding:    portMapping,
				AutoscalingConfig: autoscalingConfig,
				RequestedCpu:      uint64(cpu),
				RequestedMemory:   uint64(memory),
				SandboxConfiguration: &proto.SandboxConfiguration{
					IterationMultiplier: int32(iterationMultiplier),
					ColdStartBusyLoopMs: int32(coldStartBusyLoopMs),
				},
			})
		}
		if err != nil {
			return
		}

		if service.Success {
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

		logrus.Debugf("Successfully registered function %s.", name)
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

		// Parse workflow
		dwfParser := dandelion_workflow.NewParser(bufio.NewReader(strings.NewReader(wfString)))
		dwf, err := dwfParser.Parse()
		if err != nil {
			logrus.Errorf("Failed to parse workflow '%s': %v", name, err)
			http.Error(w, fmt.Sprintf("Invalid workflow (failed to parse: %s).", err), http.StatusBadRequest)
			return
		}
		dwf.Name = name

		// Process workflow
		wfs, wfTasks, err := dwf.ExportWorkflow(dandelion_workflow.FullPartition)
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
					Functions:          task.Functions,
					FunctionInNum:      task.FunctionInNum,
					FunctionOutNum:     task.FunctionOutNum,
					FunctionDataFlow:   task.FunctionDataFlow,
					ConsumerTasks:      workflow.TaskToStr(task.ConsumerTasks),
					ConsumerDataSrcIdx: task.ConsumerDataSrcIdx,
					ConsumerDataDstIdx: task.ConsumerDataDstIdx,
				})
			}

			wfInfo := &proto.WorkflowInfo{
				Name:              wf.Name,
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

		w.WriteHeader(http.StatusOK)
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
			select {
			case <-time.After(50 * time.Millisecond):
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

		return
	}()

	return stopCh
}

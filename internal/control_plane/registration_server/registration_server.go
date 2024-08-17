package registration_server

import (
	"cluster_manager/api"
	"cluster_manager/api/proto"
	"cluster_manager/internal/control_plane/autoscaling"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
)

func registrationHandler(cpApi *api.CpApiServer) func(w http.ResponseWriter, r *http.Request) {
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

		var autoscalingConfig *proto.AutoscalingConfiguration
		coldStartSweep, ok := r.Form["cold_start_sweep"]
		if ok {
			parsed, err := strconv.ParseBool(coldStartSweep[0])
			if err != nil {
				http.Error(w, "Invalid cold start sweep value", http.StatusBadRequest)
				return
			}

			autoscalingConfig = autoscaling.NewDefaultAutoscalingMetadata(parsed)
			if parsed {
				logrus.Infof("Autoscaling configuration for %s set to cold start sweep mode.", name)
			}
		} else {
			autoscalingConfig = autoscaling.NewDefaultAutoscalingMetadata(false)
		}

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

		var service *proto.ActionStatus
		if cpApi.LeaderElectionServer.IsLeader() {
			service, err = cpApi.RegisterService(r.Context(), &proto.ServiceInfo{
				Name:              name,
				Image:             image,
				PortForwarding:    portMapping,
				AutoscalingConfig: autoscalingConfig,
				RequestedCpu:      uint64(cpu),
				RequestedMemory:   uint64(memory),
			})
		} else {
			service, err = cpApi.LeaderElectionServer.GetLeader().RegisterService(r.Context(), &proto.ServiceInfo{
				Name:              name,
				Image:             image,
				PortForwarding:    portMapping,
				AutoscalingConfig: autoscalingConfig,
				RequestedCpu:      uint64(cpu),
				RequestedMemory:   uint64(memory),
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

func GetLoadBalancerAddress(cpApi *api.CpApiServer) string {
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

func HealthHandler(writer http.ResponseWriter, _ *http.Request) {
	writer.WriteHeader(http.StatusOK)
}

func StartServiceRegistrationServer(cpApi *api.CpApiServer, registrationPort string) chan struct{} {
	mux := http.NewServeMux()
	mux.HandleFunc("/", registrationHandler(cpApi))
	mux.HandleFunc("/health", HealthHandler)

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
				if err != http.ErrServerClosed {
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

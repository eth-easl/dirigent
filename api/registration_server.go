package api

import (
	"cluster_manager/api/proto"
	"cluster_manager/internal/control_plane/autoscaling"
	"fmt"
	"net/http"
	"strconv"

	"github.com/sirupsen/logrus"
)

func StartServiceRegistrationServer(cpApi *CpApiServer, registrationPort string) {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed.", http.StatusMethodNotAllowed)
			return
		} else if cpApi.DataPlaneConnections == nil || len(cpApi.DataPlaneConnections) == 0 {
			http.Error(w, "No data plane found.", http.StatusPreconditionFailed)
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

		autoscalingConfig := autoscaling.NewDefaultAutoscalingMetadata()

		if len(r.FormValue("scaling_upper_bound")) != 0 {
			upperBound, err := strconv.Atoi(r.FormValue("scaling_upper_bound"))
			if err != nil {
				http.Error(w, "Invalid port forwarding protocol.", http.StatusBadRequest)
				return
			}

			autoscalingConfig.ScalingUpperBound = int32(upperBound)
		}
		if len(r.FormValue("scaling_lower_bound")) != 0 {
			lowerBound, err := strconv.Atoi(r.FormValue("scaling_lower_bound"))
			if err != nil {
				http.Error(w, "Invalid port forwarding protocol.", http.StatusBadRequest)
				return
			}

			autoscalingConfig.ScalingLowerBound = int32(lowerBound)
		}

		service, err := cpApi.RegisterService(r.Context(), &proto.ServiceInfo{
			Name:              name,
			Image:             image,
			PortForwarding:    portMapping,
			AutoscalingConfig: autoscalingConfig,
		})
		if err != nil {
			return
		}

		if service.Success {
			w.WriteHeader(http.StatusOK)
		} else {
			http.Error(w, "Failed to register service.", http.StatusConflict)
			return
		}

		endpointList := ""

		cnt := 0
		for _, conn := range cpApi.DataPlaneConnections {
			setDelimiter := cnt != len(cpApi.DataPlaneConnections)-1
			delimiter := ""

			if setDelimiter {
				delimiter = ";"
			}

			endpointList += fmt.Sprintf("%s:%s%s", conn.IP, conn.ProxyPort, delimiter)

			cnt++
		}

		_, err = w.Write([]byte(endpointList))
		if err != nil {
			http.Error(w, "Error writing endpoints.", http.StatusInternalServerError)
			return
		}
	})

	logrus.Info("Starting service registration service")

	err := http.ListenAndServe(fmt.Sprintf(":%s", registrationPort), nil)

	if err != http.ErrServerClosed {
		logrus.Fatal("Failed to start service registration server")
	}
}

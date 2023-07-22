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
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		} else if cpApi.DataPlaneConnections == nil || len(cpApi.DataPlaneConnections) == 0 {
			http.Error(w, "No data plane found", http.StatusPreconditionFailed)
			return
		}

		err := r.ParseForm()
		if err != nil {
			logrus.Error("Error parsing form")
		}

		name := r.FormValue("name")
		image := r.FormValue("image")

		portForwarding := r.Form["port_forwarding"]
		guestPort, err := strconv.Atoi(portForwarding[0])
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
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
			w.WriteHeader(http.StatusBadRequest)
		}

		service, err := cpApi.RegisterService(r.Context(), &proto.ServiceInfo{
			Name:              name,
			Image:             image,
			PortForwarding:    portMapping,
			AutoscalingConfig: autoscaling.NewDefaultAutoscalingMetadata(),
		})
		if err != nil {
			return
		}

		if service.Success {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusConflict)
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
			w.WriteHeader(http.StatusInternalServerError)
		}
	})

	logrus.Info("Starting service registration service")

	err := http.ListenAndServe(fmt.Sprintf(":%s", registrationPort), nil)

	if err != http.ErrServerClosed {
		logrus.Fatal("Failed to start service registration server")
	}
}

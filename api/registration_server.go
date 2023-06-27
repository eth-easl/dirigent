package api

import (
	"cluster_manager/api/proto"
	"cluster_manager/internal/control_plane"
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
		} else if cpApi.dpiInterface == nil {
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
			AutoscalingConfig: control_plane.NewDefaultAutoscalingMetadata(),
		})
		if err != nil {
			return
		}

		if service.Success {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusConflict)
		}

		w.Write([]byte(fmt.Sprintf("%s:%s", cpApi.dpiIP, cpApi.dpiProxyPort)))
	})

	logrus.Info("Starting service registration service")

	err := http.ListenAndServe(fmt.Sprintf(":%s", registrationPort), nil)

	if err != http.ErrServerClosed {
		logrus.Fatal("Failed to start service registration server")
	}
}

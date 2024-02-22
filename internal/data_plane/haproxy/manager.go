package haproxy

import (
	"fmt"
	client_native "github.com/haproxytech/client-native"
	"github.com/haproxytech/models"
	"github.com/sirupsen/logrus"
	"math/rand"
)

const (
	BackendName = "dirigent_data_planes"
)

type API struct {
	client *client_native.HAProxyClient
}

func NewHAProxyAPI() *API {
	config := getConfigClient()
	runtime := getRuntimeClient(config)

	return &API{
		client: getHAProxyClient(config, runtime),
	}
}

func (api *API) ClearTargetList() {
	version, servers, err := api.client.Configuration.GetServers(BackendName, "")
	if err != nil {
		logrus.Errorf("Failed to get list of servers for HAProxy backend %s", BackendName)
		return
	}

	for _, server := range servers {
		err = api.client.Configuration.DeleteServer(server.Name, BackendName, "", version)
		if err != nil {
			logrus.Errorf("Failed to delete server %s from the list of endpoints for HAProxy backend %s", server.Name, BackendName)
		}
	}
}

func (api *API) ListLBTargets() []string {
	_, servers, err := api.client.Configuration.GetServers(BackendName, "")
	if err != nil {
		fmt.Println(err.Error())
	}

	var result []string
	for _, s := range servers {
		result = append(result, s.Name)
	}

	return result
}

func (api *API) AddDataplane(ipAddress string, port int) {
	var p = new(int64)
	*p = int64(port)

	newServer := &models.Server{
		Name:    fmt.Sprintf("dp_%d", rand.Int()),
		Address: ipAddress,
		Port:    p,
		Check:   models.ServerCheckEnabled,
	}

	version, err := api.client.Configuration.GetVersion("")
	if err != nil {
		logrus.Error("Failed to add server to HAProxy backend. Error getting HAProxy configuration version.")
		return
	}

	err = api.client.Configuration.CreateServer(BackendName, newServer, "", version)
	if err != nil {
		logrus.Errorf("Error adding data plane with address %s:%d to the HAProxy load balancer.", ipAddress, port)
	}
}

func (api *API) RemoveDataplane(ipAddress string, port int) {

}

func (api *API) ReviseDataplanes(toKeep []string) {

}

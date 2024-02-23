package haproxy

import (
	"cluster_manager/pkg/grpc_helpers"
	_map "cluster_manager/pkg/map"
	"fmt"
	clientnative "github.com/haproxytech/client-native"
	"github.com/haproxytech/models"
	"github.com/sirupsen/logrus"
	"math/rand"
	"sync"
)

const (
	BackendName  = "dirigent_data_planes"
	ServerPrefix = "dp_"
)

type API struct {
	client *clientnative.HAProxyClient

	// addressTo - e.g., 10.0.1.2:8080 -> dataplane_123
	addressToName map[string]string
	mutex         sync.Mutex
}

func NewHAProxyAPI() *API {
	config := getConfigClient()
	runtime := getRuntimeClient(config)

	api := &API{
		client:        getHAProxyClient(config, runtime),
		addressToName: make(map[string]string),
	}

	names, addresses := api.ListDataplanes()
	for i := 0; i < len(names); i++ {
		// no need for locking as no one is using the API object
		api.addressToName[addresses[i]] = names[i]
	}

	return api
}

func (api *API) ListDataplanes() (names []string, addresses []string) {
	_, servers, err := api.client.Configuration.GetServers(BackendName, "")
	if err != nil {
		fmt.Println(err.Error())
	}

	for _, s := range servers {
		names = append(names, s.Name)
		addresses = append(addresses, fmt.Sprintf("%s:%d", s.Address, *s.Port))
	}

	return names, addresses
}

func (api *API) AddDataplane(ipAddress string, port int) {
	version, err := api.client.Configuration.GetVersion("")
	if err != nil {
		logrus.Errorf("Failed to add server to HAProxy backend. Error getting configuration version - %v", err)
		return
	}

	api.addServer(ipAddress, port, "", version)
}

func (api *API) RemoveDataplane(ipAddress string, port int) {
	version, err := api.client.Configuration.GetVersion("")
	if err != nil {
		logrus.Errorf("Failed to add server to HAProxy backend. Error getting configuration version - %v", err)
		return
	}

	api.removeServerByName(ipAddress, port, "", version)
}

func (api *API) addServer(ipAddress string, port int, transaction string, version int64) {
	name := fmt.Sprintf("%s%d", ServerPrefix, rand.Int())
	url := fmt.Sprintf("%s:%d", ipAddress, port)

	api.mutex.Lock()
	defer api.mutex.Unlock()

	if _, ok := api.addressToName[url]; ok {
		logrus.Errorf("A server with the same network address already exists.")
		return
	} else {
		api.persistServerMetadata(name, ipAddress, port, transaction, version)
		api.addressToName[url] = name
	}
}

func (api *API) persistServerMetadata(name string, ipAddress string, port int, transaction string, version int64) {
	var p = new(int64)
	*p = int64(port)

	newServer := &models.Server{
		Name:    name,
		Address: ipAddress,
		Port:    p,
		Check:   models.ServerCheckEnabled,
	}

	err := api.client.Configuration.CreateServer(BackendName, newServer, transaction, version)
	if err != nil {
		logrus.Errorf("Error adding data plane with address %s:%d to the load balancer - %v", ipAddress, port, err)
	}
}

func (api *API) removeServerByName(ipAddress string, port int, transaction string, version int64) {
	url := fmt.Sprintf("%s:%d", ipAddress, port)

	api.mutex.Lock()
	defer api.mutex.Unlock()

	if dataplaneName, ok := api.addressToName[url]; !ok {
		logrus.Errorf("Cannot delete requested server as it does not exist.")
		return
	} else {
		err := api.client.Configuration.DeleteServer(dataplaneName, BackendName, transaction, version)
		if err != nil {
			logrus.Errorf("Error removing server with name %s from the load balancer - %v", dataplaneName, err)
		}

		delete(api.addressToName, url)
	}
}

func (api *API) ReviseDataplanes(addressesToKeep []string) {
	version, err := api.client.Configuration.GetVersion("")
	if err != nil {
		logrus.Errorf("Failed to revise data planes. Error obtaining configuration version - %v", err)
		return
	}

	transaction, err := api.client.Configuration.StartTransaction(version)
	if err != nil {
		logrus.Errorf("Failed to revise data planes. Error starting HAProxy transaction - %v", err)
		return
	}

	version, servers, err := api.client.Configuration.GetServers(BackendName, transaction.ID)
	if err != nil {
		logrus.Errorf("Failed to revise data planes. Error obtaining list of existing servers - %v", err)
		return
	}

	var existingURLs []string
	existingAddresses := make(map[string]string)
	for _, server := range servers {
		url := fmt.Sprintf("%s:%d", server.Address, *server.Port)

		existingURLs = append(existingURLs, url)
		existingAddresses[url] = server.Name
	}

	toAdd := _map.Difference(addressesToKeep, existingURLs)
	toRemove := _map.Difference(existingURLs, addressesToKeep)

	for _, server := range toRemove {
		ipAddress, port := grpc_helpers.SplitAddress(server)
		api.removeServerByName(ipAddress, port, transaction.ID, 0)
	}

	for _, address := range toAdd {
		ipAddress, port := grpc_helpers.SplitAddress(address)
		api.addServer(ipAddress, port, transaction.ID, 0)
	}

	_, err = api.client.Configuration.CommitTransaction(transaction.ID)
	if err != nil {
		logrus.Errorf("Failed to revise data planes. Error while commiting transaction - %v", err)
		return
	}
}

func (api *API) DeleteAllDataplanes() {
	_, servers, err := api.client.Configuration.GetServers(BackendName, "")
	if err != nil {
		fmt.Println(err.Error())
	}

	for _, s := range servers {
		api.RemoveDataplane(s.Address, int(*s.Port))
	}
}

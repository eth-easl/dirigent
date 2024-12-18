/*
 * MIT License
 *
 * Copyright (c) 2024 EASL
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */

package haproxy

import (
	"cluster_manager/api/proto"
	"cluster_manager/internal/control_plane/leader_election"
	"cluster_manager/pkg/config"
	"cluster_manager/pkg/grpc_helpers"
	_map "cluster_manager/pkg/map"
	"context"
	"fmt"
	clientnative "github.com/haproxytech/client-native"
	"github.com/haproxytech/models"
	"github.com/sirupsen/logrus"
	"math/rand"
	"os/exec"
	"sync"
)

const (
	DataplaneBackend = "dirigent_data_planes"
	DataplanePrefix  = "dp_"

	RegistrationServerBackend = "dirigent_registration_server"
	RegistrationServerPrefix  = "rs_"

	ServerHealthCheckInterval = 2500
)

type ReviseDataplanesInLBCall func(func([]string) bool) bool

type API struct {
	client    *clientnative.HAProxyClient
	lbAddress string

	// addressTo - e.g., 10.0.1.2:8080 -> dataplane_123
	addressToName map[string]string
	mutex         sync.Mutex
}

func NewHAProxyAPI(loadBalancerAddress string) *API {
	api := &API{
		client:    getHAProxyClient(),
		lbAddress: loadBalancerAddress,

		addressToName: make(map[string]string),
	}

	names, addresses := api.ListDataplanes()
	for i := 0; i < len(names); i++ {
		// no need for locking as no one is using the API object
		api.addressToName[addresses[i]] = names[i]
	}

	return api
}

func (api *API) StartHAProxy() {
	go func() {
		err := exec.Command("sudo", "systemctl", "start", "haproxy").Run()
		if err != nil {
			logrus.Errorf("Error starting HAProxy - %v", err.Error())
		}
	}()
}

func (api *API) StopHAProxy() {
	go func() {
		err := exec.Command("sudo", "systemctl", "stop", "haproxy").Run()
		if err != nil {
			logrus.Errorf("Error stopping HAProxy - %v", err.Error())
		}
	}()
}

// RestartHAProxy Should be called to commit every action to the running instance of HAProxy
func (api *API) RestartHAProxy() {
	go func() {
		logrus.Info("Restarting HAProxy...")

		err := exec.Command("sudo", "systemctl", "restart", "haproxy").Run()
		if err != nil {
			api.forceResetHAProxy()
		}
	}()
}

func (api *API) forceResetHAProxy() {
	go func() {
		err := exec.Command("sudo", "systemctl", "reset-failed", "haproxy").Run()
		if err != nil {
			logrus.Errorf("Error while doing force reset of HAProxy - %v", err.Error())
		}
	}()
}

func (api *API) GetLoadBalancerAddress() string {
	return api.lbAddress
}

func (api *API) ListDataplanes() (names []string, addresses []string) {
	return api.listServers(DataplaneBackend)
}

func (api *API) ListRegistrationServers() (names []string, addresses []string) {
	return api.listServers(RegistrationServerBackend)
}

func (api *API) listServers(backend string) (names []string, addresses []string) {
	_, servers, err := api.client.Configuration.GetServers(backend, "")
	if err != nil {
		fmt.Println(err.Error())
	}

	for _, s := range servers {
		names = append(names, s.Name)
		addresses = append(addresses, fmt.Sprintf("%s:%d", s.Address, *s.Port))
	}

	return names, addresses
}

func (api *API) AddDataplane(ipAddress string, port int, restart bool) {
	version, err := api.client.Configuration.GetVersion("")
	if err != nil {
		logrus.Errorf("Failed to add server to HAProxy backend. Error getting configuration version - %v", err)
		return
	}

	api.addServer(ipAddress, port, "", version)

	if restart {
		api.RestartHAProxy()
	}
}

func (api *API) RemoveDataplane(ipAddress string, port int, restart bool) {
	version, err := api.client.Configuration.GetVersion("")
	if err != nil {
		logrus.Errorf("Failed to add server to HAProxy backend. Error getting configuration version - %v", err)
		return
	}

	api.removeServerByName(ipAddress, port, "", version)

	if restart {
		api.RestartHAProxy()
	}
}

func (api *API) addServer(ipAddress string, port int, transaction string, version int64) {
	name := fmt.Sprintf("%s%d", DataplanePrefix, rand.Int())
	url := fmt.Sprintf("%s:%d", ipAddress, port)

	api.mutex.Lock()
	defer api.mutex.Unlock()

	if _, ok := api.addressToName[url]; ok {
		return
	} else {
		api.persistServerMetadata(name, ipAddress, port, transaction, version)
		api.addressToName[url] = name
		logrus.Infof("Added data plane with address %s:%d to HAProxy backend.", ipAddress, port)
	}
}

func (api *API) addRegistrationServer(ipAddress string, port int, transaction string, version int64) {
	name := fmt.Sprintf("%s%d", RegistrationServerPrefix, rand.Int())

	newServer := &models.Server{
		Name:    name,
		Address: ipAddress,
		Port:    Int64Ptr(port),
		Check:   models.ServerCheckEnabled,
		Inter:   Int64Ptr(ServerHealthCheckInterval), // ms; default fall: 3, default rise: 2
	}

	err := api.client.Configuration.CreateServer(RegistrationServerBackend, newServer, transaction, version)
	if err != nil {
		logrus.Errorf("Error adding registration server with address %s:%d to the load balancer - %v", ipAddress, port, err)
	}
}

func Int64Ptr(val int) *int64 {
	var p = new(int64)
	*p = int64(val)

	return p
}

func (api *API) persistServerMetadata(name string, ipAddress string, port int, transaction string, version int64) {
	newServer := &models.Server{
		Name:    name,
		Address: ipAddress,
		Port:    Int64Ptr(port),
		Check:   models.ServerCheckEnabled,
		Inter:   Int64Ptr(ServerHealthCheckInterval), // ms; default fall: 3, default rise: 2
	}

	err := api.client.Configuration.CreateServer(DataplaneBackend, newServer, transaction, version)
	if err != nil {
		logrus.Errorf("Error adding data plane with address %s:%d to the load balancer - %v", ipAddress, port, err)
	}
}

func (api *API) removeServerByName(ipAddress string, port int, transaction string, version int64) {
	url := fmt.Sprintf("%s:%d", ipAddress, port)

	api.mutex.Lock()
	defer api.mutex.Unlock()

	if dataplaneName, ok := api.addressToName[url]; !ok {
		return
	} else {
		err := api.client.Configuration.DeleteServer(dataplaneName, DataplaneBackend, transaction, version)
		if err != nil {
			logrus.Errorf("Error removing server with name %s from the load balancer - %v", dataplaneName, err)
		}

		delete(api.addressToName, url)
		logrus.Infof("Removed data plane with address %s:%d to HAProxy backend.", ipAddress, port)
	}
}

func (api *API) removeRegistrationServers(transaction string) {
	_, servers, err := api.client.Configuration.GetServers(RegistrationServerBackend, transaction)
	if err != nil {
		logrus.Errorf("Error listing all registration server backends - %v", err)
		return
	}

	for _, s := range servers {
		err = api.client.Configuration.DeleteServer(s.Name, RegistrationServerBackend, transaction, 0)
		if err != nil {
			logrus.Errorf("Error removing server %s - %v", s.Name, err)
		}
	}
}

func (api *API) genericRevise(backend string, addressesToKeep []string) (success bool, anyChanges bool) {
	errorIn := ""

	switch backend {
	case DataplaneBackend:
		errorIn = "data planes"
	case RegistrationServerBackend:
		errorIn = "registration servers"
	default:
		logrus.Fatalf("Invalid backend name for HAProxy backend revision function.")
	}

	version, err := api.client.Configuration.GetVersion("")
	if err != nil {
		logrus.Errorf("Failed to revise %s. Error obtaining configuration version - %v", errorIn, err)
		return false, false
	}

	transaction, err := api.client.Configuration.StartTransaction(version)
	if err != nil {
		logrus.Errorf("Failed to revise %s. Error starting HAProxy transaction - %v", errorIn, err)
		return false, false
	}

	version, servers, err := api.client.Configuration.GetServers(backend, transaction.ID)
	if err != nil {
		logrus.Errorf("Failed to revise %s. Error obtaining list of existing servers - %v", errorIn, err)
		return false, false
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
	anyChanges = (len(toAdd) != 0) || (len(toRemove) != 0)

	// REMOVAL
	switch backend {
	case DataplaneBackend:
		for _, server := range toRemove {
			ipAddress, port := grpc_helpers.SplitAddress(server)
			api.removeServerByName(ipAddress, port, transaction.ID, 0)
		}
	case RegistrationServerBackend:
		api.removeRegistrationServers(transaction.ID)
	}

	// ADDITION
	switch backend {
	case DataplaneBackend:
		for _, address := range toAdd {
			ipAddress, port := grpc_helpers.SplitAddress(address)
			api.addServer(ipAddress, port, transaction.ID, 0)
		}
	case RegistrationServerBackend:
		for _, address := range addressesToKeep {
			ipAddress, port := grpc_helpers.SplitAddress(address)
			api.addRegistrationServer(ipAddress, port, transaction.ID, 0)
		}
	}

	_, err = api.client.Configuration.CommitTransaction(transaction.ID)
	if err != nil {
		logrus.Errorf("Failed to revise %s. Error while commiting transaction - %v", errorIn, err)
		return false, false
	}

	return true, anyChanges
}

// ReviseRegistrationServers Need to call restart after this call
func (api *API) ReviseRegistrationServers(addressesToKeep []string) bool {
	success, changes := false, false

	for !success {
		api.client = getHAProxyClient()
		success, changes = api.genericRevise(RegistrationServerBackend, addressesToKeep)
	}

	return changes
}

// ReviseDataplanes Need to call restart after this call
func (api *API) ReviseDataplanes(addressesToKeep []string) bool {
	success, changes := false, false

	for !success {
		api.client = getHAProxyClient()
		success, changes = api.genericRevise(DataplaneBackend, addressesToKeep)
	}

	return changes
}

func (api *API) DeleteAllDataplanes() {
	_, servers, err := api.client.Configuration.GetServers(DataplaneBackend, "")
	if err != nil {
		fmt.Println(err.Error())
	}

	for _, s := range servers {
		api.RemoveDataplane(s.Address, int(*s.Port), false)
	}

	api.RestartHAProxy()
}

func (api *API) ReviseHAProxyConfiguration(args *proto.HAProxyConfig) (*proto.ActionStatus, error) {
	dpChanges := api.ReviseDataplanes(args.Dataplanes)
	rsChanges := api.ReviseRegistrationServers(args.RegistrationServers)

	toRestart := dpChanges || rsChanges
	if toRestart {
		api.RestartHAProxy()
		logrus.Info("HAProxy configuration revision done with restart!")
	} else {
		logrus.Info("HAProxy configuration revision done without restart!")
	}

	return &proto.ActionStatus{Success: true}, nil
}

func DisseminateHAProxyConfig(config *proto.HAProxyConfig, les *leader_election.LeaderElectionServer) {
	for peer, conn := range les.GetPeers() {
		_, err := conn.ReviseHAProxyConfiguration(context.Background(), config)
		if err != nil {
			logrus.Errorf("Failed disseminating HAProxy configuration to peer #%d", peer)
		}
	}
}

func (api *API) HAProxyReconstructionCallback(config *config.ControlPlaneConfig, reviseDataplanes ReviseDataplanesInLBCall) {
	dpChanges := reviseDataplanes(api.ReviseDataplanes)
	rsChanges := api.ReviseRegistrationServers(append(
		[]string{config.RegistrationServer},
		config.RegistrationServerReplicas...),
	)

	toRestart := dpChanges || rsChanges
	if toRestart {
		api.RestartHAProxy()
		logrus.Info("HAProxy configuration revision done with restart!")
	} else {
		logrus.Info("HAProxy configuration revision done without restart!")
	}
}

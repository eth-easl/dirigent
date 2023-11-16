package firecracker

import (
	"cluster_manager/internal/worker_node/managers"
	"fmt"
	"github.com/sirupsen/logrus"
	"net"
	"os/exec"
	"strings"
	"sync"
)

const (
	vethPrefix = "fc-"
	vethSuffix = "-veth"
)

type NetworkConfig struct {
	NetNS string

	TapDeviceName string
	TapMAC        string
	TapExternalIP string
	TapInternalIP string

	VETHHostName   string
	VETHInternalIP string
	VETHExternalIP string

	ExposedIP string
}

type NetworkPoolManager struct {
	sync.Mutex

	InternalIPManager *managers.IPManager
	ExposedIPManager  *managers.IPManager

	InterfaceNameGenerator *managers.VETHNameGenerator
	NetNSNameGenerator     *managers.ThreadSafeRandomGenerator

	MaxNetworkPoolSize int

	pool []*NetworkConfig
}

func NewNetworkPoolManager(internalPrefix, externalPrefix string, networkPoolSize int) *NetworkPoolManager {
	internalIPManager := managers.NewIPManager(internalPrefix)
	externalIPManager := managers.NewIPManager(externalPrefix)

	// NetworkPoolSize should be divisible by 16
	networkPoolSize -= (networkPoolSize % 16)
	if networkPoolSize < 16 {
		networkPoolSize = 16
	}

	pool := &NetworkPoolManager{
		InternalIPManager: internalIPManager,
		ExposedIPManager:  externalIPManager,

		InterfaceNameGenerator: managers.NewVETHNameGenerator(),
		NetNSNameGenerator:     managers.NewThreadSafeRandomGenerator(),

		MaxNetworkPoolSize: networkPoolSize,
	}

	pool.populate()

	return pool
}

func (np *NetworkPoolManager) populate() {
	np.Lock()
	defer np.Unlock()

	for len(np.pool) < np.MaxNetworkPoolSize {
		config, err := np.createNetwork()
		if err != nil {
			logrus.Errorf("Error creating a network - %v", err)
		}

		np.pool = append(np.pool, config)
	}
}

func (np *NetworkPoolManager) GetOneConfig() *NetworkConfig {
	np.Lock()
	defer np.Unlock()

	if len(np.pool) == 0 {
		config, err := np.createNetwork()
		if err != nil {
			logrus.Error("Error creating network on the critical path.")
			return nil
		}

		logrus.Warn("Trying to repopulating the network pool as it's empty.")
		// asynchronous repopulation
		go np.populate()

		return config
	} else if len(np.pool) == np.MaxNetworkPoolSize/4 {
		// asynchronous repopulation
		go np.populate()
	}

	cfg := np.pool[0]
	np.pool = np.pool[1:]

	return cfg
}

func (np *NetworkPoolManager) createNetwork() (*NetworkConfig, error) {
	// Networking for clones -- guide:
	// https://github.com/firecracker-microvm/firecracker/blob/main/docs/snapshotting/network-for-clones.md

	// Hardcode each IP address. This is possible as they all reside in different network namespaces.
	gatewayIP, vmip, mac, dev := "169.254.0.1", "169.254.0.2", "02:FC:00:00:00:00", "fc-tap"

	config := &NetworkConfig{
		NetNS: fmt.Sprintf("firecracker-%d", np.NetNSNameGenerator.Int()),

		TapDeviceName: dev,
		TapMAC:        mac,
		TapExternalIP: gatewayIP,
		TapInternalIP: vmip,
	}

	err := exec.Command("sudo", "ip", "netns", "add", config.NetNS).Run()
	if err != nil {
		logrus.Errorf("failed to create network namespace %s - %v", config.NetNS, err)
		return nil, err
	}

	err = exec.Command("sudo", "ip", "netns", "exec", config.NetNS, "ip", "tuntap", "add", "dev", dev, "mode", "tap").Run()
	if err != nil {
		logrus.Errorf("failed to create a TUN/TAP device %s - %v", dev, err)
		deleteNetworkNamespaceByName(config.NetNS)
		return nil, err
	}

	err = exec.Command("sudo", "ip", "netns", "exec", config.NetNS, "ip", "addr", "add", gatewayIP+"/30", "dev", dev).Run()
	if err != nil {
		logrus.Errorf("failed to assign IP address to device %s - %v", dev, err)
		deleteNetworkNamespaceByName(config.NetNS)
		return nil, err
	}

	err = exec.Command("sudo", "ip", "netns", "exec", config.NetNS, "ip", "link", "set", "dev", dev, "up").Run()
	if err != nil {
		logrus.Errorf("failed to bring up the device %s - %v", dev, err)
		deleteNetworkNamespaceByName(config.NetNS)
		return nil, err
	}

	exposedIP, vethInternalIP, vethExternalIP, err := np.createVETHPair(config, vmip)
	if err != nil {
		logrus.Errorf("failed to create and configure veth pair - %v", err)
		// network removal done inside the called function
		return nil, err
	}

	config.VETHInternalIP = vethInternalIP
	config.VETHExternalIP = vethExternalIP
	config.ExposedIP = exposedIP

	return config, nil
}

func (np *NetworkPoolManager) createVETHPair(config *NetworkConfig, guestTAP string) (string, string, string, error) {
	id := np.InterfaceNameGenerator.Generate()

	guestSidePairName := fmt.Sprintf("%s%d%s", vethPrefix, id, vethSuffix)
	hostSidePairName := fmt.Sprintf("%s%d%s", vethPrefix, id+1, vethSuffix)

	config.VETHHostName = hostSidePairName

	hostSideIpAddress, workloadIpAddress := np.InternalIPManager.GenerateIPMACPair()
	hostSideIpAddressWithMask := hostSideIpAddress + "/24"
	workloadIpAddressWithMask := workloadIpAddress + "/24"

	exposedIP := np.ExposedIPManager.GenerateSingleIP()

	//hostSideIpAddress := "10.0.0.1"
	//workloadIpAddress := "10.0.0.2"
	//guestTAP := "169.254.0.2"
	//exposedIP := "192.168.0.3"

	// create a vETH pair - both created in the netNS namespace
	err := exec.Command("sudo", "ip", "netns", "exec", config.NetNS, "ip", "link", "add", hostSidePairName, "type", "veth", "peer", "name", guestSidePairName).Run()
	if err != nil {
		logrus.Error("Error creating a virtual Ethernet pair - ", err)
		deleteNetworkNamespaceByName(config.NetNS)
		return exposedIP, hostSideIpAddress, workloadIpAddress, err
	}

	// move the host-side of te pair to the root namespace
	err = exec.Command("sudo", "ip", "netns", "exec", config.NetNS, "ip", "link", "set", hostSidePairName, "netns", "1").Run()
	if err != nil {
		logrus.Error("Error moving host-side of the pair to the root namespace - ", err)
		deleteNetworkNamespaceByName(config.NetNS)
		return exposedIP, hostSideIpAddress, workloadIpAddress, err
	}

	// assign an IP address to the guest-side pair
	err = exec.Command("sudo", "ip", "netns", "exec", config.NetNS, "ip", "addr", "add", workloadIpAddressWithMask, "dev", guestSidePairName).Run()
	if err != nil {
		logrus.Error("Error assigning an IP address to the guest-side of the pair - ", err)
		deleteNetworkNamespaceByName(config.NetNS)
		return exposedIP, hostSideIpAddress, workloadIpAddress, err
	}

	// bringing the guest-side pair up
	err = exec.Command("sudo", "ip", "netns", "exec", config.NetNS, "ip", "link", "set", "dev", guestSidePairName, "up").Run()
	if err != nil {
		logrus.Error("Error bringing the guest-end of the pair up - ", err)
		deleteNetworkNamespaceByName(config.NetNS)
		return exposedIP, hostSideIpAddress, workloadIpAddress, err
	}

	// assigning an IP address to the host-side of the pair
	err = exec.Command("sudo", "ip", "addr", "add", hostSideIpAddressWithMask, "dev", hostSidePairName).Run()
	if err != nil {
		logrus.Error("Error assigning an IP address to the host-end of the pair - ", err)
		deleteNetworkNamespaceByName(config.NetNS)
		deleteDeviceByName(hostSidePairName)
		return exposedIP, hostSideIpAddress, workloadIpAddress, err
	}

	// bringing the host-side pair up
	err = exec.Command("sudo", "ip", "link", "set", "dev", hostSidePairName, "up").Run()
	if err != nil {
		logrus.Error("Error bringing the host-end of the pair up - ", err)
		deleteNetworkNamespaceByName(config.NetNS)
		deleteDeviceByName(hostSidePairName)
		return exposedIP, hostSideIpAddress, workloadIpAddress, err
	}

	// assigning default gateway of the namespace
	err = exec.Command("sudo", "ip", "netns", "exec", config.NetNS, "ip", "route", "add", "default", "via", hostSideIpAddress).Run()
	if err != nil {
		logrus.Errorf("Error assigning default gateway of the namespace %s - %v", config.NetNS, err)
		deleteNetworkNamespaceByName(config.NetNS)
		deleteDeviceByName(hostSidePairName)
		return exposedIP, hostSideIpAddress, workloadIpAddress, err
	}

	// NAT for outgoing traffic from the namespace
	err = exec.Command("sudo", "ip", "netns", "exec", config.NetNS, "iptables", "-t", "nat", "-A", "POSTROUTING", "-o", guestSidePairName, "-s", guestTAP, "-j", "SNAT", "--to", exposedIP).Run()
	if err != nil {
		logrus.Errorf("Error adding a NAT rule for the outgoing traffic from the namespace - %v", err)
		deleteNetworkNamespaceByName(config.NetNS)
		deleteDeviceByName(hostSidePairName)
		return exposedIP, hostSideIpAddress, workloadIpAddress, err
	}

	// NAT for incoming traffic to the namespace
	err = exec.Command("sudo", "ip", "netns", "exec", config.NetNS, "iptables", "-t", "nat", "-A", "PREROUTING", "-i", guestSidePairName, "-d", exposedIP, "-j", "DNAT", "--to", guestTAP).Run()
	if err != nil {
		logrus.Errorf("Error adding a NAT rule for the incoming traffic to the namespace - %v", err)
		deleteNetworkNamespaceByName(config.NetNS)
		deleteDeviceByName(hostSidePairName)
		return exposedIP, hostSideIpAddress, workloadIpAddress, err
	}

	// adding route to the workload
	err = exec.Command("sudo", "ip", "route", "add", exposedIP, "via", workloadIpAddress).Run()
	if err != nil {
		logrus.Errorf("Error adding route to the workload - %v", err)
		deleteNetworkNamespaceByName(config.NetNS)
		deleteDeviceByName(hostSidePairName)
		return exposedIP, hostSideIpAddress, workloadIpAddress, err
	}

	return exposedIP, hostSideIpAddress, workloadIpAddress, err
}

func DeleteUnusedNetworkDevices() error {
	devices, err := net.Interfaces()
	if err != nil {
		logrus.Errorf("Error listing network devices - %v.", err)
		return err
	}

	for _, dev := range devices {
		if !(strings.HasPrefix(dev.Name, vethPrefix) && strings.HasSuffix(dev.Name, vethSuffix)) {
			continue
		}

		deleteDeviceByName(dev.Name)
	}

	err = exec.Command("sudo", "ip", "-all", "netns", "delete").Run()
	if err != nil {
		logrus.Errorf("Error deleting network network namespaces - %v.", err)
	}

	return err
}

func deleteDeviceByName(name string) {
	err := exec.Command("sudo", "ip", "link", "del", name).Run()
	if err != nil {
		logrus.Errorf("Failed to delete network device %s - %v.", name, err)
	}
}

func deleteNetworkNamespaceByName(name string) {
	err := exec.Command("sudo", "ip", "netns", "delete", name).Run()
	if err != nil {
		logrus.Errorf("Failed to delete namespace %s - %v.", name, err)
	}
}

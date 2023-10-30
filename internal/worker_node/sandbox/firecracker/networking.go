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
	tapDevicePrefix = "fc-"
	tapDeviceSuffix = "-tap"
	vethPrefix      = "fc-"
	vethSuffix      = "-veth"

	MaximumPoolSize = 32
)

type NetworkConfig struct {
	NetNS string

	TapDeviceName string
	TapMAC        string
	TapExternalIP string
	TapInternalIP string

	VETHInternalIP string
	VETHExternalIP string

	ExposedIP string
}

type NetworkManager struct {
	InternalIPManager *managers.IPManager
	ExposedIPManager  *managers.IPManager

	InterfaceNameGenerator *managers.VETHNameGenerator
	NetNSNameGenerator     *managers.ThreadSafeRandomGenerator

	PoolMutex       *sync.Mutex
	CurrentPoolSize int
	MaxPoolSize     int
}

func NewNetworkManager(internalPrefix, externalPrefix string) *NetworkManager {
	return &NetworkManager{
		InternalIPManager: managers.NewIPManager(internalPrefix),
		ExposedIPManager:  managers.NewIPManager(externalPrefix),

		InterfaceNameGenerator: managers.NewVETHNameGenerator(),
		NetNSNameGenerator:     managers.NewThreadSafeRandomGenerator(),

		PoolMutex:       &sync.Mutex{},
		CurrentPoolSize: 0,
		MaxPoolSize:     MaximumPoolSize,
	}
}

func (nm *NetworkManager) GetTAPDevice() {
	nm.PoolMutex.Lock()
	//if
	nm.PoolMutex.Unlock()
}

func (nm *NetworkManager) CreateTAPDevice() (*NetworkConfig, error) {
	// Networking for clones -- guide:
	// https://github.com/firecracker-microvm/firecracker/blob/main/docs/snapshotting/network-for-clones.md

	// Hardcode each IP address. This is possible as they all reside in different network namespaces.
	gatewayIP, vmip, mac, dev := "169.254.0.1", "169.254.0.2", "02:FC:00:00:00:00", "fc-tap"

	config := &NetworkConfig{
		NetNS: fmt.Sprintf("firecracker-%d", nm.NetNSNameGenerator.Int()),

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
		return nil, err
	}

	err = exec.Command("sudo", "sysctl", "-w net.ipv4.conf."+dev+".proxy_arp=1").Run()
	if err != nil {
		logrus.Errorf("failed to set proxy_arp for device %s - %v", dev, err)
		return nil, err
	}

	err = exec.Command("sudo", "sysctl", "-w net.ipv6.conf."+dev+".disable_ipv6=1").Run()
	if err != nil {
		logrus.Errorf("failed to disable IPv6 for device %s - %v", dev, err)
		return nil, err
	}

	err = exec.Command("sudo", "ip", "netns", "exec", config.NetNS, "ip", "addr", "add", gatewayIP+"/30", "dev", dev).Run()
	if err != nil {
		logrus.Errorf("failed to assign IP address to device %s - %v", dev, err)
		return nil, err
	}

	err = exec.Command("sudo", "ip", "netns", "exec", config.NetNS, "ip", "link", "set", "dev", dev, "up").Run()
	if err != nil {
		logrus.Errorf("failed to bring up the device %s - %v", dev, err)
		return nil, err
	}

	exposedIP, vethInternalIP, vethExternalIP, err := nm.createVETHPair(config, vmip)
	if err != nil {
		logrus.Errorf("failed to create and configure veth pair - %v", err)
		return nil, err
	}

	config.VETHInternalIP = vethInternalIP
	config.VETHExternalIP = vethExternalIP
	config.ExposedIP = exposedIP

	return config, nil
}

func (nm *NetworkManager) createVETHPair(config *NetworkConfig, guestTAP string) (string, string, string, error) {
	id := nm.InterfaceNameGenerator.Generate()

	guestSidePairName := fmt.Sprintf("%s%d%s", vethPrefix, id, vethSuffix)
	hostSidePairName := fmt.Sprintf("%s%d%s", vethPrefix, id+1, vethSuffix)

	hostSideIpAddress, workloadIpAddress := nm.InternalIPManager.GenerateIPMACPair()
	hostSideIpAddressWithMask := hostSideIpAddress + "/24"
	workloadIpAddressWithMask := workloadIpAddress + "/24"

	logrus.Debugf("Host-side IP address: %s", hostSideIpAddress)
	logrus.Debugf("Guest-side IP address: %s", workloadIpAddress)

	exposedIP := nm.ExposedIPManager.GenerateSingleIP()
	logrus.Debugf("Exposed IP address: %s", exposedIP)

	//hostSideIpAddress := "10.0.0.1"
	//workloadIpAddress := "10.0.0.2"
	//guestTAP := "169.254.0.2"
	//exposedIP := "192.168.0.3"

	// create a vETH pair - both created in the netNS namespace
	err := exec.Command("sudo", "ip", "netns", "exec", config.NetNS, "ip", "link", "add", hostSidePairName, "type", "veth", "peer", "name", guestSidePairName).Run()
	if err != nil {
		logrus.Error("Error creating a virtual Ethernet pair - ", err)
		return exposedIP, hostSideIpAddress, workloadIpAddress, err
	}

	// move the host-side of te pair to the root namespace
	err = exec.Command("sudo", "ip", "netns", "exec", config.NetNS, "ip", "link", "set", hostSidePairName, "netns", "1").Run()
	if err != nil {
		logrus.Error("Error moving host-side of the pair to the root namespace - ", err)
		return exposedIP, hostSideIpAddress, workloadIpAddress, err
	}

	// assign an IP address to the guest-side pair
	err = exec.Command("sudo", "ip", "netns", "exec", config.NetNS, "ip", "addr", "add", workloadIpAddressWithMask, "dev", guestSidePairName).Run()
	if err != nil {
		logrus.Error("Error assigning an IP address to the guest-side of the pair - ", err)
		return exposedIP, hostSideIpAddress, workloadIpAddress, err
	}

	// bringing the guest-side pair up
	err = exec.Command("sudo", "ip", "netns", "exec", config.NetNS, "ip", "link", "set", "dev", guestSidePairName, "up").Run()
	if err != nil {
		logrus.Error("Error bringing the guest-end of the pair up - ", err)
		return exposedIP, hostSideIpAddress, workloadIpAddress, err
	}

	// assigning an IP address to the host-side of the pair
	err = exec.Command("sudo", "ip", "addr", "add", hostSideIpAddressWithMask, "dev", hostSidePairName).Run()
	if err != nil {
		logrus.Error("Error assigning an IP address to the host-end of the pair - ", err)
		return exposedIP, hostSideIpAddress, workloadIpAddress, err
	}

	// bringing the host-side pair up
	err = exec.Command("sudo", "ip", "link", "set", "dev", hostSidePairName, "up").Run()
	if err != nil {
		logrus.Error("Error bringing the host-end of the pair up - ", err)
		return exposedIP, hostSideIpAddress, workloadIpAddress, err
	}

	// assigning default gateway of the namespace
	err = exec.Command("sudo", "ip", "netns", "exec", config.NetNS, "ip", "route", "add", "default", "via", hostSideIpAddress).Run()
	if err != nil {
		logrus.Error("Error assigning default gateway of the namespace - ", config.NetNS)
		return exposedIP, hostSideIpAddress, workloadIpAddress, err
	}

	// NAT for outgoing traffic from the namespace
	err = exec.Command("sudo", "ip", "netns", "exec", config.NetNS, "iptables", "-t", "nat", "-A", "POSTROUTING", "-o", guestSidePairName, "-s", guestTAP, "-j", "SNAT", "--to", exposedIP).Run()
	if err != nil {
		logrus.Error("Error adding a NAT rule for the outgoing traffic from the namespace - ", err)
		return exposedIP, hostSideIpAddress, workloadIpAddress, err
	}

	// NAT for incoming traffic to the namespace
	err = exec.Command("sudo", "ip", "netns", "exec", config.NetNS, "iptables", "-t", "nat", "-A", "PREROUTING", "-i", guestSidePairName, "-d", exposedIP, "-j", "DNAT", "--to", guestTAP).Run()
	if err != nil {
		logrus.Error("Error adding a NAT rule for the incoming traffic to the namespace - ", err)
		return exposedIP, hostSideIpAddress, workloadIpAddress, err
	}

	// adding route to the workload
	err = exec.Command("sudo", "ip", "route", "add", exposedIP, "via", workloadIpAddress).Run()
	if err != nil {
		logrus.Error("Error adding route to the workload - ", err)
		return exposedIP, hostSideIpAddress, workloadIpAddress, err
	}

	return exposedIP, hostSideIpAddress, workloadIpAddress, err
}

func DeleteUnusedNetworkDevices() error {
	devices, err := net.Interfaces()
	if err != nil {
		logrus.Error("Error listing network devices.")
		return err
	}

	for _, dev := range devices {
		if !((strings.HasPrefix(dev.Name, tapDevicePrefix) && strings.HasSuffix(dev.Name, tapDeviceSuffix)) ||
			(strings.HasPrefix(dev.Name, vethPrefix) && strings.HasSuffix(dev.Name, vethSuffix))) {
			continue
		}

		err = exec.Command("sudo", "ip", "link", "del", dev.Name).Run()
		if err != nil {
			logrus.Error("Error deleting network device '", dev.Name, "'.")
			return err
		}
	}

	err = exec.Command("sudo", "ip", "-all", "netns", "delete").Run()
	if err != nil {
		logrus.Error("Error deleting network network namespaces.")
		return err
	}

	return err
}

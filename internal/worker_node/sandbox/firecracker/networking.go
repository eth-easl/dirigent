package firecracker

import (
	"cluster_manager/internal/worker_node/managers"
	"fmt"
	"github.com/sirupsen/logrus"
	"math/rand"
	"net"
	"os/exec"
	"strings"
)

const (
	tapDevicePrefix = "fc-"
	tapDeviceSuffix = "-tap"
	vethPrefix      = "fc-"
	vethSuffix      = "-veth"
)

type TAPLink struct {
	Device    string
	MAC       string
	GatewayIP string
	VmIP      string
	ExposedIP string
}

func createTAPDevice(internalIPManager *IPManager, exposedIPManager *IPManager,
	vmcs *VMControlStructure, snapshotMetadata *SnapshotMetadata) error {
	// Networking for clones -- guide:
	// https://github.com/firecracker-microvm/firecracker/blob/main/docs/snapshotting/network-for-clones.md

	// TODO: make a pool of TAPs and remove TAP creation from the critical path

	var gatewayIP, vmip, mac, dev string
	if snapshotMetadata == nil {
		// Hardcode each IP address. This is possible as they all reside in different network namespaces.
		gatewayIP, vmip, mac = "169.254.0.1", "169.254.0.2", "02:FC:00:00:00:00"
		dev = fmt.Sprintf("%s%d%s", tapDevicePrefix, rand.Int()%1_000_000, tapDeviceSuffix)
	} else {
		gatewayIP = snapshotMetadata.GatewayIP
		vmip = snapshotMetadata.VMIP
		mac = snapshotMetadata.MacAddress
		dev = snapshotMetadata.HostDevName
	}

	err := exec.Command("sudo", "ip", "netns", "add", vmcs.NetworkNS).Run()
	if err != nil {
		logrus.Errorf("failed to create network namespace %s - %v", vmcs.NetworkNS, err)
		return err
	}

	err = exec.Command("sudo", "ip", "netns", "exec", vmcs.NetworkNS, "ip", "tuntap", "add", "dev", dev, "mode", "tap").Run()
	if err != nil {
		logrus.Errorf("failed to create a TUN/TAP device %s - %v", dev, err)
		return err
	}

	err = exec.Command("sudo", "sysctl", "-w net.ipv4.conf."+dev+".proxy_arp=1").Run()
	if err != nil {
		logrus.Errorf("failed to set proxy_arp for device %s - %v", dev, err)
		return err
	}

	err = exec.Command("sudo", "sysctl", "-w net.ipv6.conf."+dev+".disable_ipv6=1").Run()
	if err != nil {
		logrus.Errorf("failed to disable IPv6 for device %s - %v", dev, err)
		return err
	}

	err = exec.Command("sudo", "ip", "netns", "exec", vmcs.NetworkNS, "ip", "addr", "add", gatewayIP+"/30", "dev", dev).Run()
	if err != nil {
		logrus.Errorf("failed to assign IP address to device %s - %v", dev, err)
		return err
	}

	err = exec.Command("sudo", "ip", "netns", "exec", vmcs.NetworkNS, "ip", "link", "set", "dev", dev, "up").Run()
	if err != nil {
		logrus.Errorf("failed to bring up the device %s - %v", dev, err)
		return err
	}

	exposedIP, err := createVETHPair(internalIPManager, exposedIPManager, vmcs, vmip)
	if err != nil {
		logrus.Errorf("failed to create and configure veth pair - %v", err)
		return err
	}

	vmcs.TapLink = &TAPLink{
		Device:    dev,
		MAC:       mac,
		GatewayIP: gatewayIP,
		VmIP:      vmip,
		ExposedIP: exposedIP,
	}

	return nil
}

var ethGenerator = managers.NewVETHNameGenerator()

func createVETHPair(internalIPGenerator *IPManager, exposedIPGenerator *IPManager, vmcs *VMControlStructure, guestTAP string) (string, error) {
	id := ethGenerator.Generate()

	guestSidePairName := fmt.Sprintf("%s%d%s", vethPrefix, id, vethSuffix)
	hostSidePairName := fmt.Sprintf("%s%d%s", vethPrefix, id+1, vethSuffix)

	hostSideIpAddress, workloadIpAddress := internalIPGenerator.GenerateIPMACPair()
	hostSideIpAddressWithMask := hostSideIpAddress + "/24"
	workloadIpAddressWithMask := workloadIpAddress + "/24"

	logrus.Debugf("Host-side IP address: %s", hostSideIpAddress)
	logrus.Debugf("Guest-side IP address: %s", workloadIpAddress)

	exposedIP := exposedIPGenerator.GenerateSingleIP()
	logrus.Debugf("Exposed IP address: %s", exposedIP)

	//hostSideIpAddress := "10.0.0.1"
	//workloadIpAddress := "10.0.0.2"
	//guestTAP := "169.254.0.2"
	//exposedIP := "192.168.0.3"

	// create a vETH pair - both created in the netNS namespace
	err := exec.Command("sudo", "ip", "netns", "exec", vmcs.NetworkNS, "ip", "link", "add", hostSidePairName, "type", "veth", "peer", "name", guestSidePairName).Run()
	if err != nil {
		logrus.Error("Error creating a virtual Ethernet pair - ", err)
		return exposedIP, err
	}

	// move the host-side of te pair to the root namespace
	err = exec.Command("sudo", "ip", "netns", "exec", vmcs.NetworkNS, "ip", "link", "set", hostSidePairName, "netns", "1").Run()
	if err != nil {
		logrus.Error("Error moving host-side of the pair to the root namespace - ", err)
		return exposedIP, err
	}

	// assign an IP address to the guest-side pair
	err = exec.Command("sudo", "ip", "netns", "exec", vmcs.NetworkNS, "ip", "addr", "add", workloadIpAddressWithMask, "dev", guestSidePairName).Run()
	if err != nil {
		logrus.Error("Error assigning an IP address to the guest-side of the pair - ", err)
		return exposedIP, err
	}

	// bringing the guest-side pair up
	err = exec.Command("sudo", "ip", "netns", "exec", vmcs.NetworkNS, "ip", "link", "set", "dev", guestSidePairName, "up").Run()
	if err != nil {
		logrus.Error("Error bringing the guest-end of the pair up - ", err)
		return exposedIP, err
	}

	// assigning an IP address to the host-side of the pair
	err = exec.Command("sudo", "ip", "addr", "add", hostSideIpAddressWithMask, "dev", hostSidePairName).Run()
	if err != nil {
		logrus.Error("Error assigning an IP address to the host-end of the pair - ", err)
		return exposedIP, err
	}

	// bringing the host-side pair up
	err = exec.Command("sudo", "ip", "link", "set", "dev", hostSidePairName, "up").Run()
	if err != nil {
		logrus.Error("Error bringing the host-end of the pair up - ", err)
		return exposedIP, err
	}

	// assigning default gateway of the namespace
	err = exec.Command("sudo", "ip", "netns", "exec", vmcs.NetworkNS, "ip", "route", "add", "default", "via", hostSideIpAddress).Run()
	if err != nil {
		logrus.Error("Error assigning default gateway of the namespace - ", vmcs.NetworkNS)
		return exposedIP, err
	}

	// NAT for outgoing traffic from the namespace
	err = exec.Command("sudo", "ip", "netns", "exec", vmcs.NetworkNS, "iptables", "-t", "nat", "-A", "POSTROUTING", "-o", guestSidePairName, "-s", guestTAP, "-j", "SNAT", "--to", exposedIP).Run()
	if err != nil {
		logrus.Error("Error adding a NAT rule for the outgoing traffic from the namespace - ", err)
		return exposedIP, err
	}

	// NAT for incoming traffic to the namespace
	err = exec.Command("sudo", "ip", "netns", "exec", vmcs.NetworkNS, "iptables", "-t", "nat", "-A", "PREROUTING", "-i", guestSidePairName, "-d", exposedIP, "-j", "DNAT", "--to", guestTAP).Run()
	if err != nil {
		logrus.Error("Error adding a NAT rule for the incoming traffic to the namespace - ", err)
		return exposedIP, err
	}

	// adding route to the workload
	err = exec.Command("sudo", "ip", "route", "add", exposedIP, "via", workloadIpAddress).Run()
	if err != nil {
		logrus.Error("Error adding route to the workload - ", err)
		return exposedIP, err
	}

	return exposedIP, nil
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

package firecracker

import (
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
)

type TAPLink struct {
	Device    string
	MAC       string
	GatewayIP string
	VmIP      string
}

func createTAPDevice(vmcs *VMControlStructure) error {
	// TODO: make a pool of TAPs and remove TAP creation from the critical path

	gatewayIP, vmip, mac := vmcs.IpManager.GenerateIPMACPair()
	dev := fmt.Sprintf("%s%d%s", tapDevicePrefix, rand.Int()%1_000_000, tapDeviceSuffix)

	err := exec.Command("sudo", "ip", "tuntap", "add", "dev", dev, "mode", "tap").Run()
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

	err = exec.Command("sudo", "ip", "addr", "add", gatewayIP+"/30", "dev", dev).Run()
	if err != nil {
		logrus.Errorf("failed to assign IP address to device %s - %v", dev, err)
		return err
	}

	err = exec.Command("sudo", "ip", "link", "set", "dev", dev, "up").Run()
	if err != nil {
		logrus.Errorf("failed to bring up the device %s - %v", dev, err)
		return err
	}

	err = exec.Command("sudo", "arp", "-s", vmip, mac).Run()
	if err != nil {
		logrus.Errorf("failed to configure arp entry for device %s - %v", dev, err)
		return err
	}
	logrus.Debug("ARP table entry inserted ", vmip, " ", mac)

	vmcs.tapLink = &TAPLink{
		Device:    dev,
		MAC:       mac,
		GatewayIP: gatewayIP,
		VmIP:      vmip,
	}

	return nil
}

func DeleteFirecrackerTAPDevices() error {
	devices, err := net.Interfaces()
	if err != nil {
		logrus.Error("Error listing network devices.")
		return err
	}

	for _, dev := range devices {
		if !strings.HasPrefix(dev.Name, tapDevicePrefix) || !strings.HasSuffix(dev.Name, tapDeviceSuffix) {
			continue
		}

		err = exec.Command("sudo", "ip", "link", "del", dev.Name).Run()
		if err != nil {
			logrus.Error("Error deleting network device '", dev.Name, "'.")
			return err
		}
	}

	return err
}

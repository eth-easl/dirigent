package firecracker

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"math/rand"
	"os/exec"
)

const ()

type TAPLink struct {
	Device string
	MAC    string
	IP     string
}

func createTAPDevice(vmcs *VMControlStructure) error {
	ip := vmcs.ipManager.GenerateIP()
	dev := fmt.Sprintf("fc-%d-tap", rand.Int()%1_000_000)

	// Remove the TAP device if it exists; error means device not found
	_ = removeTapDevice(dev)

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

	err = exec.Command("sudo", "ip", "addr", "add", ip+"/30", "dev", dev).Run()
	if err != nil {
		logrus.Errorf("failed to assign IP address to device %s - %v", dev, err)
		return err
	}

	err = exec.Command("sudo", "ip", "link", "set", "dev", dev, "up").Run()
	if err != nil {
		logrus.Errorf("failed to bring up the device %s - %v", dev, err)
		return err
	}

	vmcs.tapLink = &TAPLink{
		Device: dev,
		IP:     ip,
	}

	return nil
}

func removeTapDevice(dev string) error {
	// We forward the error for now, as a failure here may mean that the TAP
	// device doesn't exist, which is not a critical error
	err := exec.Command("sudo", "ip", "link", "del", dev).Run()
	if err != nil {
		return err
	}

	// TODO: should we set the link down first before removing it
	err = exec.Command("sudo", "ip", "link", "set", "dev", dev, "down").Run()
	return err
}

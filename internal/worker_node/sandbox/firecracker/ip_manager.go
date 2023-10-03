package firecracker

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"sync/atomic"
)

type IPManager struct {
	networkPrefix     string
	allocationCounter uint32
}

func NewIPManager(networkPrefix string) *IPManager {
	return &IPManager{
		networkPrefix:     networkPrefix,
		allocationCounter: 0,
	}
}

func (ipm *IPManager) getUniqueCounterValue() uint32 {
	var oldValue uint32
	swapped := false

	for !swapped {
		oldValue = atomic.LoadUint32(&ipm.allocationCounter)
		if oldValue >= 65536 {
			logrus.Fatal("Run out of IP addresses.")
		}

		// We need four IP addresses per VM - gateway, 2 TAPs, broadcast.
		// Hence, there are enough IP addresses for 16 384 VMs
		newValue := oldValue + 4

		swapped = atomic.CompareAndSwapUint32(&ipm.allocationCounter, oldValue, newValue)
	}

	return oldValue
}

func (ipm *IPManager) generateRawGatewayIP() (uint32, uint32) {
	val := ipm.getUniqueCounterValue()

	return extractThirdField(val), extractFourthField(val)
}

func (ipm *IPManager) GenerateIPMACPair() (string, string, string) {
	c, d := ipm.generateRawGatewayIP()

	ip := fmt.Sprintf("%s.%d.%d", ipm.networkPrefix, c, d+1)
	vmip := fmt.Sprintf("%s.%d.%d", ipm.networkPrefix, c, d+2)
	mac := fmt.Sprintf("02:FC:00:00:%02x:%02x", c, d)

	return ip, vmip, mac
}

func extractThirdField(counterValue uint32) uint32 {
	return counterValue / 256
}

func extractFourthField(counterValue uint32) uint32 {
	return counterValue % 256
}

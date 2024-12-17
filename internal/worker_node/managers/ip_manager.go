package managers

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
		newValue := oldValue + 1

		swapped = atomic.CompareAndSwapUint32(&ipm.allocationCounter, oldValue, newValue)
	}

	return oldValue
}

func (ipm *IPManager) generateRawGatewayIP() (uint32, uint32) {
	val := ipm.getUniqueCounterValue()

	thirdField := extractThirdField(val)
	fourthField := extractFourthField(val)

	if thirdField > 255 || fourthField > 255 {
		logrus.Errorf("Invalid IP generated.")
		return 256, 256
	}

	return thirdField, fourthField
}

func (ipm *IPManager) GenerateSingleIP() string {
	var c, d uint32
	for d == 0 || d == 255 {
		c, d = ipm.generateRawGatewayIP()
	}

	ip := fmt.Sprintf("%s.%d.%d", ipm.networkPrefix, c, d)

	return ip
}

func (ipm *IPManager) GenerateIPMACPair() (string, string) {
	c, d := ipm.generateRawGatewayIP()

	ip := fmt.Sprintf("%s.%d.%d.1", ipm.networkPrefix, c, d)
	vmip := fmt.Sprintf("%s.%d.%d.2", ipm.networkPrefix, c, d)

	return ip, vmip
}

func extractThirdField(counterValue uint32) uint32 {
	return counterValue / 256
}

func extractFourthField(counterValue uint32) uint32 {
	return counterValue % 256
}

func CIDRToPrefix(cidr string) string {
	var s1, s2, s3, s4, s5 int
	cnt, err := fmt.Sscanf(cidr, "%d.%d.%d.%d/%d", &s1, &s2, &s3, &s4, &s5)
	if err != nil || cnt != 5 {
		logrus.Fatalf("Invalid CIDR received from the control plane. Terminating... - %v", err)
	}

	return fmt.Sprintf("%d.%d", s1, s2)
}

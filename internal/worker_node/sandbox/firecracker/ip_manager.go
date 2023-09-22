package firecracker

import (
	"fmt"
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
		newValue := oldValue + 1

		swapped = atomic.CompareAndSwapUint32(&ipm.allocationCounter, oldValue, newValue)
	}

	return oldValue
}

func (ipm *IPManager) generateRawIP() (uint32, uint32) {
	val := ipm.getUniqueCounterValue()

	// x.x.x.0.and x.x.x.255 are reserved addresses
	for val%256 == 0 || val%256 == 255 {
		val = ipm.getUniqueCounterValue()
	}

	return extractThirdField(val), extractFourthField(val)
}

func (ipm *IPManager) GenerateIP() string {
	c, d := ipm.generateRawIP()
	return fmt.Sprintf("%s.%d.%d", ipm.networkPrefix, c, d)
}

func extractThirdField(counterValue uint32) uint32 {
	return counterValue / 256
}
func extractFourthField(counterValue uint32) uint32 {
	return counterValue % 256
}

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

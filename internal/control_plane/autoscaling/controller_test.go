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

package autoscaling

import (
	"cluster_manager/api/proto"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func getScalingChannel() chan int {
	return make(chan int)
}

func getServiceInfo() *proto.ServiceInfo {
	return &proto.ServiceInfo{
		Name:           "mockService",
		Image:          "",
		PortForwarding: nil,
		AutoscalingConfig: &proto.AutoscalingConfiguration{
			ScalingUpperBound:                    1,
			ScalingLowerBound:                    0,
			PanicThresholdPercentage:             0,
			MaxScaleUpRate:                       0,
			MaxScaleDownRate:                     0,
			ContainerConcurrency:                 0,
			ContainerConcurrencyTargetPercentage: 0,
			StableWindowWidthSeconds:             0,
			PanicWindowWidthSeconds:              0,
			ScalingPeriodSeconds:                 0,
		},
	}
}

func TestSimpleController(t *testing.T) {
	scalingChannel := getScalingChannel()
	serviceInfo := getServiceInfo()

	pfStateController := NewPerFunctionStateController(scalingChannel, serviceInfo, time.Millisecond*100)

	assert.True(t, pfStateController.Start(), "Start should return true")
}

func TestMultipleStarts(t *testing.T) {
	scalingChannel := getScalingChannel()
	serviceInfo := getServiceInfo()

	pfStateController := NewPerFunctionStateController(scalingChannel, serviceInfo, time.Millisecond*100)
	assert.True(t, pfStateController.Start(), "Start should return true")
	for i := 0; i < 10000; i++ {
		assert.False(t, pfStateController.Start(), "Start should return false")
	}
}

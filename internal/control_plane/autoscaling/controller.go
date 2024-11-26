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
	"cluster_manager/internal/control_plane/core"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
)

type PFStateController struct {
	EndpointLock sync.Mutex

	ServiceName string

	AutoscalingRunning  int32
	DesiredStateChannel chan int

	ScalingMetadata AutoscalingMetadata
	Endpoints       []*core.Endpoint

	Period time.Duration
	StopCh chan struct{}
}

func NewPerFunctionStateController(scalingChannel chan int, serviceInfo *proto.ServiceInfo, period time.Duration) *PFStateController {
	return &PFStateController{
		ServiceName:         serviceInfo.Name,
		DesiredStateChannel: scalingChannel,
		Period:              period,
		ScalingMetadata: AutoscalingMetadata{
			AutoscalingConfig:            serviceInfo.AutoscalingConfig,
			inflightRequestsPerDataPlane: make(map[string]int32),
			inflightRequestsLock:         sync.RWMutex{},
		},
	}
}

func (as *PFStateController) Start() bool {
	if atomic.CompareAndSwapInt32(&as.AutoscalingRunning, 0, 1) {
		as.StopCh = make(chan struct{}, 1)
		go as.ScalingLoop()
		return true
	}

	return false
}

func (as *PFStateController) Stop() {
	if atomic.LoadInt32(&as.AutoscalingRunning) != 0 {
		as.StopCh <- struct{}{}
	}
}

func (as *PFStateController) scalingCycle(isScaleFromZero bool) (stopped bool) {
	desiredScale := as.ScalingMetadata.KnativeScaling(isScaleFromZero)
	logrus.Debugf("Desired scale for %s is %d", as.ServiceName, desiredScale)

	as.DesiredStateChannel <- desiredScale

	if desiredScale == 0 {
		as.stopAutoscalingLoop()
		return true
	}

	return false
}

func (as *PFStateController) stopAutoscalingLoop() {
	logrus.Debugf("Exited scaling loop for %s.", as.ServiceName)

	atomic.StoreInt32(&as.AutoscalingRunning, 0)
	close(as.StopCh)
	as.StopCh = nil
}

func (as *PFStateController) ScalingLoop() {
	logrus.Debugf("Starting scaling loop for %s.", as.ServiceName)

	// need to make the first tick happen right away
	toStop := as.scalingCycle(true)
	if toStop {
		return
	}

	ticker := time.NewTicker(as.Period)
	for {
		select {
		case <-ticker.C: // first event only after as.Period delay
			toStop = as.scalingCycle(false)
			if toStop {
				return
			}
		case <-as.StopCh:
			as.stopAutoscalingLoop()
			return
		}
	}
}

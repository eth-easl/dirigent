package per_function_state

import (
	"cluster_manager/internal/control_plane/control_plane/core"
	"cluster_manager/proto"
	"sync"
	"time"
)

type PFState struct {
	ServiceName string

	AutoscalingRunning  int32
	DesiredStateChannel chan int

	ScalingMetadata *AutoscalingMetadata

	EndpointLock sync.Mutex
	Endpoints    []*core.Endpoint

	Period time.Duration
	StopCh chan struct{}
}

func NewPerFunctionState(scalingChannel chan int, serviceInfo *proto.ServiceInfo, period time.Duration) *PFState {
	return &PFState{
		ServiceName:         serviceInfo.Name,
		DesiredStateChannel: scalingChannel,
		Period:              period,
		ScalingMetadata: &AutoscalingMetadata{
			AutoscalingConfig:            serviceInfo.AutoscalingConfig,
			inflightRequestsPerDataPlane: make(map[string]int32),
			inflightRequestsLock:         sync.RWMutex{},
		},
	}
}

func (pfState *PFState) GetNumberEndpoint() int {
	pfState.EndpointLock.Lock()
	defer pfState.EndpointLock.Unlock()

	return len(pfState.Endpoints)
}

// TODO: Extract this in a common function
/*func (as *PFState) Start() bool {
	if atomic.CompareAndSwapInt32(&as.AutoscalingRunning, 0, 1) {
		as.StopCh = make(chan struct{}, 1)
		go as.scalingLoop()
		return true
	}

	return false
}

func (as *PFState) Stop() {
	if atomic.LoadInt32(&as.AutoscalingRunning) != 0 {
		as.StopCh <- struct{}{}
	}
}

func (as *PFState) scalingCycle(isScaleFromZero bool) (stopped bool) {
	desiredScale := as.ScalingMetadata.KnativeScaling(isScaleFromZero)
	logrus.Debugf("Desired scale for %s is %d", as.ServiceName, desiredScale)

	as.DesiredStateChannel <- desiredScale

	if desiredScale == 0 {
		as.stopAutoscalingLoop()
		return true
	}

	return false
}

func (as *PFState) stopAutoscalingLoop() {
	logrus.Debugf("Exited scaling loop for %s.", as.ServiceName)

	atomic.StoreInt32(&as.AutoscalingRunning, 0)
	close(as.StopCh)
	as.StopCh = nil
}

func (as *PFState) scalingLoop() {
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
}*/

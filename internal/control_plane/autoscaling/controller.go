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
		go as.ScalingLoop()
		return true
	}

	return false
}

func (as *PFStateController) ScalingLoop() {
	ticker := time.NewTicker(as.Period)

	isScaleFromZero := true

	logrus.Debugf("Starting scaling loop for %s.", as.ServiceName)

	for ; true; <-ticker.C {
		desiredScale := as.ScalingMetadata.KnativeScaling(isScaleFromZero)
		logrus.Debugf("Desired scale for %s is %d", as.ServiceName, desiredScale)

		as.DesiredStateChannel <- desiredScale

		isScaleFromZero = false

		if desiredScale == 0 {
			atomic.StoreInt32(&as.AutoscalingRunning, 0)
			logrus.Debugf("Exited scaling loop for %s.", as.ServiceName)

			break
		}
	}
}

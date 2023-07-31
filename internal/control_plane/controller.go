package control_plane

import (
	"cluster_manager/internal/control_plane/autoscaling"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
)

type PFStateController struct {
	EndpointLock sync.Mutex

	AutoscalingRunning  int32
	DesiredStateChannel *chan int

	ScalingMetadata autoscaling.AutoscalingMetadata
	Endpoints       []*Endpoint

	Period time.Duration
}

func (as *PFStateController) Start() {
	if atomic.CompareAndSwapInt32(&as.AutoscalingRunning, 0, 1) {
		go as.ScalingLoop()
	}
}

func (as *PFStateController) ScalingLoop() {
	ticker := time.NewTicker(as.Period)

	isScaleFromZero := true

	logrus.Debug("Starting scaling loop")

	for ; true; <-ticker.C {
		desiredScale := as.ScalingMetadata.KnativeScaling(isScaleFromZero)
		logrus.Debug("Desired scale: ", desiredScale)

		*as.DesiredStateChannel <- desiredScale

		isScaleFromZero = false
		if desiredScale == 0 {
			atomic.StoreInt32(&as.AutoscalingRunning, 0)
			logrus.Debug("Existed scaling loop")

			break
		}
	}
}

package control_plane

import (
	"cluster_manager/internal/control_plane/autoscaling"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

type ScalingMetric string

type PFStateController struct {
	sync.Mutex

	AutoscalingRunning  bool
	DesiredStateChannel *chan int

	ScalingMetadata autoscaling.AutoscalingMetadata
	Endpoints       []*Endpoint

	Period time.Duration
}

func (as *PFStateController) Start() {
	as.Lock()
	defer as.Unlock()

	if !as.AutoscalingRunning {
		as.AutoscalingRunning = true
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

		if isScaleFromZero {
			isScaleFromZero = false
		}

		if desiredScale == 0 {
			as.Lock()
			as.AutoscalingRunning = false
			as.Unlock()

			logrus.Debug("Existed scaling loop")

			break
		}
	}
}

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

	AutoscalingRunning  int32
	DesiredStateChannel *chan int

	ScalingMetadata AutoscalingMetadata
	Endpoints       []*core.Endpoint

	Period time.Duration
}

func NewPerFunctionStateController(scalingChannel *chan int, serviceInfo *proto.ServiceInfo) *PFStateController {
	return &PFStateController{
		DesiredStateChannel: scalingChannel,
		Period:              2 * time.Second, // TODO: hardcoded autoscaling period for now
		ScalingMetadata: AutoscalingMetadata{
			AutoscalingConfig: serviceInfo.AutoscalingConfig,
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

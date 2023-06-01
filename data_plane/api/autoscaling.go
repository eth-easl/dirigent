package api

import (
	"github.com/sirupsen/logrus"
	"sync"
	"time"
)

type Autoscaler struct {
	sync.Mutex

	Running       bool
	StopChannel   chan struct{}
	NotifyChannel *chan int

	Period time.Duration
}

func (as *Autoscaler) determineScale() int {
	// TODO: implement some autoscaling policy
	return 1
}

func (as *Autoscaler) Start() {
	as.Lock()
	defer as.Unlock()

	if !as.Running {
		go as.ScalingLoop()
	}
}

func (as *Autoscaler) ScalingLoop() {
	timer := time.NewTimer(as.Period)

	for {
		select {
		case <-timer.C:
			*as.NotifyChannel <- as.determineScale()
		case <-as.StopChannel:
			as.Lock()
			as.Running = false
			as.Unlock()

			logrus.Debug("Autoscaling stopped for ")
			break
		}
	}
}

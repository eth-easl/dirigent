package api

import (
	"cluster_manager/common"
	"github.com/sirupsen/logrus"
	"log"
	"math"
	"math/rand"
	"sync"
	"time"
)

type ScalingMetric string

type PFStateController struct {
	sync.Mutex

	ActualScale  int
	DesiredScale int

	DesiredStateChannel *chan int

	AutoscalingRunning bool
	StopChannel        chan struct{}
	NotifyChannel      *chan int

	scalingMetric map[ScalingMetric]float64

	Period time.Duration
}

//////////////////////////////////////////////////////
//////////////////////////////////////////////////////
//////////////////////////////////////////////////////

func (as *PFStateController) determineScale() int {
	if as.ActualScale == 0 {
		return 1
	}

	metric, ok := as.scalingMetric["testMetric"]
	if !ok {
		log.Fatal("Invalid autoscaling metric requested from the cache")
	}
	return int(math.Ceil(metric / 10))
}

func (as *PFStateController) Start() {
	as.Lock()
	defer as.Unlock()

	if !as.AutoscalingRunning {
		go as.ScalingLoop()
	}
}

func (as *PFStateController) ScalingLoop() {
	timer := time.NewTimer(as.Period)

	for {
		select {
		case <-timer.C:
			*as.NotifyChannel <- as.determineScale()
		case <-as.StopChannel:
			as.Lock()
			as.AutoscalingRunning = false
			as.Unlock()

			logrus.Debug("Autoscaling stopped")
			break
		}
	}
}

//////////////////////////////////////////////////////
// POLICIES
//////////////////////////////////////////////////////

func placementPolicy(storage *NodeInfoStorage) *WorkerNode {
	storage.Lock()
	defer storage.Unlock()

	nodes := common.Keys(storage.NodeInfo)
	index := rand.Intn(len(nodes))

	nodeName := nodes[index]
	nodeRef := storage.NodeInfo[nodeName]

	return nodeRef
}

func evictionPolicy(endpoint *[]Endpoint) (*Endpoint, []Endpoint) {
	index := rand.Intn(len(*endpoint))
	newEndpointList := append((*endpoint)[:index], (*endpoint)[index+1:]...)

	return &(*endpoint)[index], newEndpointList
}

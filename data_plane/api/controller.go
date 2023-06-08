package api

import (
	"cluster_manager/common"
	"github.com/sirupsen/logrus"
	"math/rand"
	"sync"
	"time"
)

type ScalingMetric string

type PFStateController struct {
	sync.Mutex

	DesiredStateChannel *chan int

	AutoscalingRunning bool
	NotifyChannel      *chan int

	ScalingMetadata AutoscalingMetadata
	Endpoints       []*Endpoint

	Period time.Duration
}

//////////////////////////////////////////////////////
//////////////////////////////////////////////////////
//////////////////////////////////////////////////////

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

		*as.NotifyChannel <- desiredScale

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

func evictionPolicy(endpoint []*Endpoint) (*Endpoint, []*Endpoint) {
	if len(endpoint) == 0 {
		return nil, []*Endpoint{}
	}

	index := rand.Intn(len(endpoint))
	var newEndpointList []*Endpoint
	for i := 0; i < len(endpoint); i++ {
		if i != index {
			newEndpointList = append(newEndpointList, endpoint[i])
		}
	}

	return endpoint[index], newEndpointList
}

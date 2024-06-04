package per_function_state

import (
	"cluster_manager/internal/control_plane/control_plane/core"
	"cluster_manager/proto"
	"sync"
	"sync/atomic"
)

type PFState struct {
	ServiceName string

	EndpointLock sync.Mutex
	Endpoints    []*core.Endpoint

	ActualScale int64

	AutoscalingConfig *proto.AutoscalingConfiguration

	inflightRequestsPerDataPlane map[string]int32
	inflightRequestsLock         sync.RWMutex

	DesiredStateChannel  chan int
	CachedScalingMetrics int32

	StopCh chan struct{}

	RpsValue float64
}

func NewPerFunctionState(serviceInfo *proto.ServiceInfo) *PFState {
	return &PFState{
		AutoscalingConfig:            serviceInfo.AutoscalingConfig,
		inflightRequestsPerDataPlane: make(map[string]int32),
		inflightRequestsLock:         sync.RWMutex{},
		DesiredStateChannel:          make(chan int),
		ServiceName:                  serviceInfo.Name,
	}
}

func (pfState *PFState) GetNumberEndpoint() int {
	pfState.EndpointLock.Lock()
	defer pfState.EndpointLock.Unlock()

	return len(pfState.Endpoints)
}

func (pfState *PFState) RemoveDataplane(dataplaneName string) {
	pfState.inflightRequestsLock.Lock()
	defer pfState.inflightRequestsLock.Unlock()

	atomic.AddInt32(&pfState.CachedScalingMetrics, -pfState.inflightRequestsPerDataPlane[dataplaneName])
	delete(pfState.inflightRequestsPerDataPlane, dataplaneName)
}

func (pfState *PFState) SetCachedScalingMetrics(metrics *proto.AutoscalingMetric) {
	pfState.inflightRequestsLock.Lock()
	defer pfState.inflightRequestsLock.Unlock()

	atomic.AddInt32(&pfState.CachedScalingMetrics, metrics.InflightRequests-pfState.inflightRequestsPerDataPlane[metrics.DataplaneName])
	pfState.inflightRequestsPerDataPlane[metrics.DataplaneName] = metrics.InflightRequests

	pfState.RpsValue = float64(metrics.RpsValue)
}

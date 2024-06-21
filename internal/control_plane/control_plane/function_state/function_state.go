package function_state

import (
	"cluster_manager/internal/control_plane/control_plane/core"
	"cluster_manager/proto"
	"sync"
	"sync/atomic"
)

type FunctionState struct {
	ServiceName string

	EndpointLock sync.Mutex
	Endpoints    []*core.Endpoint

	ActualScale int64

	ServiceInfo *proto.ServiceInfo

	inflightRequestsPerDataPlane map[string]int32
	inflightRequestsLock         sync.RWMutex

	DesiredStateChannel  chan int
	CachedScalingMetrics int32

	StopCh chan struct{}

	RpsValue float64
}

func NewFunctionState(serviceInfo *proto.ServiceInfo) *FunctionState {
	return &FunctionState{
		ServiceInfo:                  serviceInfo,
		inflightRequestsPerDataPlane: make(map[string]int32),
		inflightRequestsLock:         sync.RWMutex{},
		DesiredStateChannel:          make(chan int),
		ServiceName:                  serviceInfo.Name,
	}
}

func (fState *FunctionState) GetNumberEndpoint() int {
	fState.EndpointLock.Lock()
	defer fState.EndpointLock.Unlock()

	return len(fState.Endpoints)
}

func (fState *FunctionState) RemoveDataplane(dataplaneName string) {
	fState.inflightRequestsLock.Lock()
	defer fState.inflightRequestsLock.Unlock()

	atomic.AddInt32(&fState.CachedScalingMetrics, -fState.inflightRequestsPerDataPlane[dataplaneName])
	delete(fState.inflightRequestsPerDataPlane, dataplaneName)
}

func (fState *FunctionState) SetCachedScalingMetrics(metrics *proto.AutoscalingMetric) {
	fState.inflightRequestsLock.Lock()
	defer fState.inflightRequestsLock.Unlock()

	atomic.AddInt32(&fState.CachedScalingMetrics, metrics.InflightRequests-fState.inflightRequestsPerDataPlane[metrics.DataplaneName])
	fState.inflightRequestsPerDataPlane[metrics.DataplaneName] = metrics.InflightRequests

	fState.RpsValue = float64(metrics.RpsValue)
}

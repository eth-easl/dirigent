package service_state

import (
	"cluster_manager/internal/control_plane/control_plane/core"
	"cluster_manager/proto"
	"sync"
	"sync/atomic"
)

type ServiceState struct {
	ServiceName string

	EndpointLock sync.Mutex
	Endpoints    []*core.Endpoint

	ActualScale int64

	// if service is a function then TaskInfo is nil and FunctionInfo contains exactly one element (the function)
	TaskInfo     *proto.WorkflowTaskInfo
	FunctionInfo []*proto.ServiceInfo

	inflightRequestsPerDataPlane map[string]int32
	inflightRequestsLock         sync.RWMutex

	DesiredStateChannel  chan int
	CachedScalingMetrics int32

	StopCh chan struct{}

	RpsValue float64
}

func NewFunctionState(serviceInfo *proto.ServiceInfo) *ServiceState {
	return &ServiceState{
		FunctionInfo:                 []*proto.ServiceInfo{serviceInfo},
		inflightRequestsPerDataPlane: make(map[string]int32),
		inflightRequestsLock:         sync.RWMutex{},
		DesiredStateChannel:          make(chan int),
		ServiceName:                  serviceInfo.Name,
	}
}
func NewTaskState(taskInfo *proto.WorkflowTaskInfo, functions []*proto.ServiceInfo) *ServiceState {
	return &ServiceState{
		TaskInfo:                     taskInfo,
		FunctionInfo:                 functions,
		inflightRequestsPerDataPlane: make(map[string]int32),
		inflightRequestsLock:         sync.RWMutex{},
		DesiredStateChannel:          make(chan int),
		ServiceName:                  taskInfo.Name,
	}
}

func (ss *ServiceState) GetNumberEndpoint() int {
	ss.EndpointLock.Lock()
	defer ss.EndpointLock.Unlock()

	return len(ss.Endpoints)
}

func (ss *ServiceState) RemoveDataplane(dataplaneName string) {
	ss.inflightRequestsLock.Lock()
	defer ss.inflightRequestsLock.Unlock()

	atomic.AddInt32(&ss.CachedScalingMetrics, -ss.inflightRequestsPerDataPlane[dataplaneName])
	delete(ss.inflightRequestsPerDataPlane, dataplaneName)
}

func (ss *ServiceState) SetCachedScalingMetrics(metrics *proto.AutoscalingMetric) {
	ss.inflightRequestsLock.Lock()
	defer ss.inflightRequestsLock.Unlock()

	atomic.AddInt32(&ss.CachedScalingMetrics, metrics.InflightRequests-ss.inflightRequestsPerDataPlane[metrics.DataplaneName])
	ss.inflightRequestsPerDataPlane[metrics.DataplaneName] = metrics.InflightRequests

	ss.RpsValue = float64(metrics.RpsValue)
}

func (ss *ServiceState) GetRequestedCpu() uint64 {
	if ss.TaskInfo == nil {
		return ss.FunctionInfo[0].RequestedCpu
	}

	// for task take the maximum over all task functions
	maxCpu := uint64(0)
	for _, fInfo := range ss.FunctionInfo {
		if fInfo.RequestedCpu > maxCpu {
			maxCpu = fInfo.RequestedCpu
		}
	}
	return maxCpu
}
func (ss *ServiceState) GetRequestedMemory() uint64 {
	if ss.TaskInfo == nil {
		return ss.FunctionInfo[0].RequestedMemory
	}

	// for task take the maximum over all task functions
	maxMemory := uint64(0)
	for _, fInfo := range ss.FunctionInfo {
		if fInfo.RequestedMemory > maxMemory {
			maxMemory = fInfo.RequestedMemory
		}
	}
	return maxMemory
}
func (ss *ServiceState) GetAutoscalingConfig() *proto.AutoscalingConfiguration {
	if ss.TaskInfo == nil {
		return ss.FunctionInfo[0].AutoscalingConfig
	}

	// TODO: how to aggregate over all configs?
	return ss.FunctionInfo[0].AutoscalingConfig
}

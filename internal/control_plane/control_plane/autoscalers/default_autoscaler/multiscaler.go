package default_autoscaler

import (
	"cluster_manager/internal/control_plane/control_plane/per_function_state"
	"cluster_manager/pkg/synchronization"
	"cluster_manager/proto"
	"time"
)

type DefaultMultiscaler struct {
	autoscalers       synchronization.SyncStructure[string, *defaultAutoscaler]
	autoscalingPeriod time.Duration
}

func NewMultiscaler(autoscalingPeriod time.Duration) *DefaultMultiscaler {
	return &DefaultMultiscaler{
		autoscalers:       synchronization.NewControlPlaneSyncStructure[string, *defaultAutoscaler](),
		autoscalingPeriod: autoscalingPeriod,
	}
}

func (m *DefaultMultiscaler) Create(perFunctionState *per_function_state.PFState) {
	m.autoscalers.AtomicSet(perFunctionState.ServiceName, newDefaultAutoscaler(perFunctionState, m.autoscalingPeriod))
}

func (m *DefaultMultiscaler) PanicPoke(functionName string, previousValue int32) {
	m.autoscalers.RLock()
	defer m.autoscalers.RUnlock()

	m.autoscalers.GetNoCheck(functionName).panicPoke()
}

func (m *DefaultMultiscaler) Poke(functionName string, previousValue int32) {
	m.autoscalers.RLock()
	defer m.autoscalers.RUnlock()

	m.autoscalers.GetNoCheck(functionName).poke()
}

// We don't need any data plane metrics for this autoscaler, we can safely ignore the call
func (m *DefaultMultiscaler) ForwardDataplaneMetrics(_ *proto.MetricsPredictiveAutoscaler) error {
	return nil
}

func (m *DefaultMultiscaler) Stop(functionName string) {
	m.autoscalers.Lock()
	defer m.autoscalers.Unlock()

	m.autoscalers.GetNoCheck(functionName).stop()
	m.autoscalers.Remove(functionName)
}

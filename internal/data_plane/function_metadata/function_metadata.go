package function_metadata

import (
	"cluster_manager/api/proto"
	"cluster_manager/pkg/atomic_map"
	_map "cluster_manager/pkg/map"
	"container/list"
	"context"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	UnlimitedConcurrency uint = 0
)

type RequestThrottler chan struct{}

type UpstreamEndpoint struct {
	ID       string
	Capacity RequestThrottler
	URL      string
}

type LoadBalancingMetadata struct {
	RoundRobinCounter           uint32
	KubernetesRoundRobinCounter uint32
	RequestCountPerInstance     *atomic_map.AtomicMapCounter[*UpstreamEndpoint]
	FAInstanceQueueLength       *atomic_map.AtomicMapCounter[*UpstreamEndpoint]
}

type FunctionMetadata struct {
	sync.RWMutex

	uniqueIdentifier string

	identifier         string
	sandboxParallelism uint

	upstreamEndpoints      []*UpstreamEndpoint
	upstreamEndpointsCount int32
	queue                  *list.List

	loadBalancingMetadata LoadBalancingMetadata

	beingDrained *chan struct{} // TODO: implement this feature
	metrics      ScalingMetric

	autoscalingTriggered bool
}

type ScalingMetric struct {
	timestamp      time.Time
	timeWindowSize time.Duration

	totalRequests    uint32
	inflightRequests int32
}

func NewFunctionMetadata(name string) *FunctionMetadata {
	hostName, err := os.Hostname()
	if err != nil {
		logrus.Warn("Error fetching host name.")
	}

	return &FunctionMetadata{
		uniqueIdentifier:   hostName,
		identifier:         name,
		sandboxParallelism: 1, // TODO: make dynamic
		queue:              list.New(),
		metrics: ScalingMetric{
			timeWindowSize: 2 * time.Second,
		},
		loadBalancingMetadata: LoadBalancingMetadata{
			RoundRobinCounter:           0,
			KubernetesRoundRobinCounter: 0,
			RequestCountPerInstance:     atomic_map.NewAtomicMapCounter[*UpstreamEndpoint](),
			FAInstanceQueueLength:       atomic_map.NewAtomicMapCounter[*UpstreamEndpoint](),
		},
	}
}

func (m *FunctionMetadata) GetUpstreamEndpoints() []*UpstreamEndpoint {
	return m.upstreamEndpoints
}

func (m *FunctionMetadata) GetSandboxParallelism() uint {
	return m.sandboxParallelism
}

func (m *FunctionMetadata) GetRoundRobinCounter() uint32 {
	return m.loadBalancingMetadata.RoundRobinCounter % uint32(len(m.GetUpstreamEndpoints()))
}

func (m *FunctionMetadata) IncrementRoundRobinCounter() {
	atomic.AddUint32(&m.loadBalancingMetadata.RoundRobinCounter, 1)
}

func (m *FunctionMetadata) GetKubernetesRoundRobinCounter() uint32 {
	return m.loadBalancingMetadata.KubernetesRoundRobinCounter % uint32(len(m.GetUpstreamEndpoints()))
}

func (m *FunctionMetadata) IncrementKubernetesRoundRobinCounter() {
	atomic.AddUint32(&m.loadBalancingMetadata.KubernetesRoundRobinCounter, 1)
}

func (m *FunctionMetadata) GetRequestCountPerInstance() *atomic_map.AtomicMapCounter[*UpstreamEndpoint] {
	return m.loadBalancingMetadata.RequestCountPerInstance
}

func (m *FunctionMetadata) UpdateRequestMetadata(endpoint *UpstreamEndpoint) {
	m.loadBalancingMetadata.RequestCountPerInstance.AtomicIncrement(endpoint)
}

func (m *FunctionMetadata) GetLocalQueueLength(endpoint *UpstreamEndpoint) int64 {
	return m.loadBalancingMetadata.RequestCountPerInstance.Get(endpoint)
}

func (m *FunctionMetadata) IncrementLocalQueueLength(endpoint *UpstreamEndpoint) {
	m.loadBalancingMetadata.FAInstanceQueueLength.AtomicIncrement(endpoint)
}

func (m *FunctionMetadata) DecrementLocalQueueLength(endpoint *UpstreamEndpoint) {
	m.loadBalancingMetadata.FAInstanceQueueLength.AtomicDecrement(endpoint)
}

func createThrottlerChannel(capacity uint) RequestThrottler {
	ccChannel := make(chan struct{}, capacity)

	for cc := 0; uint(cc) < capacity; cc++ {
		ccChannel <- struct{}{}
	}

	return ccChannel
}

func (m *FunctionMetadata) updateEndpointList(data []*proto.EndpointInfo) {
	oldURLs, mmOld := _map.ExtractField[*UpstreamEndpoint](m.upstreamEndpoints, func(info *UpstreamEndpoint) string { return info.URL })
	newURLs, mmNew := _map.ExtractField[*proto.EndpointInfo](data, func(info *proto.EndpointInfo) string { return info.URL })

	toAdd := _map.Difference(newURLs, oldURLs)
	toRemove := _map.Difference(oldURLs, newURLs)

	for i := 0; i < len(toAdd); i++ {
		m.upstreamEndpoints = append(m.upstreamEndpoints, &UpstreamEndpoint{
			ID:       mmNew[toAdd[i]].ID,
			URL:      mmNew[toAdd[i]].URL,
			Capacity: createThrottlerChannel(m.sandboxParallelism),
		})
	}

	for i := 0; i < len(toRemove); i++ {
		for j := 0; j < len(m.upstreamEndpoints); j++ {
			if mmOld[toRemove[i]].URL == m.upstreamEndpoints[j].URL {
				m.upstreamEndpoints = append(m.upstreamEndpoints[:j], m.upstreamEndpoints[j+1:]...)
				break
			}
		}
	}
}

func (m *FunctionMetadata) SetUpstreamURLs(endpoints []*proto.EndpointInfo) error {
	m.Lock()
	defer m.Unlock()

	logrus.Debug("Updated endpoint list for ", m.identifier)
	m.updateEndpointList(endpoints)
	atomic.StoreInt32(&m.upstreamEndpointsCount, int32(len(m.upstreamEndpoints)))

	if len(m.upstreamEndpoints) > 0 {
		dequeueCnt := 0

		for m.queue.Len() > 0 {
			dequeue := m.queue.Front()
			m.queue.Remove(dequeue)

			signal, ok := dequeue.Value.(chan struct{})
			if !ok {
				return errors.New("Failed to convert dequeue into channel")
			}

			signal <- struct{}{}
			close(signal)

			dequeueCnt++
		}

		if dequeueCnt > 0 {
			logrus.Debug("Dequeued ", dequeueCnt, " requests for ", m.identifier)
		}
	}

	return nil
}

func (m *FunctionMetadata) dequeueRequests() {
	dequeueCnt := 0

	for m.queue.Len() > 0 {
		dequeue := m.queue.Front()
		m.queue.Remove(dequeue)

		signal, ok := dequeue.Value.(chan struct{})
		if !ok {
			logrus.Fatal("Failed to convert dequeue into channel")
		}

		// unblock proxy request
		signal <- struct{}{}
		close(signal)

		dequeueCnt++
	}

	if dequeueCnt > 0 {
		logrus.Debug("Dequeued ", dequeueCnt, " requests for ", m.identifier)
	}
}

func (m *FunctionMetadata) SetEndpoints(newEndpoints []*UpstreamEndpoint) {
	m.upstreamEndpoints = newEndpoints
}

func (m *FunctionMetadata) DecreaseInflight() {
	atomic.AddInt32(&m.metrics.inflightRequests, -1)
}

func (m *FunctionMetadata) TryWarmStart(cp *proto.CpiInterfaceClient) (chan struct{}, time.Duration) {
	start := time.Now()

	// autoscaling metric
	atomic.AddInt32(&m.metrics.inflightRequests, 1)
	// runtime statistics
	atomic.AddUint32(&m.metrics.totalRequests, 1)

	endpointCount := atomic.LoadInt32(&m.upstreamEndpointsCount)
	if endpointCount == 0 {
		waitChannel := make(chan time.Duration, 1)

		m.Lock()
		defer m.Unlock()

		m.queue.PushBack(waitChannel)

		// trigger autoscaling
		if !m.autoscalingTriggered {
			m.autoscalingTriggered = true
			m.metrics.timestamp = time.Now()

			go m.sendMetricsToAutoscaler(cp)
		}

		return waitChannel, time.Since(start)
	} else {
		return nil, 0 // assume 0 for warm starts
	}
}

func (m *FunctionMetadata) sendMetricsToAutoscaler(cp *proto.CpiInterfaceClient) {
	timer := time.NewTicker(m.metrics.timeWindowSize)

	logrus.Debug("Started metrics loop")

	for ; true; <-timer.C {
		logrus.Debug("Timer ticked.")

		m.Lock()

		inflightRequests := atomic.LoadInt32(&m.metrics.inflightRequests)

		go func() {
			status, err := (*cp).OnMetricsReceive(context.Background(), &proto.AutoscalingMetric{
				ServiceName:      m.identifier,
				DataplaneName:    m.uniqueIdentifier,
				InflightRequests: inflightRequests,
			})
			if err != nil || !status.Success {
				logrus.Warn("Failed to forward metrics to the control plane for service '", m.identifier, "'")
			}

			logrus.Debug("Metric value of ", inflightRequests, " was reported for ", m.identifier)
		}()

		toBreak := inflightRequests == 0
		if toBreak {
			m.autoscalingTriggered = false
		}

		m.Unlock()

		if toBreak {
			logrus.Debug("Metrics loop exited")
			break
		}
	}
}

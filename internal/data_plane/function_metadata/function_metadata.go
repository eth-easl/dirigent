package function_metadata

import (
	"cluster_manager/api/proto"
	"cluster_manager/pkg/atomic_map"
	"container/list"
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	UNLIMITED_CONCURENCY uint = 0
)

type DataPlaneConnectionInfo struct {
	Iface     proto.DpiInterfaceClient
	IP        string
	APIPort   string
	ProxyPort string
}

type RequestThrottler chan struct{}

type UpstreamEndpoint struct {
	ID       string
	Capacity RequestThrottler
	URL      string
}

type LoadBalancingMetadata struct {
	RoundRobinCounter           uint32
	KubernetesRoundRobinCounter uint32
	RequestCountPerInstance     atomic_map.AtomicMap[*UpstreamEndpoint]
	FAInstanceQueueLength       atomic_map.AtomicMap[*UpstreamEndpoint]
}

type FunctionMetadata struct {
	sync.RWMutex

	identifier         string
	sandboxParallelism uint

	upstreamEndpoints      []*UpstreamEndpoint
	upstreamEndpointsCount int32
	queue                  *list.List

	loadBalancingMetadata LoadBalancingMetadata

	beingDrained   *chan struct{} // TODO: implement this feature
	metrics        ScalingMetric
	coldStartDelay time.Duration

	autoscalingTriggered bool
}

type ScalingMetric struct {
	timestamp      time.Time
	timeWindowSize time.Duration

	totalRequests    int32
	inflightRequests int32
}

func NewFunctionMetadata(name string) *FunctionMetadata {
	return &FunctionMetadata{
		identifier:         name,
		sandboxParallelism: 1, // TODO: make dynamic
		queue:              list.New(),
		metrics: ScalingMetric{
			timeWindowSize: 2 * time.Second,
		},
		coldStartDelay: 50 * time.Millisecond, // TODO: implement readiness probing
		loadBalancingMetadata: LoadBalancingMetadata{
			RoundRobinCounter:           0,
			KubernetesRoundRobinCounter: 0,
			RequestCountPerInstance:     atomic_map.NewAtomicMap[*UpstreamEndpoint](),
			FAInstanceQueueLength:       atomic_map.NewAtomicMap[*UpstreamEndpoint](),
		},
	}
}

func (m *FunctionMetadata) GetColdStartDelay() time.Duration {
	return m.coldStartDelay
}

// TODO: Should we lock it or maybe try to find a better synchronization primitive?
var upstreamEndpointsLock sync.Mutex

// TODO: This lock is really strange. It locks only reads...... Investigate this
func (m *FunctionMetadata) GetUpstreamEndpoints() []*UpstreamEndpoint {
	upstreamEndpointsLock.Lock()
	defer upstreamEndpointsLock.Unlock()

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

func (m *FunctionMetadata) GetRequestCountPerInstance() *atomic_map.AtomicMap[*UpstreamEndpoint] {
	return &m.loadBalancingMetadata.RequestCountPerInstance
}

func (m *FunctionMetadata) UpdateRequestMetadata(endpoint *UpstreamEndpoint) {
	m.loadBalancingMetadata.RequestCountPerInstance.AtomicIncrement(endpoint)
}

func (m *FunctionMetadata) GetLocalQueueLength(endpoint *UpstreamEndpoint) int64 {
	return m.loadBalancingMetadata.RequestCountPerInstance.AtomicGet(endpoint)
}

func (m *FunctionMetadata) IncrementLocalQueueLength(endpoint *UpstreamEndpoint) {
	m.loadBalancingMetadata.FAInstanceQueueLength.AtomicIncrement(endpoint)
}

func (m *FunctionMetadata) DecrementLocalQueueLength(endpoint *UpstreamEndpoint) {
	m.loadBalancingMetadata.FAInstanceQueueLength.AtomicDecrement(endpoint)
}

func difference(a, b []string) []string {
	mb := make(map[string]struct{}, len(b))
	for _, x := range b {
		mb[x] = struct{}{}
	}

	var diff []string

	for _, x := range a {
		if _, found := mb[x]; !found {
			diff = append(diff, x)
		}
	}

	return diff
}

func extractField[T any](m []T, extractor func(T) string) ([]string, map[string]T) {
	var res []string

	mm := make(map[string]T)

	for i := 0; i < len(m); i++ {
		val := extractor(m[i])

		res = append(res, val)
		mm[val] = m[i]
	}

	return res, mm
}

func createThrottlerChannel(capacity uint) RequestThrottler {
	ccChannel := make(chan struct{}, capacity)

	for cc := 0; uint(cc) < capacity; cc++ {
		ccChannel <- struct{}{}
	}

	return ccChannel
}

func (m *FunctionMetadata) updateEndpointList(data []*proto.EndpointInfo) {
	oldURLs, mmOld := extractField[*UpstreamEndpoint](m.upstreamEndpoints, func(info *UpstreamEndpoint) string { return info.URL })
	newURLs, mmNew := extractField[*proto.EndpointInfo](data, func(info *proto.EndpointInfo) string { return info.URL })

	toAdd := difference(newURLs, oldURLs)
	toRemove := difference(oldURLs, newURLs)

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
	atomic.AddInt32(&m.metrics.totalRequests, 1)

	endpointCount := atomic.LoadInt32(&m.upstreamEndpointsCount)
	if endpointCount == 0 {
		waitChannel := make(chan struct{}, 1)

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
				ServiceName: m.identifier,
				Metric:      float32(inflightRequests),
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

type Deployments struct {
	data map[string]*FunctionMetadata
	sync.RWMutex
}

func NewDeploymentList() *Deployments {
	return &Deployments{
		data: make(map[string]*FunctionMetadata),
	}
}

func (d *Deployments) AddDeployment(name string) bool {
	d.Lock()
	defer d.Unlock()

	if m, ok := d.data[name]; ok {
		if m.beingDrained != nil {
			logrus.Warn("Failed registering a deployment. The deployment exists and is being drained.")
		} else {
			logrus.Warn("Failed registering a deployment. Name already taken.")
		}

		return false
	}

	d.data[name] = NewFunctionMetadata(name)

	logrus.Info("Service with name '", name, "' has been registered")

	return true
}

func (d *Deployments) GetDeployment(name string) (*FunctionMetadata, time.Duration) {
	start := time.Now()

	d.RLock()
	defer d.RUnlock()

	data, ok := d.data[name]

	if !ok || (ok && data.beingDrained != nil) {
		return nil, time.Since(start)
	} else {
		return data, time.Since(start)
	}
}

func (d *Deployments) DeleteDeployment(name string) bool {
	d.Lock()
	defer d.Unlock()

	metadata, ok := d.data[name]
	if ok {
		// deployment cannot be modified while there is at least one
		// HTTP request associated with that deployment
		metadata.Lock()
		defer metadata.Unlock()

		if metadata.beingDrained != nil {
			<-*metadata.beingDrained
		}

		delete(d.data, name)

		return true
	}

	return false
}

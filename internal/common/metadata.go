package common

import (
	"cluster_manager/api/proto"
	"container/list"
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
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

type FunctionMetadata struct {
	sync.RWMutex

	identifier         string
	sandboxParallelism int

	upstreamEndpoints      []*UpstreamEndpoint
	upstreamEndpointsCount int32
	queue                  *list.List

	requestCountPerInstance *map[*UpstreamEndpoint]int64

	beingDrained   *chan struct{} // TODO: implement this feature
	metrics        ScalingMetric
	coldStartDelay time.Duration

	sendMetricsTriggered bool
}

type ScalingMetric struct {
	timestamp      time.Time
	timeWindowSize time.Duration

	totalRequests    int32
	inflightRequests int32
}

func NewFunctionMetadata(name string) *FunctionMetadata {
	requestPerInstance := make(map[*UpstreamEndpoint]int64)

	return &FunctionMetadata{
		identifier:         name,
		sandboxParallelism: 1, // TODO: make dynamic
		queue:              list.New(),
		metrics: ScalingMetric{
			timeWindowSize: 2 * time.Second,
		},
		coldStartDelay:          50 * time.Millisecond, // TODO: implement readiness probing
		requestCountPerInstance: &requestPerInstance,
	}
}

func (m *FunctionMetadata) GetColdStartDelay() time.Duration {
	return m.coldStartDelay
}

func (m *FunctionMetadata) GetUpstreamEndpoints() []*UpstreamEndpoint {
	return m.upstreamEndpoints
}

func (m *FunctionMetadata) GetRequestCountPerInstance() *map[*UpstreamEndpoint]int64 {
	return m.requestCountPerInstance
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

func createThrottlerChannel(capacity int) RequestThrottler {
	ccChannel := make(chan struct{}, capacity)
	for cc := 0; cc < capacity; cc++ {
		ccChannel <- struct{}{}
	}

	return ccChannel
}

func (m *FunctionMetadata) mergeEndpointList(data []*proto.EndpointInfo) {
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
	m.mergeEndpointList(endpoints)
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
		if !m.sendMetricsTriggered {
			m.sendMetricsTriggered = true
			m.metrics.timestamp = time.Now()

			go m.sendMetricsLoop(cp)
		}

		return waitChannel, time.Since(start)
	} else {
		return nil, 0 // assume 0 for warm starts
	}
}

func (m *FunctionMetadata) sendMetricsLoop(cp *proto.CpiInterfaceClient) {
	timer := time.NewTicker(m.metrics.timeWindowSize)

	logrus.Debug("Started metrics loop")

	for ; true; <-timer.C {
		logrus.Debug("Timer ticked.")

		m.Lock()

		metricValue := atomic.LoadInt32(&m.metrics.inflightRequests)

		go func() {
			status, err := (*cp).OnMetricsReceive(context.Background(), &proto.AutoscalingMetric{
				ServiceName: m.identifier,
				Metric:      float32(metricValue),
			})
			if err != nil || !status.Success {
				logrus.Warn("Failed to forward metrics to the control plane for service '", m.identifier, "'")
			}

			logrus.Debug("Metric value of ", metricValue, " was reported for ", m.identifier)
		}()

		toBreak := metricValue == 0
		if toBreak {
			m.sendMetricsTriggered = false
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

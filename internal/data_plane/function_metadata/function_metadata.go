/*
 * MIT License
 *
 * Copyright (c) 2024 EASL
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */

package function_metadata

import (
	"cluster_manager/api/proto"
	"cluster_manager/pkg/atomic_map"
	_map "cluster_manager/pkg/map"
	"container/list"
	"context"
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
	// Zero equals false; Everything else is true
	Drained          int32
	DrainingCallback chan struct{}
	InFlight         int32
}

type LoadBalancingMetadata struct {
	RoundRobinCounter           uint32
	KubernetesRoundRobinCounter uint32
	RequestCountPerInstance     *atomic_map.AtomicMapCounter[*UpstreamEndpoint]
}

type FunctionMetadata struct {
	sync.RWMutex

	dataPlaneID string

	identifier         string
	sandboxParallelism uint

	upstreamEndpoints []*UpstreamEndpoint
	// Endpoint is active if it is not being drained.
	activeEndpointCount int32
	queue               *list.List

	loadBalancingMetadata LoadBalancingMetadata

	scalingMetric ScalingMetric
	statistics    *FunctionStatistics

	// autoscalingTriggered - 0 = not running; other = running
	autoscalingTriggered int32
}

type ScalingMetric struct {
	timestamp      time.Time
	timeWindowSize time.Duration

	inflightRequests int32
}

type ColdStartOutcome int

const (
	SuccessfulColdStart ColdStartOutcome = iota
	CanceledColdStart
)

type ColdStartChannelStruct struct {
	Outcome             ColdStartOutcome
	AddEndpointDuration time.Duration
}

func NewFunctionMetadata(name string, dataplaneID string) *FunctionMetadata {
	return &FunctionMetadata{
		dataPlaneID:        dataplaneID,
		identifier:         name,
		sandboxParallelism: 1, // TODO: make dynamic
		queue:              list.New(),
		scalingMetric: ScalingMetric{
			timeWindowSize: 2 * time.Second,
		},
		statistics: &FunctionStatistics{},
		loadBalancingMetadata: LoadBalancingMetadata{
			RoundRobinCounter:           0,
			KubernetesRoundRobinCounter: 0,
			RequestCountPerInstance:     atomic_map.NewAtomicMapCounter[*UpstreamEndpoint](),
		},
	}
}

func (m *FunctionMetadata) GetStatistics() *FunctionStatistics {
	return m.statistics
}

func (m *FunctionMetadata) GetIdentifier() string {
	return m.identifier
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

func (m *FunctionMetadata) IncrementRequestCountPerInstance(endpoint *UpstreamEndpoint) {
	m.loadBalancingMetadata.RequestCountPerInstance.AtomicIncrement(endpoint)
}

func (m *FunctionMetadata) GetLocalQueueLength(endpoint *UpstreamEndpoint) int64 {
	return m.loadBalancingMetadata.RequestCountPerInstance.Get(endpoint)
}

func createThrottlerChannel(capacity uint) RequestThrottler {
	ccChannel := make(chan struct{}, capacity)

	for cc := 0; uint(cc) < capacity; cc++ {
		ccChannel <- struct{}{}
	}

	return ccChannel
}

func endpointDelta(oldEndpoints []*UpstreamEndpoint, newEndpoints []*proto.EndpointInfo) (map[string]*UpstreamEndpoint, map[string]*proto.EndpointInfo, []string, []string) {
	oldURLs, mmOld := _map.ExtractField[*UpstreamEndpoint](oldEndpoints, func(info *UpstreamEndpoint) string { return info.URL })
	newURLs, mmNew := _map.ExtractField[*proto.EndpointInfo](newEndpoints, func(info *proto.EndpointInfo) string { return info.URL })

	toAdd := _map.Difference(newURLs, oldURLs)
	toRemove := _map.Difference(oldURLs, newURLs)

	return mmOld, mmNew, toAdd, toRemove
}

func (m *FunctionMetadata) addToEndpointList(data []*proto.EndpointInfo) {
	for i := 0; i < len(data); i++ {
		m.upstreamEndpoints = append(m.upstreamEndpoints, &UpstreamEndpoint{
			ID:       data[i].ID,
			URL:      data[i].URL,
			Capacity: createThrottlerChannel(m.sandboxParallelism),
			Drained:  0, // not in drain mode
			InFlight: 0,
		})
	}

	if len(m.upstreamEndpoints) > 1 {
		logrus.Errorf("ASSERTION - UPSTREAM ENDPOINTS > 1")
	}
}

func (m *FunctionMetadata) AddEndpoints(endpoints []*proto.EndpointInfo) {
	timeToAddEndpoint := time.Now()

	m.Lock()
	defer m.Unlock()

	m.addToEndpointList(endpoints)
	atomic.StoreInt32(&m.activeEndpointCount, int32(len(m.upstreamEndpoints)))
	logrus.Debugf("Added %d endpoint(s) for %s.", len(endpoints), m.identifier)

	if len(m.upstreamEndpoints) > 0 {
		dequeueCnt := 0

		for m.queue.Len() > 0 {
			dequeue := m.queue.Front()
			m.queue.Remove(dequeue)

			// ColdStartOutcome channel shouldn't be closed because of context termination listener
			signal, _ := dequeue.Value.(chan ColdStartChannelStruct)
			signal <- ColdStartChannelStruct{
				Outcome:             SuccessfulColdStart,
				AddEndpointDuration: time.Since(timeToAddEndpoint),
			}

			dequeueCnt++
		}

		if dequeueCnt > 0 {
			logrus.Debug("Dequeued ", dequeueCnt, " requests for ", m.identifier)
		}
	}
}

func (m *FunctionMetadata) DrainEndpoints(endpoints []*proto.EndpointInfo) error {
	m.Lock()
	barriers := m.markEndpointsAsDraining(endpoints)
	m.removeEndpoints(endpoints)
	m.Unlock()

	m.waitForDrainingToComplete(barriers)

	return nil
}

func (m *FunctionMetadata) RemoveAllEndpoints() {
	m.Lock()
	defer m.Unlock()

	m.upstreamEndpoints = nil
	atomic.StoreInt32(&m.activeEndpointCount, 0)
}

func (m *FunctionMetadata) removeEndpoints(endpoints []*proto.EndpointInfo) {
	for i := 0; i < len(endpoints); i++ {
		for j := 0; j < len(m.upstreamEndpoints); j++ {
			if endpoints[i].URL == m.upstreamEndpoints[j].URL {
				m.upstreamEndpoints = append(m.upstreamEndpoints[:j], m.upstreamEndpoints[j+1:]...)
				break
			}
		}
	}

	atomic.StoreInt32(&m.activeEndpointCount, int32(len(m.upstreamEndpoints)))

	if len(m.upstreamEndpoints) != 0 {
		logrus.Errorf("ASSERTION - DRAIN - UPSTREAM ENDPOINTS != 0")
	}
}

func (m *FunctionMetadata) markEndpointsAsDraining(endpoints []*proto.EndpointInfo) []chan struct{} {
	infos := make(map[string]*proto.EndpointInfo)
	for _, e := range endpoints {
		infos[e.ID] = e
	}
	///////////////////////////////////////////////
	var callbackBlocks []chan struct{}

	count := 0
	for _, endpoint := range m.upstreamEndpoints {
		if _, ok := infos[endpoint.ID]; ok {
			swapped := false
			var oldValue int32
			for !swapped {
				oldValue = atomic.LoadInt32(&endpoint.Drained)
				swapped = atomic.CompareAndSwapInt32(&endpoint.Drained, oldValue, 1)
			}

			if swapped && oldValue == 0 {
				endpoint.DrainingCallback = make(chan struct{}, 1)
				callbackBlocks = append(callbackBlocks, endpoint.DrainingCallback)

				if atomic.LoadInt32(&endpoint.InFlight) == 0 {
					endpoint.DrainingCallback <- struct{}{}
				}
			}
		} else {
			// counting endpoints that are not in drain mode
			count++
		}
	}

	return callbackBlocks
}

func (m *FunctionMetadata) waitForDrainingToComplete(barriers []chan struct{}) {
	if len(barriers) == 0 {
		return
	}

	wg := sync.WaitGroup{}
	wg.Add(len(barriers))

	for i := 0; i < len(barriers); i++ {
		go func(endpointBarrier chan struct{}) {
			defer wg.Done()

			<-endpointBarrier
		}(barriers[i])
	}

	wg.Wait()
}

func (m *FunctionMetadata) SetEndpoints(newEndpoints []*UpstreamEndpoint) {
	m.upstreamEndpoints = newEndpoints
}

func (m *FunctionMetadata) DecreaseInflight() {
	atomic.AddInt32(&m.scalingMetric.inflightRequests, -1)
}

func (m *FunctionMetadata) TryWarmStart(cp proto.CpiInterfaceClient) (chan ColdStartChannelStruct, time.Duration) {
	start := time.Now()

	// autoscaling metric
	atomic.AddInt32(&m.scalingMetric.inflightRequests, 1)
	m.statistics.IncrementInflight()

	m.triggerAutoscaling(cp)

	if atomic.LoadInt32(&m.activeEndpointCount) == 0 {
		waitChannel := make(chan ColdStartChannelStruct, 1)

		m.Lock()
		defer m.Unlock()

		m.queue.PushBack(waitChannel)
		m.statistics.IncrementQueueDepth()

		return waitChannel, time.Since(start)
	} else {
		return nil, 0 // assume 0 for warm starts
	}
}

func (m *FunctionMetadata) triggerAutoscaling(cp proto.CpiInterfaceClient) {
	swapped := false
	for !swapped {
		oldValue := atomic.LoadInt32(&m.autoscalingTriggered)
		if oldValue == 0 {
			swapped = atomic.CompareAndSwapInt32(&m.autoscalingTriggered, oldValue, 1)
		} else {
			break
		}
	}

	if swapped {
		m.scalingMetric.timestamp = time.Now()
		go m.sendMetricsToAutoscaler(cp)
	}
}

func (m *FunctionMetadata) sendMetricsToAutoscaler(cp proto.CpiInterfaceClient) {
	timer := time.NewTicker(m.scalingMetric.timeWindowSize)

	logrus.Debug("Started metrics loop")

	for ; true; <-timer.C {
		m.Lock()

		inflightRequests := atomic.LoadInt32(&m.scalingMetric.inflightRequests)

		go func() {
			// TODO: NEED TO IMPLEMENT - the data plane shouldn't stop send metrics until the control plane at least once confirms
			status, err := cp.OnMetricsReceive(context.Background(), &proto.AutoscalingMetric{
				ServiceName:      m.identifier,
				DataplaneName:    m.dataPlaneID,
				InflightRequests: inflightRequests,
			})
			if err != nil || !status.Success {
				logrus.Warn("Failed to forward scaling metric to the control plane for service '", m.identifier, "'")
			}

			logrus.Debug("Metric value of ", inflightRequests, " was reported for ", m.identifier)
		}()

		toBreak := inflightRequests == 0
		if toBreak {
			atomic.StoreInt32(&m.autoscalingTriggered, 0)
		}

		m.Unlock()

		if toBreak {
			logrus.Debug("Metrics loop exited")
			break
		}
	}
}

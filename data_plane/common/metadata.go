package common

import (
	"cluster_manager/api/proto"
	"container/list"
	"context"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/semaphore"
	"sync"
	"sync/atomic"
	"time"
)

type UpstreamEndpoint struct {
	Capacity *semaphore.Weighted
	URL      string
}

type FunctionMetadata struct {
	sync.RWMutex

	identifier         string
	sandboxParallelism int

	upstreamEndpoints []UpstreamEndpoint
	queue             *list.List

	beingDrained     *chan struct{}
	inflightRequests int32

	scaleFromZeroSentAt time.Time
}

func (m *FunctionMetadata) GetUpstreamEndpoints() []UpstreamEndpoint {
	return m.upstreamEndpoints
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

func (m *FunctionMetadata) getAllUrls() []string {
	var res []string
	for i := 0; i < len(m.upstreamEndpoints); i++ {
		res = append(res, m.upstreamEndpoints[i].URL)
	}

	return res
}

func (m *FunctionMetadata) mergeEndpointList(newURLs []string) {
	oldURLs := m.getAllUrls()

	toAdd := difference(newURLs, oldURLs)
	toRemove := difference(oldURLs, newURLs)

	for i := 0; i < len(toAdd); i++ {
		m.upstreamEndpoints = append(m.upstreamEndpoints, UpstreamEndpoint{
			URL:      toAdd[i],
			Capacity: semaphore.NewWeighted(int64(m.sandboxParallelism)),
		})
	}

	for i := 0; i < len(toRemove); i++ {
		for j := 0; j < len(m.upstreamEndpoints); j++ {
			if toRemove[i] == m.upstreamEndpoints[j].URL {
				m.upstreamEndpoints = append(m.upstreamEndpoints[:j], m.upstreamEndpoints[j+1:]...)
				break
			}
		}
	}
}

func (m *FunctionMetadata) SetUpstreamURLs(urls []string) {
	m.Lock()
	defer m.Unlock()

	logrus.Debug("Updated endpoint list for ", m.identifier)
	m.mergeEndpointList(urls)

	if len(m.upstreamEndpoints) > 0 {
		dequeueCnt := 0

		for m.queue.Len() > 0 {
			dequeue := m.queue.Front()
			m.queue.Remove(dequeue)

			signal := dequeue.Value.(chan bool)
			signal <- true
			close(signal)

			dequeueCnt++
		}

		if dequeueCnt > 0 {
			logrus.Debug("Dequeued ", dequeueCnt, " requests for ", m.identifier)
		}
	}
}

func (m *FunctionMetadata) DecreaseInflight() {
	newVal := atomic.AddInt32(&m.inflightRequests, -1)

	if newVal == 0 {
		m.Lock()
		defer m.Unlock()

		if m.beingDrained != nil {
			*m.beingDrained <- struct{}{}
		}
	}
}

func (m *FunctionMetadata) TryWarmStart(cp *proto.CpiInterfaceClient) (chan bool, func()) {
	atomic.AddInt32(&m.inflightRequests, 1)

	m.Lock()
	defer m.Unlock()

	if m.upstreamEndpoints == nil || len(m.upstreamEndpoints) == 0 {
		waitChannel := make(chan bool, 1)
		m.queue.PushBack(waitChannel)

		var requestScaling = func() {}

		if m.scaleFromZeroSentAt.IsZero() || time.Since(m.scaleFromZeroSentAt) > 5*time.Second {
			m.scaleFromZeroSentAt = time.Now()

			requestScaling = func() {
				status, err := (*cp).ScaleFromZero(context.Background(), &proto.DeploymentName{Name: m.identifier})
				if err != nil || !status.Success {
					logrus.Warn("Scale from zero failed for function ", m.identifier)
				} else {
					logrus.Debug("Scale from zero request issued for ", m.identifier)
				}
			}
		}

		return waitChannel, requestScaling
	} else {
		return nil, func() {}
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

	d.data[name] = &FunctionMetadata{
		identifier:         name,
		sandboxParallelism: 1,
		queue:              list.New(),
	}

	logrus.Info("Service with name '", name, "' has been registered")

	return true
}

func (d *Deployments) GetDeployment(name string) *FunctionMetadata {
	d.RLock()
	defer d.RUnlock()

	data, ok := d.data[name]

	if !ok || (ok && data.beingDrained != nil) {
		return nil
	} else {
		return data
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

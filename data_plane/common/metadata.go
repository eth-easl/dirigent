package common

import (
	"container/list"
	"errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/semaphore"
	"sync"
)

type UpstreamEndpoint struct {
	URL      string
	Capacity *semaphore.Weighted
}

type FunctionMetadata struct {
	identifier         string
	sandboxParallelism int
	upstreamEndpoints  []UpstreamEndpoint

	queue *list.List
	mutex sync.Mutex
}

func (m *FunctionMetadata) GetUpstreamEndpoints() []UpstreamEndpoint {
	m.mutex.Lock()
	defer m.mutex.Unlock()

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
	m.mutex.Lock()
	defer m.mutex.Unlock()

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

func (m *FunctionMetadata) TryWarmStart() chan bool {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.upstreamEndpoints == nil || len(m.upstreamEndpoints) == 0 {
		waitChannel := make(chan bool, 1)
		m.queue.PushBack(waitChannel)

		return waitChannel
	} else {
		return nil
	}
}

type Deployments struct {
	data  map[string]*FunctionMetadata
	mutex sync.RWMutex
}

func NewDeploymentList() *Deployments {
	return &Deployments{
		data:  make(map[string]*FunctionMetadata),
		mutex: sync.RWMutex{},
	}
}

func (d *Deployments) AddDeployment(name string) error {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if _, ok := d.data[name]; ok {
		return errors.New("Failed registering a deployment. Name already taken.")
	}

	d.data[name] = &FunctionMetadata{
		identifier:         name,
		sandboxParallelism: 1,
		queue:              list.New(),
	}

	return nil
}

func (d *Deployments) GetDeployment(name string) *FunctionMetadata {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	data, ok := d.data[name]
	if !ok {
		return nil
	} else {
		return data
	}
}

func (d *Deployments) DeleteDeployment(name string) bool {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	data, ok := d.data[name]
	if ok {
		data.mutex.Lock()
		defer data.mutex.Unlock()

		// TODO: unsure about race conditions here

		delete(d.data, name)

		return true
	}

	return false
}

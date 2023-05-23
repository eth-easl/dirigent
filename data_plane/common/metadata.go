package common

import (
	"container/list"
	"errors"
	"github.com/sirupsen/logrus"
	"sync"
)

type FunctionMetadata struct {
	identifier  string
	upstreamURL []string

	queue *list.List
	mutex sync.Mutex
}

func (m *FunctionMetadata) SetUpstreamURL(urls []string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	logrus.Debug("Updated endpoint list for ", m.identifier)
	m.upstreamURL = urls

	if len(m.upstreamURL) > 0 {
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

	if m.upstreamURL == nil || len(m.upstreamURL) == 0 {
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
		identifier: name,
		queue:      list.New(),
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

		delete(d.data, name)

		return true
	}

	return false
}

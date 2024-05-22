package function_metadata

import (
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

type Deployments struct {
	data map[string]*FunctionMetadata
	sync.RWMutex
}

func NewDeploymentList() *Deployments {
	return &Deployments{
		data: make(map[string]*FunctionMetadata),
	}
}

func (d *Deployments) AddDeployment(name string, dataplaneID string) bool {
	d.Lock()
	defer d.Unlock()

	if _, ok := d.data[name]; ok {
		logrus.Errorf("Failed registering a deployment %s. Name already taken.", name)
		return false
	}

	// TODO: make container concurrency configurable
	d.data[name] = NewFunctionMetadata(name, dataplaneID, 1)

	logrus.Debugf("Service with name %s has been registered.", name)
	return true
}

func (d *Deployments) GetDeployment(name string) (*FunctionMetadata, time.Duration) {
	start := time.Now()

	d.RLock()
	defer d.RUnlock()

	data, ok := d.data[name]

	if !ok {
		return nil, time.Since(start)
	} else {
		return data, time.Since(start)
	}
}

func (d *Deployments) DeleteDeployment(name string) bool {
	d.Lock()
	defer d.Unlock()

	if _, ok := d.data[name]; ok {
		// TODO: implement draining here

		delete(d.data, name)
		return true
	}

	return false
}

func (d *Deployments) ListDeployments() []*FunctionMetadata {
	d.RLock()
	defer d.RUnlock()

	var result []*FunctionMetadata
	for _, vm := range d.data {
		result = append(result, vm)
	}

	return result
}

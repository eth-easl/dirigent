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

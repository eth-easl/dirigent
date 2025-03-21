package service_metadata

import (
	"cluster_manager/internal/data_plane/workflow"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

type DeploymentType int

const (
	Function DeploymentType = iota
	Task
	Workflow
)

type Deployment struct {
	dType    DeploymentType
	metadata *ServiceMetadata
	wfPtr    *workflow.Workflow
}

func (d *Deployment) GetType() DeploymentType {
	return d.dType
}
func (d *Deployment) GetFunction() *ServiceMetadata {
	return d.metadata
}
func (d *Deployment) GetWorkflow() *workflow.Workflow {
	return d.wfPtr
}

type Deployments struct {
	data map[string]*Deployment
	sync.RWMutex
}

func NewDeploymentList() *Deployments {
	return &Deployments{
		data: make(map[string]*Deployment),
	}
}

func (d *Deployments) AddFunctionDeployment(name string, dataplaneID string) bool {
	d.Lock()
	defer d.Unlock()

	if _, ok := d.data[name]; ok {
		logrus.Errorf("Failed registering a deployment %s. Name already taken.", name)
		return false
	}

	// TODO: make container concurrency configurable
	d.data[name] = &Deployment{
		dType:    Function,
		metadata: NewFunctionMetadata(name, dataplaneID, 1),
	}

	logrus.Debugf("Function deployment with name '%s' has been registered.", name)
	return true
}
func (d *Deployments) AddWorkflowDeployment(name string, dataplaneID string, wf *workflow.Workflow) bool {
	d.Lock()
	defer d.Unlock()

	if _, ok := d.data[name]; ok {
		logrus.Errorf("Failed registering a deployment %s. Name already taken.", name)
		return false
	}

	// TODO: make container concurrency configurable
	for _, task := range wf.Tasks {
		// no need to register separate task for single function without parallelism
		if len(task.Functions) == 1 && !task.IsParallel() {
			continue
		}
		d.data[task.Name] = &Deployment{
			dType:    Task,
			metadata: NewFunctionMetadata(task.Name, dataplaneID, 1),
		}
	}

	d.data[name] = &Deployment{
		dType: Workflow,
		wfPtr: wf,
	}

	logrus.Debugf("Workflow deployment with name '%s' has been registered.", name)
	return true
}

func (d *Deployments) GetDeployment(name string) (*Deployment, time.Duration) {
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
func (d *Deployments) GetServiceMetadata(name string) (*ServiceMetadata, time.Duration) {
	start := time.Now()

	d.RLock()
	defer d.RUnlock()

	data, ok := d.data[name]

	if !ok || data.dType == Workflow {
		return nil, time.Since(start)
	} else {
		return data.metadata, time.Since(start)
	}
}

func (d *Deployments) DeleteDeployment(name string) bool {
	d.Lock()
	defer d.Unlock()

	if deployment, ok := d.data[name]; ok {
		// TODO: implement draining here

		// remove task deployments as well
		if deployment.dType == Workflow {
			for _, task := range deployment.wfPtr.Tasks {
				if len(task.Functions) > 1 {
					delete(d.data, task.Name)
				}
			}
		}

		delete(d.data, name)
		return true
	}

	return false
}

func (d *Deployments) ListDeployments() []*Deployment {
	d.RLock()
	defer d.RUnlock()

	var result []*Deployment
	for _, vm := range d.data {
		result = append(result, vm)
	}

	return result
}
func (d *Deployments) ListFunctions() []*ServiceMetadata {
	d.RLock()
	defer d.RUnlock()

	var result []*ServiceMetadata
	for _, dep := range d.data {
		if dep.dType == Function {
			result = append(result, dep.metadata)
		}
	}

	return result
}

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

	d.data[name] = NewFunctionMetadata(name, dataplaneID)

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

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

package placement_policy

import "github.com/sirupsen/logrus"

const (
	RM_CPU_KEY    = "cpu"
	RM_MEMORY_KEY = "memory"
)

type ResourceMap struct {
	resources map[string]uint64
}

func (r *ResourceMap) GetCPUCores() uint64 {
	return r.resources[RM_CPU_KEY]
}

func (r *ResourceMap) GetMemory() uint64 {
	return r.resources[RM_MEMORY_KEY]
}

func (r *ResourceMap) SetCPUCores(v uint64) {
	r.resources[RM_CPU_KEY] = v
}

func (r *ResourceMap) SetMemory(v uint64) {
	r.resources[RM_MEMORY_KEY] = v
}

func (r *ResourceMap) ResourceKeys() []string {
	keys := make([]string, 0)

	for k := range r.resources {
		keys = append(keys, k)
	}

	return keys
}

func (r *ResourceMap) GetByKey(key string) uint64 {
	if v, ok := r.resources[key]; ok {
		return v
	} else {
		logrus.Fatal("There is no resource with the given key.")
		return 0
	}
}

func (r *ResourceMap) SumAllResourceTypes() uint64 {
	var sum uint64 = 0

	for _, key := range r.ResourceKeys() {
		req := r.GetByKey(key)
		sum += req
	}

	return sum
}

// CreateResourceMap CPU in milliCPUs.
func CreateResourceMap(cpu, memory uint64) *ResourceMap {
	r := &ResourceMap{
		resources: make(map[string]uint64),
	}

	r.SetCPUCores(cpu)
	r.SetMemory(memory)

	return r
}

func ExtendCPU(resourceMap *ResourceMap) *ResourceMap {
	resourceMap.SetCPUCores(resourceMap.GetCPUCores() * 1000)

	return resourceMap
}

func SumResources(a, b *ResourceMap) *ResourceMap {
	return CreateResourceMap(
		a.GetCPUCores()+b.GetCPUCores(),
		a.GetMemory()+b.GetMemory(),
	)
}

func SubtractResources(a, b *ResourceMap) *ResourceMap {
	return CreateResourceMap(
		a.GetCPUCores()-b.GetCPUCores(),
		a.GetMemory()-b.GetMemory(),
	)
}

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

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	mockCpus      uint64 = 4
	mockCpus2     uint64 = 8
	mockMemory    uint64 = 10
	mockMemory2   uint64 = 20
	mockSumCpu    uint64 = 12
	mockSumMemory uint64 = 30
)

func TestResourceMap(t *testing.T) {
	resourceMap := CreateResourceMap(mockCpus, mockMemory)

	assert.Equal(t, resourceMap.GetCPUCores(), mockCpus, "Cpu cores should be the same")
	assert.Equal(t, resourceMap.GetMemory(), mockMemory, "Memory should be the same")

	resourceMap.SetCPUCores(mockCpus2)
	resourceMap.SetMemory(mockMemory2)

	assert.Equal(t, resourceMap.GetCPUCores(), mockCpus2, "Cpu cores should be the same")
	assert.Equal(t, resourceMap.GetMemory(), mockMemory2, "Memory should be the same")

	assert.Equal(t, resourceMap.SumAllResourceTypes(), mockCpus2+mockMemory2)
}

func TestSumResources(t *testing.T) {
	resourceMap1 := CreateResourceMap(mockCpus, mockMemory)
	resourceMap2 := CreateResourceMap(mockCpus2, mockMemory2)
	sumResourceMap := SumResources(resourceMap1, resourceMap2)
	assert.Equal(t, sumResourceMap, CreateResourceMap(mockSumCpu, mockSumMemory))
}

func TestSubtractResource(t *testing.T) {
	resourceMap1 := CreateResourceMap(mockCpus2, mockMemory2)
	resourceMap2 := CreateResourceMap(mockCpus, mockMemory)
	sumResourceMap := SubtractResources(resourceMap1, resourceMap2)
	assert.Equal(t, sumResourceMap, resourceMap2)
}

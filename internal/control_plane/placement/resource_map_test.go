package placement

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	mockCpus      int = 4
	mockCpus2     int = 8
	mockMemory    int = 10
	mockMemory2   int = 20
	mockSumCpu    int = 12
	mockSumMemory int = 30
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

func TestSubstractResource(t *testing.T) {
	resourceMap1 := CreateResourceMap(mockCpus2, mockMemory2)
	resourceMap2 := CreateResourceMap(mockCpus, mockMemory)
	sumResourceMap := SubtractResources(resourceMap1, resourceMap2)
	assert.Equal(t, sumResourceMap, resourceMap2)
}

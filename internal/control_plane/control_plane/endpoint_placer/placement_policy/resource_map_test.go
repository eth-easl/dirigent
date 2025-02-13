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
	mockGpus      uint64 = 1
	mockGpus2     uint64 = 2
	mockSumCpu    uint64 = 12
	mockSumMemory uint64 = 30
	mockSumGpus   uint64 = 3
	mockImage     string = "image"
)

func TestResourceMap(t *testing.T) {
	resourceMap := CreateResourceMap(mockCpus, mockMemory, mockGpus, mockImage)

	assert.Equal(t, resourceMap.GetCpu(), mockCpus, "Cpu cores should be the same")
	assert.Equal(t, resourceMap.GetMemory(), mockMemory, "Memory should be the same")

	resourceMap.SetCpu(mockCpus2)
	resourceMap.SetMemory(mockMemory2)
	resourceMap.SetGpu(mockGpus2)

	assert.Equal(t, resourceMap.GetCpu(), mockCpus2, "Cpu cores should be the same")
	assert.Equal(t, resourceMap.GetMemory(), mockMemory2, "Memory should be the same")
	assert.Equal(t, resourceMap.GetGpu(), mockGpus2, "Gpus should be the same")

	assert.Equal(t, resourceMap.SumAllResourceTypes(), mockCpus2+mockMemory2+mockGpus2)
}

func TestSumResources(t *testing.T) {
	resourceMap1 := CreateResourceMap(mockCpus, mockMemory, mockGpus, mockImage)
	resourceMap2 := CreateResourceMap(mockCpus2, mockMemory2, mockGpus2, mockImage)
	sumResourceMap := SumResources(resourceMap1, resourceMap2)
	assert.Equal(t, sumResourceMap, CreateResourceMap(mockSumCpu, mockSumMemory, mockSumGpus, mockImage))
}

func TestSubtractResource(t *testing.T) {
	resourceMap1 := CreateResourceMap(mockCpus2, mockMemory2, mockGpus2, mockImage)
	resourceMap2 := CreateResourceMap(mockCpus, mockMemory, mockGpus, mockImage)
	sumResourceMap := SubtractResources(resourceMap1, resourceMap2)
	assert.Equal(t, sumResourceMap, resourceMap2)
}

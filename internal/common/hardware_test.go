package common

import (
	"fmt"
	"testing"

	"github.com/pbnjay/memory"
	"github.com/stretchr/testify/assert"
)

func TestGetCpu(t *testing.T) {
	assert.Positive(t, GetNumberCpus())
}

func TestGetMemory(t *testing.T) {
	fmt.Println(memory.TotalMemory())
	assert.Positive(t, GetMemory())
}

func TestCpuPercentage(t *testing.T) {
	assert.True(t, 0 <= getCpuUsage() && getCpuUsage() <= 100)
}

func TestMemoryUsage(t *testing.T) {
	assert.True(t, 0 <= getMemoryUsage() && getMemoryUsage() <= 100)
}

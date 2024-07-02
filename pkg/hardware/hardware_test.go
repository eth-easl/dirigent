package hardware

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetCpu(t *testing.T) {
	assert.Positive(t, GetNumberCpus())
}

func TestGetMemory(t *testing.T) {
	assert.Positive(t, GetMemory())
}

func TestCpuPercentage(t *testing.T) {
	assert.True(t, 0.0 <= getCpuUsage() && getCpuUsage() <= 1.0)
}

func TestMemoryUsage(t *testing.T) {
	assert.True(t, 0.0 <= getMemoryUsage() && getMemoryUsage() <= 1.0)
}

package common

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
	assert.True(t, 0 <= getCpuUsage() && getCpuUsage() <= 100)
}

func TestMemoryUsage(t *testing.T) {
	assert.True(t, 0 <= getMemoryUsage() && getMemoryUsage() <= 100)
}

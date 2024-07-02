package data_plane

import (
	"cluster_manager/internal/control_plane/control_plane/core"
	"cluster_manager/internal/control_plane/control_plane/endpoint_placer/workers"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDataPlaneCreation(t *testing.T) {
	conf := core.WorkerNodeConfiguration{
		Name:   "mockName",
		IP:     "mockIp",
		Port:   "100",
		Cpu:    10000,
		Memory: 10,
	}

	wn := workers.NewWorkerNode(conf)

	assert.Equal(t, wn.GetName(), conf.Name)
	assert.Equal(t, wn.GetIP(), conf.IP)
	assert.Equal(t, wn.GetPort(), conf.Port)
	assert.Equal(t, wn.GetCpuAvailable(), conf.Cpu)
	assert.Equal(t, wn.GetMemoryAvailable(), conf.Memory)

	wn.UpdateLastHearBeat()

	assert.True(t, time.Since(wn.GetLastHeartBeat()) < time.Second)
}

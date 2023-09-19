package data_plane

import (
	"cluster_manager/internal/control_plane/core"
	"cluster_manager/internal/control_plane/workers"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestDataPlaneCreation(t *testing.T) {
	conf := core.WorkerNodeConfiguration{
		Name:     "mockName",
		IP:       "mockIp",
		Port:     "100",
		CpuCores: 10,
		Memory:   10,
	}

	wn := workers.NewWorkerNode(conf)

	assert.Equal(t, wn.GetName(), conf.Name)
	assert.Equal(t, wn.GetIP(), conf.IP)
	assert.Equal(t, wn.GetPort(), conf.Port)
	assert.Equal(t, wn.GetCpuCores(), conf.CpuCores)
	assert.Equal(t, wn.GetMemory(), conf.Memory)

	wn.UpdateLastHearBeat()

	assert.True(t, time.Since(wn.GetLastHeartBeat()) < time.Second)
}

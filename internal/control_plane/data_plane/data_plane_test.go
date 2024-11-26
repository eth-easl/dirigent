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

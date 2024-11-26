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
	"cluster_manager/internal/control_plane/core"
	"cluster_manager/pkg/synchronization"
	"sync/atomic"
)

type RoundRobinPlacement struct {
	schedulingCounterRoundRobin int32
}

func NewRoundRobinPlacement() *RoundRobinPlacement {
	return &RoundRobinPlacement{
		schedulingCounterRoundRobin: 0,
	}
}

func (policy *RoundRobinPlacement) Place(storage synchronization.SyncStructure[string, core.WorkerNodeInterface], _ *ResourceMap) core.WorkerNodeInterface {
	schedulable := getSchedulableNodes(storage.GetValues())
	if len(schedulable) == 0 {
		return nil
	}

	var index int32
	swapped := false
	for !swapped {
		index = atomic.LoadInt32(&policy.schedulingCounterRoundRobin) % int32(len(schedulable))
		newIndex := (index + 1) % int32(len(schedulable))

		swapped = atomic.CompareAndSwapInt32(&policy.schedulingCounterRoundRobin, index, newIndex)
	}

	return schedulable[index]
}

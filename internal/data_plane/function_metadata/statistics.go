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

package function_metadata

import (
	"sync"
	"sync/atomic"
)

type FunctionStatistics struct {
	SuccessfulInvocations int64 `json:"SuccessfulInvocations"`
	Inflight              int64 `json:"Inflight"`
	QueueDepth            int64 `json:"QueueDepth"`
}

func (fs *FunctionStatistics) GetSuccessfulInvocations() int64 {
	return atomic.LoadInt64(&fs.SuccessfulInvocations)
}

func (fs *FunctionStatistics) IncrementSuccessfulInvocations() {
	atomic.AddInt64(&fs.SuccessfulInvocations, 1)
}

func (fs *FunctionStatistics) GetInflight() int64 {
	return atomic.LoadInt64(&fs.Inflight)
}

func (fs *FunctionStatistics) IncrementInflight() {
	atomic.AddInt64(&fs.Inflight, 1)
}

func (fs *FunctionStatistics) DecrementInflight() {
	atomic.AddInt64(&fs.Inflight, -1)
}

func (fs *FunctionStatistics) GetQueueDepth() int64 {
	return atomic.LoadInt64(&fs.QueueDepth)
}

func (fs *FunctionStatistics) IncrementQueueDepth() {
	atomic.AddInt64(&fs.QueueDepth, 1)
}

func (fs *FunctionStatistics) DecrementQueueDepth() {
	atomic.AddInt64(&fs.QueueDepth, -1)
}

func AggregateStatistics(functions []*FunctionMetadata) *FunctionStatistics {
	aggregate := &FunctionStatistics{}
	wg := &sync.WaitGroup{}

	wg.Add(len(functions))
	for _, f := range functions {
		go func(metadata *FunctionMetadata) {
			defer wg.Done()

			atomic.AddInt64(&aggregate.SuccessfulInvocations, metadata.GetStatistics().GetSuccessfulInvocations())
			atomic.AddInt64(&aggregate.Inflight, metadata.GetStatistics().GetInflight())
			atomic.AddInt64(&aggregate.QueueDepth, metadata.GetStatistics().GetQueueDepth())
		}(f)
	}
	wg.Wait()

	return aggregate
}

package service_metadata

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

func AggregateStatistics(functions []*ServiceMetadata) *FunctionStatistics {
	aggregate := &FunctionStatistics{}
	wg := &sync.WaitGroup{}

	wg.Add(len(functions))
	for _, f := range functions {
		go func(metadata *ServiceMetadata) {
			defer wg.Done()

			atomic.AddInt64(&aggregate.SuccessfulInvocations, metadata.GetStatistics().GetSuccessfulInvocations())
			atomic.AddInt64(&aggregate.Inflight, metadata.GetStatistics().GetInflight())
			atomic.AddInt64(&aggregate.QueueDepth, metadata.GetStatistics().GetQueueDepth())
		}(f)
	}
	wg.Wait()

	return aggregate
}

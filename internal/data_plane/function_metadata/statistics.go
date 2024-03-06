package function_metadata

import "sync/atomic"

type FunctionStatistics struct {
	SuccessfulInvocations int64 `json:"SuccessfulInvocations"`
	Inflight              int64 `json:"Inflight"`
	QueueDepth            int64 `json:"QueueDepth"`
}

func (fs *FunctionStatistics) IncrementSuccessInvocations() {
	atomic.AddInt64(&fs.SuccessfulInvocations, 1)
}

func (fs *FunctionStatistics) IncrementInflight() {
	atomic.AddInt64(&fs.Inflight, 1)
}

func (fs *FunctionStatistics) DecrementInflight() {
	atomic.AddInt64(&fs.Inflight, -1)
}

func (fs *FunctionStatistics) IncrementQueueDepth() {
	atomic.AddInt64(&fs.QueueDepth, 1)
}

func (fs *FunctionStatistics) DecrementQueueDepth() {
	atomic.AddInt64(&fs.QueueDepth, -1)
}

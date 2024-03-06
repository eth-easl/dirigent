package function_metadata

import "sync/atomic"

type FunctionStatistics struct {
	ProcessedTotal int64 `json:"ProcessedTotal"`
	Inflight       int64 `json:"Inflight"`
	QueueDepth     int64 `json:"QueueDepth"`
}

func (fs *FunctionStatistics) IncreaseProcessedTotal() {
	atomic.AddInt64(&fs.ProcessedTotal, 1)
}

func (fs *FunctionStatistics) IncreaseInflight() {
	atomic.AddInt64(&fs.Inflight, 1)
}

func (fs *FunctionStatistics) DecreaseInflight() {
	atomic.AddInt64(&fs.Inflight, -1)
}

func (fs *FunctionStatistics) IncreaseQueueDepth() {
	atomic.AddInt64(&fs.QueueDepth, 1)
}

func (fs *FunctionStatistics) DecreaseQueueDepth() {
	atomic.AddInt64(&fs.QueueDepth, -1)
}

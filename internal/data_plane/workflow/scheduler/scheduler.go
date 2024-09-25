package scheduler

import (
	"cluster_manager/internal/data_plane/workflow"
	"fmt"
)

type SchedulerType int

const (
	SequentialFifo SchedulerType = iota
	ConcurrentFifo
)

type ScheduleTaskFunc func(*workflow.Statement) error

type Scheduler interface {
	Schedule(ScheduleTaskFunc, []workflow.Data) error
	CollectOutput() ([]workflow.Data, error)
}

func NewScheduler(wf *workflow.Workflow, t SchedulerType) Scheduler {
	switch t {
	case SequentialFifo:
		return NewSequentialFifoScheduler(wf)
	case ConcurrentFifo:
		return NewConcurrentFifoScheduler(wf)
	default:
		fmt.Printf("Unsupported scheduler type.\n")
		return nil
	}
}

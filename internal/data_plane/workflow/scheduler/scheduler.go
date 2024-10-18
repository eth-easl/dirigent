package scheduler

import (
	"cluster_manager/internal/data_plane/workflow"
	"github.com/sirupsen/logrus"
)

type SchedulerType int

const (
	Invalid SchedulerType = iota
	SequentialFifo
	ConcurrentFifo
)

func SchedulerTypeFromString(s string) SchedulerType {
	switch s {
	case "sequentialFifo":
		return SequentialFifo
	case "concurrentFifo":
		return ConcurrentFifo
	default:
		return Invalid
	}
}

type ScheduleTaskFunc func(*workflow.TaskOrchestrator, *workflow.Task) error

type Scheduler interface {
	Schedule(ScheduleTaskFunc, []*workflow.Data) error
	CollectOutput() ([]*workflow.Data, error)
}

func NewScheduler(wf *workflow.Workflow, t SchedulerType) Scheduler {
	switch t {
	case SequentialFifo:
		return NewSequentialFifoScheduler(wf)
	case ConcurrentFifo:
		return NewConcurrentFifoScheduler(wf)
	default:
		logrus.Errorf("Unsupported scheduler type.")
		return nil
	}
}

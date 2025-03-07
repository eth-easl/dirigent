package scheduler

import (
	"cluster_manager/internal/data_plane/workflow"
	"cluster_manager/pkg/config"
	"context"
	"github.com/sirupsen/logrus"
	"time"
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

type SchedulingSummary struct {
	Startup    time.Duration
	Scheduling time.Duration
	Execution  time.Duration
}

type ScheduleTaskFunc func(*workflow.TaskOrchestrator, *workflow.SchedulerTask, context.Context) error

type Scheduler interface {
	Schedule(ScheduleTaskFunc, []*workflow.Data, *config.DataPlaneConfig, context.Context) (*SchedulingSummary, error)
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

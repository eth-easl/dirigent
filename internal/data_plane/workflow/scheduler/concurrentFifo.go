package scheduler

import (
	"cluster_manager/internal/data_plane/workflow"
	"cluster_manager/pkg/config"
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	"sync/atomic"
	"time"
)

type ConcurrentFifoScheduler struct {
	wf            *workflow.Workflow
	eg            *errgroup.Group
	egCtx         context.Context
	orchestrator  *workflow.TaskOrchestrator
	tasksFinished atomic.Uint64
	sCtx          context.Context

	schedulingDuration atomic.Int64 // duration in nanoseconds
	executionDuration  atomic.Int64 // duration in nanoseconds
}

func NewConcurrentFifoScheduler(wf *workflow.Workflow) *ConcurrentFifoScheduler {
	eg, ctx := errgroup.WithContext(context.Background())
	return &ConcurrentFifoScheduler{
		wf:                 wf,
		eg:                 eg,
		egCtx:              ctx,
		tasksFinished:      atomic.Uint64{},
		schedulingDuration: atomic.Int64{},
		executionDuration:  atomic.Int64{},
	}
}

func (s *ConcurrentFifoScheduler) workTask(sTask *workflow.SchedulerTask, scheduleTask ScheduleTaskFunc) func() error {
	return func() error {
		// schedule (sub)task
		executionStart := time.Now()
		logrus.Tracef(
			"ConcurrentFifoScheduler: Scheduling task %s-%d (p=%d)...",
			sTask.GetTask().Name, sTask.SubtaskIdx, sTask.GetDataParallelism(),
		)
		err := scheduleTask(s.orchestrator, sTask, s.sCtx)
		if err != nil {
			return err
		}
		executionDuration := time.Since(executionStart)

		// check if task is done
		schedulingStart := time.Now()
		taskDone, nextSchedulerTasks := s.orchestrator.SetDone(sTask)
		if taskDone {
			logrus.Tracef("ConcurrentFifoScheduler: Task '%s' finished.", sTask.GetTask().Name)
			s.tasksFinished.Add(1)
		} else {
			logrus.Tracef("SequentialFifoScheduler: Subtask %s-%d finished.", sTask.GetTask().Name, sTask.SubtaskIdx)
		}

		// check if scheduling context has been cancelled
		if s.egCtx.Err() != nil {
			return nil
		}

		// enqueue next runnable tasks
		for _, nextTask := range nextSchedulerTasks {
			s.eg.Go(s.workTask(nextTask, scheduleTask))
		}

		s.schedulingDuration.Add(time.Since(schedulingStart).Nanoseconds())
		s.executionDuration.Add(executionDuration.Nanoseconds())
		return nil
	}
}

func (s *ConcurrentFifoScheduler) Schedule(scheduleTask ScheduleTaskFunc, inData []*workflow.Data,
	dpConfig *config.DataPlaneConfig, ctx context.Context) (*SchedulingSummary, error) {
	startTime := time.Now()

	// set scheduling context
	s.sCtx = ctx

	// create workers for all initially runnable tasks
	orchestrator, initTasks, err := workflow.GetInitialRunnable(s.wf, inData, dpConfig)
	if err != nil {
		return nil, fmt.Errorf("scheduler failed to get initial runnable statements: %v", err)
	}
	s.orchestrator = orchestrator
	startDuration := time.Since(startTime)
	for _, task := range initTasks {
		s.eg.Go(s.workTask(task, scheduleTask))
	}
	tasksTotal := uint64(len(s.wf.Tasks))

	// wait for workers to finish
	err = s.eg.Wait()
	if err != nil {
		return nil, fmt.Errorf("scheduler failed to schedule tasks: %v", err)
	}

	if s.tasksFinished.Load() < tasksTotal {
		return nil, fmt.Errorf("scheduler cannot find anymore runnable tasks but not all tasks finished")
	}

	return &SchedulingSummary{
		Startup:    startDuration,
		Scheduling: time.Duration(s.schedulingDuration.Load()),
		Execution:  time.Duration(s.executionDuration.Load()),
	}, nil
}

func (s *ConcurrentFifoScheduler) CollectOutput() ([]*workflow.Data, error) {
	if s.orchestrator == nil {
		return nil, fmt.Errorf("did not yet run workflow")
	}
	return s.orchestrator.CollectOutData(), nil
}

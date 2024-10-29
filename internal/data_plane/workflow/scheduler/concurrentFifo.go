package scheduler

import (
	"cluster_manager/internal/data_plane/workflow"
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	"sync/atomic"
)

type ConcurrentFifoScheduler struct {
	wf            *workflow.Workflow
	eg            *errgroup.Group
	egCtx         context.Context
	orchestrator  *workflow.TaskOrchestrator
	tasksFinished atomic.Uint64
	sCtx          context.Context
}

func NewConcurrentFifoScheduler(wf *workflow.Workflow) *ConcurrentFifoScheduler {
	eg, ctx := errgroup.WithContext(context.Background())
	return &ConcurrentFifoScheduler{
		wf:            wf,
		eg:            eg,
		egCtx:         ctx,
		tasksFinished: atomic.Uint64{},
	}
}

func (s *ConcurrentFifoScheduler) workTask(sTask *workflow.SchedulerTask, scheduleTask ScheduleTaskFunc) func() error {
	return func() error {
		// schedule (sub)task
		logrus.Tracef("ConcurrentFifoScheduler: Scheduling task '%s'...", sTask.GetTask().Name)
		err := scheduleTask(s.orchestrator, sTask, s.sCtx)
		if err != nil {
			return err
		}

		// check if task is done
		taskDone, nextSchedulerTasks := s.orchestrator.SetDone(sTask)
		if taskDone {
			logrus.Tracef("ConcurrentFifoScheduler: Task '%s' finished.", sTask.GetTask().Name)
			s.tasksFinished.Add(1)
		}

		// check if scheduling context has been cancelled
		if s.egCtx.Err() != nil {
			return nil
		}

		// enqueue next runnable tasks
		for _, nextTask := range nextSchedulerTasks {
			s.eg.Go(s.workTask(nextTask, scheduleTask))
		}

		return nil
	}
}

func (s *ConcurrentFifoScheduler) Schedule(scheduleTask ScheduleTaskFunc, inData []*workflow.Data, ctx context.Context) error {
	// set scheduling context
	s.sCtx = ctx

	// create workers for all initially runnable tasks
	orchestrator, initTasks, err := workflow.GetInitialRunnable(s.wf, inData)
	if err != nil {
		return fmt.Errorf("scheduler failed to get initial runnable statements: %v", err)
	}
	s.orchestrator = orchestrator
	for _, task := range initTasks {
		s.eg.Go(s.workTask(task, scheduleTask))
	}
	tasksTotal := uint64(len(s.wf.Tasks))

	// wait for workers to finish
	err = s.eg.Wait()
	if err != nil {
		return fmt.Errorf("scheduler failed to schedule tasks: %v", err)
	}

	if s.tasksFinished.Load() < tasksTotal {
		return fmt.Errorf("scheduler cannot find anymore runnable tasks but not all tasks finished")
	}

	return nil
}

func (s *ConcurrentFifoScheduler) CollectOutput() ([]*workflow.Data, error) {
	if s.orchestrator == nil {
		return nil, fmt.Errorf("did not yet run workflow")
	}
	return s.orchestrator.CollectOutData(), nil
}

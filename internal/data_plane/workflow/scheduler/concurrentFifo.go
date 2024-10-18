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
	sCtx          context.Context
	orchestrator  *workflow.TaskOrchestrator
	tasksFinished atomic.Uint64
}

func NewConcurrentFifoScheduler(wf *workflow.Workflow) *ConcurrentFifoScheduler {
	eg, ctx := errgroup.WithContext(context.Background())
	return &ConcurrentFifoScheduler{
		wf:            wf,
		eg:            eg,
		sCtx:          ctx,
		tasksFinished: atomic.Uint64{},
	}
}

func (s *ConcurrentFifoScheduler) workStmt(task *workflow.Task, scheduleTask ScheduleTaskFunc) func() error {
	return func() error {
		// schedule task
		logrus.Tracef("ConcurrentFifoScheduler: Scheduling task '%s'...", task.Name)
		err := scheduleTask(s.orchestrator, task)
		if err != nil {
			return err
		}
		s.tasksFinished.Add(1)
		logrus.Tracef("ConcurrentFifoScheduler: Task '%s' finished.", task.Name)

		// check if scheduling context has been cancelled
		if s.sCtx.Err() != nil {
			return nil
		}

		// enqueue next runnable tasks
		for _, nextTask := range s.orchestrator.SetDone(task) {
			s.eg.Go(s.workStmt(nextTask, scheduleTask))
		}

		return nil
	}
}

func (s *ConcurrentFifoScheduler) Schedule(scheduleTask ScheduleTaskFunc, inData []*workflow.Data) error {
	// create workers for all initially runnable tasks
	orchestrator, initTasks, err := workflow.GetInitialRunnable(s.wf, inData)
	if err != nil {
		return fmt.Errorf("scheduler failed to get initial runnable statements: %v", err)
	}
	s.orchestrator = orchestrator
	for _, task := range initTasks {
		s.eg.Go(s.workStmt(task, scheduleTask))
	}
	tasksTotal := uint64(len(s.wf.Tasks))

	// wait for workers orchestrator finish
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

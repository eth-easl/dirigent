package scheduler

import (
	"cluster_manager/internal/data_plane/workflow"
	"fmt"
	"github.com/sirupsen/logrus"
	"sync"
	"sync/atomic"
)

type ConcurrentFifoScheduler struct {
	wf            *workflow.Workflow
	wg            *sync.WaitGroup
	errorChan     chan error
	tasksFinished atomic.Uint64
}

func NewConcurrentFifoScheduler(wf *workflow.Workflow) *ConcurrentFifoScheduler {
	return &ConcurrentFifoScheduler{
		wf:            wf,
		wg:            &sync.WaitGroup{},
		errorChan:     make(chan error),
		tasksFinished: atomic.Uint64{},
	}
}

func (s *ConcurrentFifoScheduler) workStmt(stmt *workflow.Statement, scheduleTask ScheduleTaskFunc) {
	defer s.wg.Done()

	// schedule task
	logrus.Debugf("Scheduling task '%s'\n", stmt.Name)
	err := scheduleTask(stmt.Name)
	if err != nil {
		s.errorChan <- err
		return
	}
	s.tasksFinished.Add(1)
	logrus.Debugf("Task '%s' finished\n", stmt.Name)

	// enqueue next runnable tasks
	for _, next := range stmt.SetDone() {
		s.wg.Add(1)
		go s.workStmt(next, scheduleTask)
	}
}

func (s *ConcurrentFifoScheduler) Schedule(scheduleTask ScheduleTaskFunc) error {
	err := s.wf.Process()
	if err != nil {
		return fmt.Errorf("scheduler failed to process workflow: %v", err)
	}

	// TODO: only schedules a single composition right now
	if len(s.wf.Compositions) != 1 {
		return fmt.Errorf("scheduler expected exactly 1 composition, but got %v", len(s.wf.Compositions))
	}
	comp := s.wf.Compositions[0]

	// create workers for all initially runnable tasks
	for _, task := range comp.GetInitialRunnable() {
		s.wg.Add(1)
		go s.workStmt(task, scheduleTask)
	}
	tasksTotal := uint64(len(comp.Statements))

	// wait for workers to finish + check if errors occured
	s.wg.Wait()
	select {
	case err = <-s.errorChan:
		return fmt.Errorf("scheduler failed to execute workflow: %v", err)
	default:
	}

	if s.tasksFinished.Load() < tasksTotal {
		return fmt.Errorf("scheduler cannot find anymore runnable tasks but not all tasks finished")
	}

	return nil
}

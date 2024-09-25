package scheduler

import (
	"cluster_manager/internal/data_plane/workflow"
	"fmt"
	"github.com/sirupsen/logrus"
)

type SequentialFifoScheduler struct {
	wf *workflow.Workflow
}

func NewSequentialFifoScheduler(wf *workflow.Workflow) *SequentialFifoScheduler {
	return &SequentialFifoScheduler{wf: wf}
}

func (s *SequentialFifoScheduler) Schedule(scheduleTask ScheduleTaskFunc, inData []workflow.Data) error {
	err := s.wf.Process()
	if err != nil {
		return fmt.Errorf("scheduler failed to process workflow: %v", err)
	}

	// TODO: only schedules a single composition right now
	if len(s.wf.Compositions) != 1 {
		return fmt.Errorf("scheduler expected exactly 1 composition, but got %v", len(s.wf.Compositions))
	}
	comp := s.wf.Compositions[0]

	queue, err := comp.GetInitialRunnable(inData)
	if err != nil {
		return fmt.Errorf("scheduler failed to get initial runnable tasks: %v", err)
	}
	tasksFinished := 0
	tasksTotal := comp.GetNumStatements()

	for len(queue) > 0 {
		currStmt := queue[0]

		// schedule task
		logrus.Tracef("Scheduling task '%s' (queue depth: %d)\n", currStmt.Name, len(queue))
		err = scheduleTask(currStmt)
		if err != nil {
			return fmt.Errorf("scheduler failed to execute task: %v", err)
		}

		tasksFinished++
		queue = queue[1:]
		logrus.Tracef("Task finished (%d/%d).\n", tasksFinished, tasksTotal)

		if tasksTotal == tasksFinished {
			if len(queue) > 0 {
				return fmt.Errorf("expected number of tasks were executed but queue is not empty")
			}
			break
		}

		// add statements that are now runnable
		for _, nextStmt := range currStmt.SetDone() {
			queue = append(queue, nextStmt)
		}
	}

	if tasksFinished < tasksTotal {
		return fmt.Errorf("scheduler cannot find anymore runnable tasks but not all tasks finished")
	}

	return nil
}

func (s *SequentialFifoScheduler) CollectOutput() ([]workflow.Data, error) {
	// TODO: check that workflow has run
	// TODO: only schedules a single composition right now
	if len(s.wf.Compositions) != 1 {
		return nil, fmt.Errorf("scheduler expected exactly 1 composition, but got %v", len(s.wf.Compositions))
	}
	comp := s.wf.Compositions[0]

	return comp.CollectOutData(), nil
}

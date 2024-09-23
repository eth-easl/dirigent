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

func (s *SequentialFifoScheduler) Schedule(scheduleTask ScheduleTaskFunc) error {
	err := s.wf.Process()
	if err != nil {
		return fmt.Errorf("scheduler failed to process workflow: %v", err)
	}

	// TODO: only schedules a single composition right now
	if len(s.wf.Compositions) != 1 {
		return fmt.Errorf("scheduler expected exactly 1 composition, but got %v", len(s.wf.Compositions))
	}
	comp := s.wf.Compositions[0]

	queue := comp.GetInitialRunnable()
	tasksFinished := 0
	tasksTotal := len(comp.Statements)

	for len(queue) > 0 {
		next := queue[0]

		logrus.Debugf("Scheduling task '%s' (queue depth: %d)\n", next.Name, len(queue))
		err = scheduleTask(next.Name)
		if err != nil {
			return fmt.Errorf("scheduler failed to execute task: %v", err)
		}

		tasksFinished++
		queue = queue[1:]
		logrus.Debugf("Task finished (%d/%d).\n", tasksFinished, tasksTotal)

		if tasksTotal == tasksFinished {
			if len(queue) > 0 {
				return fmt.Errorf("expected number of tasks were executed but queue is not empty")
			}
			break
		}

		// add statements that are now runnable
		for _, stmt := range next.SetDone() {
			queue = append(queue, stmt)
		}
	}

	if tasksFinished < tasksTotal {
		return fmt.Errorf("scheduler cannot find anymore runnable tasks but not all tasks finished")
	}

	return nil
}

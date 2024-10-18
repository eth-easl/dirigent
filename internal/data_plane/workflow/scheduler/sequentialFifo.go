package scheduler

import (
	"cluster_manager/internal/data_plane/workflow"
	"fmt"
	"github.com/sirupsen/logrus"
)

type SequentialFifoScheduler struct {
	wf           *workflow.Workflow
	orchestrator *workflow.TaskOrchestrator
}

func NewSequentialFifoScheduler(wf *workflow.Workflow) *SequentialFifoScheduler {
	return &SequentialFifoScheduler{wf: wf}
}

func (s *SequentialFifoScheduler) Schedule(scheduleTask ScheduleTaskFunc, inData []*workflow.Data) error {
	orchestrator, queue, err := workflow.GetInitialRunnable(s.wf, inData)
	if err != nil {
		return fmt.Errorf("scheduler failed to get initial runnable tasks: %v", err)
	}
	s.orchestrator = orchestrator
	tasksFinished := 0
	tasksTotal := len(s.wf.Tasks)

	for len(queue) > 0 {
		currTask := queue[0]

		// schedule task
		logrus.Tracef("SequentialFifoScheduler: Scheduling task '%s' (queue depth: %d)...", currTask.Name, len(queue))
		err = scheduleTask(orchestrator, currTask)
		if err != nil {
			return fmt.Errorf("scheduler failed to execute task: %v", err)
		}

		tasksFinished++
		queue = queue[1:]
		logrus.Tracef("SequentialFifoScheduler: Task finished (%d/%d).", tasksFinished, tasksTotal)

		if tasksTotal == tasksFinished {
			if len(queue) > 0 {
				return fmt.Errorf("expected number of tasks were executed but queue is not empty")
			}
			break
		}

		// add statements that are now runnable
		queue = append(queue, orchestrator.SetDone(currTask)...)
	}

	if tasksFinished < tasksTotal {
		return fmt.Errorf("scheduler cannot find anymore runnable tasks but not all tasks finished")
	}

	return nil
}

func (s *SequentialFifoScheduler) CollectOutput() ([]*workflow.Data, error) {
	if s.orchestrator == nil {
		return nil, fmt.Errorf("did not yet run workflow")
	}
	return s.orchestrator.CollectOutData(), nil
}

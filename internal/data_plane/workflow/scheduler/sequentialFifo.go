package scheduler

import (
	"cluster_manager/internal/data_plane/workflow"
	"cluster_manager/pkg/config"
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"time"
)

type SequentialFifoScheduler struct {
	wf           *workflow.Workflow
	orchestrator *workflow.TaskOrchestrator
}

func NewSequentialFifoScheduler(wf *workflow.Workflow) *SequentialFifoScheduler {
	return &SequentialFifoScheduler{wf: wf}
}

func (s *SequentialFifoScheduler) Schedule(scheduleTask ScheduleTaskFunc, inData []*workflow.Data, dpConfig *config.DataPlaneConfig, ctx context.Context) (*SchedulingSummary, error) {
	startTime := time.Now()
	orchestrator, queue, err := workflow.GetInitialRunnable(s.wf, inData, dpConfig)
	if err != nil {
		return nil, fmt.Errorf("scheduler failed to get initial runnable tasks: %v", err)
	}
	s.orchestrator = orchestrator
	tasksFinished := 0
	tasksTotal := len(s.wf.Tasks)
	startDuration := time.Since(startTime)

	schedulingStart := time.Now()
	schedulingDuration := time.Duration(0)
	executionDuration := time.Duration(0)
	for len(queue) > 0 {
		currTask := queue[0]
		queue = queue[1:]
		schedulingDuration += time.Since(schedulingStart)

		// schedule task
		executionStart := time.Now()
		logrus.Tracef("SequentialFifoScheduler: Scheduling task %s-%d (p=%d) (queue depth: %d)...", currTask.GetTask().Name, currTask.SubtaskIdx, currTask.GetDataParallelism(), len(queue))
		err = scheduleTask(orchestrator, currTask, ctx)
		if err != nil {
			return nil, fmt.Errorf("scheduler failed to execute task: %v", err)
		}
		executionDuration += time.Since(executionStart)

		// check if task is done
		schedulingStart = time.Now()
		taskDone, nextSchedulerTasks := orchestrator.SetDone(currTask)
		if taskDone {
			logrus.Tracef("SequentialFifoScheduler: Task finished (%d/%d).", tasksFinished, tasksTotal)
			tasksFinished++

			if tasksTotal == tasksFinished {
				if len(queue) > 0 {
					return nil, fmt.Errorf("expected number of tasks were executed but queue is not empty")
				}
				break
			}
		} else {
			logrus.Tracef("SequentialFifoScheduler: Subtask %s-%d finished.", currTask.GetTask().Name, currTask.SubtaskIdx)
		}

		// add statements that are now runnable
		queue = append(queue, nextSchedulerTasks...)
	}

	if tasksFinished < tasksTotal {
		return nil, fmt.Errorf("scheduler cannot find anymore runnable tasks but not all tasks finished")
	}
	schedulingDuration += time.Since(schedulingStart)

	return &SchedulingSummary{
		Startup:    startDuration,
		Scheduling: schedulingDuration,
		Execution:  executionDuration,
	}, nil
}

func (s *SequentialFifoScheduler) CollectOutput() ([]*workflow.Data, error) {
	if s.orchestrator == nil {
		return nil, fmt.Errorf("did not yet run workflow")
	}
	return s.orchestrator.CollectOutData(), nil
}

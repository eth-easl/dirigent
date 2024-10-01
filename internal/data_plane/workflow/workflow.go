package workflow

import (
	"cluster_manager/proto"
	"github.com/sirupsen/logrus"
)

type Task struct {
	Name   string
	NumIn  uint32
	NumOut uint32

	Functions        []string
	FunctionInNum    []int32
	FunctionOutNum   []int32
	FunctionDataFlow []int32

	ConsumerTasks      []*Task
	ConsumerDataSrcIdx []int32
	ConsumerDataDstIdx []int32
}

type Workflow struct {
	Name  string
	Tasks []*Task

	InitialTasks      []*Task
	InitialDataSrcIdx []int32
	InitialDataDstIdx []int32

	OutTasks      []*Task
	OutDataSrcIdx []int32
}

func CreateFromWorkflowInfo(wfInfo *proto.WorkflowInfo) *Workflow {
	nameToTask := make(map[string]*Task)

	for _, tInfo := range wfInfo.Tasks {
		nameToTask[tInfo.Name] = &Task{}
	}
	for _, tInfo := range wfInfo.Tasks {
		logrus.Printf("creating task %s", tInfo.Name)
		task := nameToTask[tInfo.Name]

		task.Name = tInfo.Name
		task.NumIn = tInfo.NumIn
		task.NumOut = tInfo.NumOut
		task.Functions = tInfo.Functions
		task.FunctionInNum = tInfo.FunctionInNum
		task.FunctionOutNum = tInfo.FunctionOutNum
		task.FunctionDataFlow = tInfo.FunctionDataFlow
		task.ConsumerDataSrcIdx = tInfo.ConsumerDataSrcIdx
		task.ConsumerDataDstIdx = tInfo.ConsumerDataDstIdx

		task.ConsumerTasks = make([]*Task, len(tInfo.ConsumerTasks))
		for i, taskName := range tInfo.ConsumerTasks {
			task.ConsumerTasks[i] = nameToTask[taskName]
		}
	}

	logrus.Printf("creating workflow %s", wfInfo.Name)
	wf := &Workflow{
		Name:              wfInfo.Name,
		Tasks:             make([]*Task, len(wfInfo.Tasks)),
		InitialTasks:      make([]*Task, len(wfInfo.InitialTasks)),
		InitialDataSrcIdx: wfInfo.InitialDataSrcIdx,
		InitialDataDstIdx: wfInfo.InitialDataDstIdx,
		OutTasks:          make([]*Task, len(wfInfo.OutTasks)),
		OutDataSrcIdx:     wfInfo.OutDataSrcIdx,
	}
	idx := 0
	for _, task := range nameToTask {
		wf.Tasks[idx] = task
		idx++
	}
	for i, initTask := range wfInfo.InitialTasks {
		wf.InitialTasks[i] = nameToTask[initTask]
	}
	for i, outTask := range wfInfo.OutTasks {
		wf.OutTasks[i] = nameToTask[outTask]
	}

	return wf
}

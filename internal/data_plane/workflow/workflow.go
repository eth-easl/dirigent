package workflow

import (
	"cluster_manager/proto"
	"github.com/sirupsen/logrus"
)

type Task struct {
	Name   string
	NumIn  uint32
	NumOut uint32

	Functions        []string // functions belonging to this task
	FunctionInNum    []int32  // # args per function (if > 1 function)
	FunctionOutNum   []int32  // # returns per function (if > 1 function)
	FunctionDataFlow []int32  // (src func idx, arg idx) for each function input + task outputs describing internal dataflow (if > 1 function)

	ConsumerTasks      []*Task // tasks consuming output of this task
	ConsumerDataSrcIdx []int32 // argument idx in consumer
	ConsumerDataDstIdx []int32 // return idx of this task
}

type Workflow struct {
	Name  string
	Tasks []*Task

	InitialTasks      []*Task // tasks consuming workflow input
	InitialDataSrcIdx []int32 // argument idx in consumer
	InitialDataDstIdx []int32 // workflow input idx

	OutTasks      []*Task // source tasks for workflow output
	OutDataSrcIdx []int32 // return idx of source task
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

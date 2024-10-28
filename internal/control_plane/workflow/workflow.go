package workflow

import "cluster_manager/proto"

type Sharding uint32

const (
	ShardingAll Sharding = iota
	ShardingKeyed
	ShardingEach
)

type Task struct {
	Name          string
	NumIn         uint32
	NumOut        uint32
	InputSharding []Sharding

	Functions          []string   // functions belonging to this task
	FunctionInNum      []int32    // # args per function (if > 1 function)
	FunctionOutNum     []int32    // # returns per function (if > 1 function)
	FunctionDataFlow   []int32    // (src func idx, arg idx) for each function input + task outputs describing internal dataflow (if > 1 function)
	FunctionInSharding []Sharding // sharding for each function input

	ConsumerTasks      []*Task // tasks consuming output of this task
	ConsumerDataSrcIdx []int32 // argument idx in consumer
	ConsumerDataDstIdx []int32 // return idx of this task
}

type Workflow struct {
	Name       string
	TotalTasks uint32
	NumIn      uint32
	NumOut     uint32

	InitialTasks      []*Task // tasks consuming workflow input
	InitialDataSrcIdx []int32 // argument idx in consumer
	InitialDataDstIdx []int32 // workflow input idx

	OutTasks      []*Task // source tasks for workflow output
	OutDataSrcIdx []int32 // return idx of source task
}

func TaskToStr(tasks []*Task) []string {
	str := make([]string, len(tasks))
	for i, t := range tasks {
		str[i] = t.Name
	}
	return str
}
func ShardingToUint32(sharding []Sharding) []uint32 {
	out := make([]uint32, len(sharding))
	for i, s := range sharding {
		out[i] = uint32(s)
	}
	return out
}

type StorageTacker struct {
	parent string
	tasks  []string
	wfInfo *proto.WorkflowInfo
}

func NewWorkflow(wfInfo *proto.WorkflowInfo) *StorageTacker {
	return &StorageTacker{
		tasks:  make([]string, len(wfInfo.Tasks)),
		wfInfo: wfInfo,
	}
}

func NewTask(parent string) *StorageTacker {
	return &StorageTacker{
		parent: parent,
	}
}

func (s *StorageTacker) IsWorkflow() bool {
	return s.parent == ""
}

func (s *StorageTacker) IsTask() bool {
	return s.parent == ""
}

func (s *StorageTacker) SetTask(idx int, t string) {
	s.tasks[idx] = t
}

func (s *StorageTacker) GetTasks() []string {
	return s.tasks
}

func (s *StorageTacker) GetWorkflowInfo() *proto.WorkflowInfo {
	return s.wfInfo
}

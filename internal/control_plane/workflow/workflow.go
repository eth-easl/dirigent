package workflow

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
	Name       string
	TotalTasks uint32

	InitialTasks      []*Task
	InitialDataSrcIdx []int32
	InitialDataDstIdx []int32

	OutTasks      []*Task
	OutDataSrcIdx []int32
}

func TaskToStr(tasks []*Task) []string {
	str := make([]string, len(tasks))
	for i, t := range tasks {
		str[i] = t.Name
	}
	return str
}

type StorageTacker struct {
	parent string
	tasks  []string
}

func NewWorkflow(numTasks int) *StorageTacker {
	return &StorageTacker{
		tasks: make([]string, numTasks),
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

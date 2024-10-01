package workflow

type Task struct {
	Name   string
	NumIn  int
	NumOut int

	Functions        []string
	FunctionInNum    []int
	FunctionOutNum   []int
	FunctionDataFlow []int

	ConsumerTasks      []*Task
	ConsumerDataSrcIdx []int
	ConsumerDataDstIdx []int
}

type Workflow struct {
	Name       string
	TotalTasks int

	InitialTasks      []*Task
	InitialDataSrcIdx []int
	InitialDataDstIdx []int

	OutTasks      []*Task
	OutDataSrcIdx []int
}

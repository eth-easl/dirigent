package dandelion_workflow

import (
	"cluster_manager/internal/control_plane/workflow"
	"fmt"
	"slices"
)

type PartitionMethod int

const (
	Invalid PartitionMethod = iota
	// FullPartition creates a task for every function
	FullPartition
	// NoPartition combines all functions into a single task
	NoPartition
	// ConsumerBased combines functions that consume exactly all output from the previous one
	ConsumerBased
)

func PartitionMethodFromString(s string) PartitionMethod {
	switch s {
	case "fullPartition":
		return FullPartition
	case "noPartition":
		return NoPartition
	case "consumerBased":
		return ConsumerBased
	default:
		return Invalid
	}
}

type partitionFunction func(*Composition, *workflow.Workflow) []*workflow.Task

func allTrue(in []bool) bool {
	for _, b := range in {
		if !b {
			return false
		}
	}
	return true
}

// createTaskFromStatements assumes wf has OutTasks and OutTasksSrcIdx initialized with correct size
func createTaskFromStatements(stmts []*Statement, wf *workflow.Workflow) *workflow.Task {
	numStmts := len(stmts)
	task := &workflow.Task{
		Functions:      make([]string, numStmts),
		FunctionOutNum: make([]int32, numStmts),
		FunctionInNum:  make([]int32, numStmts),
	}

	fToIdx := make(map[*Statement]int)
	var taskIn []string
	var taskOut []int
	var taskOutSrcIdx []int
	totalFuncInputs := 0
	for stmtIdx, stmt := range stmts {
		task.Functions[stmtIdx] = stmt.Name
		task.FunctionOutNum[stmtIdx] = int32(len(stmt.Rets))
		fToIdx[stmt] = stmtIdx
		totalFuncInputs += len(stmt.Args)
	}
	task.FunctionDataFlow = make([]int32, 0, totalFuncInputs) // + # taskOut (but this is unknown at this point)
	task.FunctionInSharding = make([]workflow.Sharding, 0, totalFuncInputs)
	task.InputSharding = make([]workflow.Sharding, 0, len(stmts[0].Args)) // may have more inputs (but this is unknown at this point)
	for stmtIdx, stmt := range stmts {
		stmtArgs := stmt.Args
		task.FunctionInNum[stmtIdx] = int32(len(stmtArgs))

		for argIdx := range stmtArgs {
			var argStmtIdx int
			var argStmtOutIdx int
			ok := false
			if stmtArgs[argIdx].SrcStmt != nil {
				argStmtIdx, ok = fToIdx[stmtArgs[argIdx].SrcStmt]
			}

			if !ok { // input from outside this task
				argStmtIdx = -1
				argStmtOutIdx = slices.Index(taskIn, stmtArgs[argIdx].src)
				if argStmtOutIdx == -1 { // not yet a task input
					argStmtOutIdx = len(taskIn)
					taskIn = append(taskIn, stmtArgs[argIdx].src)
					task.InputSharding = append(task.InputSharding, stmtArgs[argIdx].sharding)
				}
				if stmtArgs[argIdx].SrcTask != nil { // source is statement
					stmtArgs[argIdx].SrcTask.ConsumerTasks = append(stmtArgs[argIdx].SrcTask.ConsumerTasks, task)
					stmtArgs[argIdx].SrcTask.ConsumerDataSrcIdx = append(stmtArgs[argIdx].SrcTask.ConsumerDataSrcIdx, int32(stmtArgs[argIdx].SrcTaskOutIdx))
					stmtArgs[argIdx].SrcTask.ConsumerDataDstIdx = append(stmtArgs[argIdx].SrcTask.ConsumerDataDstIdx, int32(argStmtOutIdx))
				} else { // source is composition input
					wf.InitialTasks = append(wf.InitialTasks, task)
					wf.InitialDataSrcIdx = append(wf.InitialDataSrcIdx, int32(stmtArgs[argIdx].SrcStmtOutIdx))
					wf.InitialDataDstIdx = append(wf.InitialDataDstIdx, int32(argStmtOutIdx))
				}
			} else { // input from inside this task
				argStmtOutIdx = stmtArgs[argIdx].SrcStmtOutIdx
			}

			task.FunctionDataFlow = append(task.FunctionDataFlow, int32(argStmtIdx), int32(argStmtOutIdx))
			task.FunctionInSharding = append(task.FunctionInSharding, stmtArgs[argIdx].sharding)
		}

		for retIdx, retDesc := range stmt.Rets {
			taskOutIdx := -1
			for destIdx, destStmt := range retDesc.DestStmt {
				if !slices.Contains(stmts, destStmt) {
					if taskOutIdx == -1 {
						taskOutIdx = len(taskOut)
						taskOut = append(taskOut, stmtIdx)
						taskOutSrcIdx = append(taskOutSrcIdx, retIdx)
					}
					destStmt.Args[retDesc.DestStmtInIdx[destIdx]].SrcTask = task
					destStmt.Args[retDesc.DestStmtInIdx[destIdx]].SrcTaskOutIdx = taskOutIdx
				}
			}
			if retDesc.isCompOutput { // return is composition output
				if taskOutIdx == -1 {
					taskOutIdx = len(taskOut)
					taskOut = append(taskOut, stmtIdx)
					taskOutSrcIdx = append(taskOutSrcIdx, retIdx)
				}
				wf.OutTasks[retDesc.compOutputIdx] = task
				wf.OutDataSrcIdx[retDesc.compOutputIdx] = int32(taskOutIdx)
			}
		}
	}

	for taskOutIdx, stmtIdx := range taskOut {
		task.FunctionDataFlow = append(task.FunctionDataFlow, int32(stmtIdx), int32(taskOutSrcIdx[taskOutIdx]))
	}

	task.NumIn = uint32(len(taskIn))
	task.NumOut = uint32(len(taskOut))

	return task
}

func consumerBased(c *Composition, wf *workflow.Workflow) []*workflow.Task {
	var tasks []*workflow.Task

	wf.OutTasks = make([]*workflow.Task, len(c.outStmts))
	wf.OutDataSrcIdx = make([]int32, len(c.outStmts))

	var stack []*Statement
	for stmtIdx, stmt := range c.Consumers {
		stmt.parentProcessed[c.ConsumerArgIdx[stmtIdx]] = true
		if allTrue(stmt.parentProcessed) {
			stack = append(stack, stmt)
		}
	}
	stackSize := len(stack)

	var currTaskStmts []*Statement
	var currTaskDataParallel bool
	for stackSize > 0 {
		stmt := stack[stackSize-1]
		stackSize--
		stack = stack[:stackSize]

		if len(currTaskStmts) == 0 {
			currTaskStmts = append(currTaskStmts, stmt)
			currTaskDataParallel = stmt.isDataParallel()
		} else {
			if !stmt.hasOneParentAndSameParallelization(currTaskDataParallel) {
				task := createTaskFromStatements(currTaskStmts, wf)
				task.Name = fmt.Sprintf("%s_%d", wf.Name, len(tasks))
				tasks = append(tasks, task)
				currTaskStmts = nil
			}
			currTaskStmts = append(currTaskStmts, stmt)
			currTaskDataParallel = stmt.isDataParallel()
		}

		if !stmt.hasOneConsumer() {
			task := createTaskFromStatements(currTaskStmts, wf)
			task.Name = fmt.Sprintf("%s_%d", wf.Name, len(tasks))
			tasks = append(tasks, task)
			currTaskStmts = nil
		}

		prevStackSize := stackSize
		for _, ret := range stmt.Rets {
			for retDestIdx, retDest := range ret.DestStmt {
				retDest.parentProcessed[ret.DestStmtInIdx[retDestIdx]] = true
				if allTrue(retDest.parentProcessed) {
					stack = append(stack, retDest)
					stackSize++
				}
			}
		}
		if prevStackSize == stackSize && len(currTaskStmts) > 0 {
			task := createTaskFromStatements(currTaskStmts, wf)
			task.Name = fmt.Sprintf("%s_%d", wf.Name, len(tasks))
			tasks = append(tasks, task)
			currTaskStmts = nil
		}
	}

	if len(currTaskStmts) > 0 {
		task := createTaskFromStatements(currTaskStmts, wf)
		task.Name = fmt.Sprintf("%s_%d", wf.Name, len(tasks))
		tasks = append(tasks, task)
	}

	wf.TotalTasks = uint32(len(tasks))

	return tasks
}

func fullPartition(c *Composition, wf *workflow.Workflow) []*workflow.Task {
	var tasks []*workflow.Task

	// convert each statement to a task
	stmtToTask := make(map[*Statement]*workflow.Task)
	for stmtIdx, stmt := range c.Statements {
		task := &workflow.Task{
			Name:               fmt.Sprintf("%s_%s%d", wf.Name, stmt.Name, stmtIdx),
			Functions:          []string{stmt.Name},
			NumIn:              uint32(len(stmt.Args)),
			NumOut:             uint32(len(stmt.Rets)),
			InputSharding:      shardingFromInDescList(stmt.Args),
			FunctionInNum:      []int32{int32(len(stmt.Args))},
			FunctionOutNum:     []int32{int32(len(stmt.Rets))},
			FunctionInSharding: shardingFromInDescList(stmt.Args),
			FunctionDataFlow:   make([]int32, 2*(len(stmt.Args)+len(stmt.Rets))),
		}
		for i := 0; i < len(stmt.Args); i++ {
			task.FunctionDataFlow[i*2] = -1
			task.FunctionDataFlow[i*2+1] = int32(i)
		}
		for i := 0; i < len(stmt.Rets); i++ {
			task.FunctionDataFlow[(i+len(stmt.Args))*2] = 0
			task.FunctionDataFlow[(i+len(stmt.Args))*2+1] = int32(i)
		}
		stmtToTask[stmt] = task
		tasks = append(tasks, task)
	}

	// set consumers
	for _, stmt := range c.Statements {
		task := stmtToTask[stmt]
		for retIdx, ret := range stmt.Rets {
			for destIdx, dest := range ret.DestStmt {
				destTask := stmtToTask[dest]
				task.ConsumerTasks = append(task.ConsumerTasks, destTask)
				task.ConsumerDataSrcIdx = append(task.ConsumerDataSrcIdx, int32(retIdx))
				task.ConsumerDataDstIdx = append(task.ConsumerDataDstIdx, int32(ret.DestStmtInIdx[destIdx]))
			}
		}
	}

	// update workflow -> set initial tasks and output tasks
	for stmtIdx, stmt := range c.Consumers {
		wf.InitialTasks = append(wf.InitialTasks, stmtToTask[stmt])
		wf.InitialDataDstIdx = append(wf.InitialDataDstIdx, int32(c.ConsumerArgIdx[stmtIdx]))
		wf.InitialDataSrcIdx = append(wf.InitialDataSrcIdx, int32(c.ConsumerOutIdx[stmtIdx]))
	}
	for stmtIdx, stmt := range c.outStmts {
		wf.OutTasks = append(wf.OutTasks, stmtToTask[stmt])
		wf.OutDataSrcIdx = append(wf.OutDataSrcIdx, int32(c.outStmtRetIdx[stmtIdx]))
	}

	wf.TotalTasks = uint32(len(tasks))

	return tasks
}

func noPartition(c *Composition, wf *workflow.Workflow) []*workflow.Task {
	numStmts := len(c.Statements)
	task := &workflow.Task{
		Name:           fmt.Sprintf("%s_task", wf.Name),
		Functions:      make([]string, numStmts),
		FunctionOutNum: make([]int32, numStmts),
		FunctionInNum:  make([]int32, numStmts),
		NumIn:          uint32(len(c.params)),
		NumOut:         uint32(len(c.returns)),
		InputSharding:  make([]workflow.Sharding, len(c.params)),
	}

	// add all functions from composition + set internal data flow information
	fToIdx := make(map[*Statement]int)
	totalFuncInputs := 0
	for stmtIdx, stmt := range c.Statements {
		task.Functions[stmtIdx] = stmt.Name
		task.FunctionOutNum[stmtIdx] = int32(len(stmt.Rets))
		fToIdx[stmt] = stmtIdx
		totalFuncInputs += len(stmt.Args)
	}
	allowDataParallelisation := true
	inputUsed := make([]bool, len(c.params))
	task.FunctionDataFlow = make([]int32, 0, totalFuncInputs+len(c.outStmts))
	task.FunctionInSharding = make([]workflow.Sharding, 0, totalFuncInputs)
	for stmtIdx, stmt := range c.Statements {
		stmtArgs := stmt.Args
		task.FunctionInNum[stmtIdx] = int32(len(stmtArgs))

		for _, arg := range stmtArgs {
			var argStmtIdx int
			if arg.SrcStmt == nil {
				argStmtIdx = -1
				if inputUsed[arg.SrcStmtOutIdx] { // if input is used multiple times it must always use the same sharding
					if task.InputSharding[arg.SrcStmtOutIdx] != arg.sharding {
						allowDataParallelisation = false
					}
				} else {
					task.InputSharding[arg.SrcStmtOutIdx] = arg.sharding
					inputUsed[arg.SrcStmtOutIdx] = true
				}
			} else {
				argStmtIdx = fToIdx[arg.SrcStmt]
				if arg.sharding != workflow.ShardingEach {
					allowDataParallelisation = false
				}
			}
			argStmtOutIdx := arg.SrcStmtOutIdx
			task.FunctionDataFlow = append(task.FunctionDataFlow, int32(argStmtIdx), int32(argStmtOutIdx))
			task.FunctionInSharding = append(task.FunctionInSharding, arg.sharding)
		}
	}
	for stmtIdx, stmt := range c.outStmts {
		argStmtIdx := fToIdx[stmt]
		argStmtOutIdx := c.outStmtRetIdx[stmtIdx]
		task.FunctionDataFlow = append(task.FunctionDataFlow, int32(argStmtIdx), int32(argStmtOutIdx))
		wf.OutTasks = append(wf.OutTasks, task)
		wf.OutDataSrcIdx = append(wf.OutDataSrcIdx, int32(stmtIdx))
	}

	// update workflow -> set initial task
	for i := 0; i < len(c.Consumers); i++ {
		if !slices.Contains(wf.InitialDataSrcIdx, int32(c.ConsumerOutIdx[i])) {
			wf.InitialTasks = append(wf.InitialTasks, task)
			wf.InitialDataSrcIdx = append(wf.InitialDataSrcIdx, int32(c.ConsumerOutIdx[i]))
			wf.InitialDataDstIdx = append(wf.InitialDataDstIdx, int32(c.ConsumerOutIdx[i])) // dstIdx = outIdx
		}
	}

	// check input sharding
	if !allowDataParallelisation {
		for i := 0; i < int(task.NumIn); i++ {
			task.InputSharding[i] = workflow.ShardingAll
		}
	}

	wf.TotalTasks = 1

	return []*workflow.Task{task}
}

package dandelion_workflow

import (
	"cluster_manager/internal/control_plane/workflow"
	"fmt"
	"slices"
)

type PartitionMethod int

const (
	// FullPartition creates a task for every function
	FullPartition PartitionMethod = iota
	// NoPartition combines all functions into a single task
	NoPartition
	// ConsumerBased combines functions that consume exactly all output from the previous one
	ConsumerBased
)

type partitionFunction func(*Composition, *workflow.Workflow) []*workflow.Task

func allTrue(in []bool) bool {
	for _, b := range in {
		if !b {
			return false
		}
	}
	return true
}
func createTaskFromStatements(stmts []*Statement, wf *workflow.Workflow) *workflow.Task {
	numStmts := len(stmts)
	task := &workflow.Task{
		Functions:      make([]string, numStmts),
		FunctionOutNum: make([]int32, numStmts),
		FunctionInNum:  make([]int32, numStmts),
	}

	fToIdx := make(map[string]int)
	var taskIn []*InputDescriptor
	var taskOut []int
	for stmtIdx, stmt := range stmts {
		task.Functions[stmtIdx] = stmt.Name
		task.FunctionOutNum[stmtIdx] = int32(len(stmt.Rets))
		fToIdx[stmt.Name] = stmtIdx
	}
	for stmtIdx, stmt := range stmts {
		stmtArgs := stmt.Args
		task.FunctionInNum[stmtIdx] = int32(len(stmtArgs))

		for argIdx := range stmtArgs {
			var argStmtIdx int
			var argStmtOutIdx int
			ok := false
			if stmtArgs[argIdx].SrcStmt != nil {
				argStmtIdx, ok = fToIdx[stmtArgs[argIdx].SrcStmt.Name]
			}

			if !ok { // input from outside this task
				argStmtIdx = -1
				argStmtOutIdx = slices.Index(taskIn, &stmtArgs[argIdx])
				if argStmtOutIdx == -1 { // not yet a task input
					argStmtOutIdx = len(taskIn)
					taskIn = append(taskIn, &stmtArgs[argIdx])
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
		}

		for retIdx := range stmt.Rets {
			taskOutIdx := -1
			for destIdx, destStmt := range stmt.Rets[retIdx].DestStmt {
				if !slices.Contains(stmts, destStmt) {
					if taskOutIdx == -1 {
						taskOutIdx = len(taskOut)
						taskOut = append(taskOut, stmtIdx)
					}
					destStmt.Args[stmt.Rets[retIdx].DestStmtInIdx[destIdx]].SrcTask = task
					destStmt.Args[stmt.Rets[retIdx].DestStmtInIdx[destIdx]].SrcTaskOutIdx = taskOutIdx
				}
			}
			if stmt.Rets[retIdx].isCompOutput { // return is composition output
				if taskOutIdx == -1 {
					taskOutIdx = len(taskOut)
					taskOut = append(taskOut, stmtIdx)
				}
				wf.OutTasks = append(wf.OutTasks, task)
				wf.OutDataSrcIdx = append(wf.OutDataSrcIdx, int32(taskOutIdx))
			}
		}
	}

	task.NumIn = uint32(len(taskIn))
	task.NumOut = uint32(len(taskOut))

	return task
}

func consumerBased(c *Composition, wf *workflow.Workflow) []*workflow.Task {
	var tasks []*workflow.Task

	var currTaskStmts []*Statement
	var stack []*Statement
	for stmtIdx, stmt := range c.Consumers {
		stmt.parentProcessed[c.ConsumerArgIdx[stmtIdx]] = true
		if allTrue(stmt.parentProcessed) {
			stack = append(stack, stmt)
		}
	}
	stackSize := len(stack)

	for stackSize > 0 {
		stmt := stack[stackSize-1]
		stackSize--
		stack = stack[:stackSize]

		if len(currTaskStmts) == 0 {
			currTaskStmts = append(currTaskStmts, stmt)
		} else {
			if !stmt.hasOneParent() {
				task := createTaskFromStatements(currTaskStmts, wf)
				task.Name = fmt.Sprintf("%s-%d", wf.Name, len(tasks))
				tasks = append(tasks, task)
				currTaskStmts = nil
			}
			currTaskStmts = append(currTaskStmts, stmt)
		}

		if !stmt.hasOneConsumer() {
			task := createTaskFromStatements(currTaskStmts, wf)
			task.Name = fmt.Sprintf("%s-%d", wf.Name, len(tasks))
			tasks = append(tasks, task)
			currTaskStmts = nil
		}

		for _, ret := range stmt.Rets {
			for retDestIdx, retDest := range ret.DestStmt {
				retDest.parentProcessed[ret.DestStmtInIdx[retDestIdx]] = true
				if allTrue(retDest.parentProcessed) {
					stack = append(stack, retDest)
					stackSize++
				}
			}
		}
	}

	if len(currTaskStmts) > 0 {
		task := createTaskFromStatements(currTaskStmts, wf)
		task.Name = fmt.Sprintf("%s-%d", wf.Name, len(tasks))
		tasks = append(tasks, task)
	}

	wf.TotalTasks = uint32(len(tasks))

	return nil
}

func fullPartition(c *Composition, wf *workflow.Workflow) []*workflow.Task {
	var tasks []*workflow.Task

	// convert each statement to a task
	stmtToTask := make(map[string]*workflow.Task)
	for _, stmt := range c.Statements {
		task := &workflow.Task{
			Name:      fmt.Sprintf("%s-%s", wf.Name, stmt.Name),
			Functions: []string{stmt.Name},
			NumIn:     uint32(len(stmt.Args)),
			NumOut:    uint32(len(stmt.Rets)),
		}
		stmtToTask[stmt.Name] = task
		tasks = append(tasks, task)
	}

	// set consumers
	for _, stmt := range c.Statements {
		task := stmtToTask[stmt.Name]
		for retIdx, ret := range stmt.Rets {
			for destIdx, dest := range ret.DestStmt {
				destTask := stmtToTask[dest.Name]
				task.ConsumerTasks = append(task.ConsumerTasks, destTask)
				task.ConsumerDataSrcIdx = append(task.ConsumerDataSrcIdx, int32(retIdx))
				task.ConsumerDataDstIdx = append(task.ConsumerDataDstIdx, int32(ret.DestStmtInIdx[destIdx]))
			}
		}
	}

	// update workflow -> set initial tasks and output tasks
	for stmtIdx, stmt := range c.Consumers {
		wf.InitialTasks = append(wf.InitialTasks, stmtToTask[stmt.Name])
		wf.InitialDataDstIdx = append(wf.InitialDataDstIdx, int32(c.ConsumerArgIdx[stmtIdx]))
		wf.InitialDataSrcIdx = append(wf.InitialDataSrcIdx, int32(c.ConsumerOutIdx[stmtIdx]))
	}
	for stmtIdx, stmt := range c.outStmts {
		wf.OutTasks = append(wf.OutTasks, stmtToTask[stmt.Name])
		wf.OutDataSrcIdx = append(wf.OutDataSrcIdx, int32(c.outStmtRetIdx[stmtIdx]))
	}

	wf.TotalTasks = uint32(len(tasks))

	return tasks
}

func noPartition(c *Composition, wf *workflow.Workflow) []*workflow.Task {
	numStmts := len(c.Statements)
	task := &workflow.Task{
		Name:           fmt.Sprintf("%s-noPartition", wf.Name),
		Functions:      make([]string, numStmts),
		FunctionOutNum: make([]int32, numStmts),
		FunctionInNum:  make([]int32, numStmts),
		NumIn:          uint32(len(c.params)),
		NumOut:         uint32(len(c.returns)),
	}

	// add all functions from composition + set internal data flow information
	fToIdx := make(map[string]int)
	for stmtIdx, stmt := range c.Statements {
		task.Functions[stmtIdx] = stmt.Name
		task.FunctionOutNum[stmtIdx] = int32(len(stmt.Rets))
		fToIdx[stmt.Name] = stmtIdx
	}
	for stmtIdx, stmt := range c.Statements {
		stmtArgs := stmt.Args
		task.FunctionInNum[stmtIdx] = int32(len(stmtArgs))

		for _, arg := range stmtArgs {
			var argStmtIdx int
			if arg.SrcStmt == nil {
				argStmtIdx = -1
			} else {
				argStmtIdx = fToIdx[arg.SrcStmt.Name]
			}
			argStmtOutIdx := arg.SrcStmtOutIdx
			task.FunctionDataFlow = append(task.FunctionDataFlow, int32(argStmtIdx), int32(argStmtOutIdx))
		}
	}
	for stmtIdx, stmt := range c.outStmts {
		argStmtIdx := fToIdx[stmt.Name]
		argStmtOutIdx := c.outStmtRetIdx[stmtIdx]
		task.FunctionDataFlow = append(task.FunctionDataFlow, int32(argStmtIdx), int32(argStmtOutIdx))
		wf.OutTasks = append(wf.OutTasks, task)
		wf.OutDataSrcIdx = append(wf.OutDataSrcIdx, int32(stmtIdx))
	}

	// update workflow -> set initial task
	for i := 0; i < len(c.Consumers); i++ {
		wf.InitialTasks = append(wf.InitialTasks, task)
		wf.InitialDataSrcIdx = append(wf.InitialDataSrcIdx, int32(c.ConsumerOutIdx[i]))
		wf.InitialDataDstIdx = append(wf.InitialDataSrcIdx, int32(c.ConsumerArgIdx[i]))
	}

	wf.TotalTasks = 1

	return []*workflow.Task{task}
}

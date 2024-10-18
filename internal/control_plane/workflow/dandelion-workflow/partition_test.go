package dandelion_workflow

import (
	"bufio"
	"cluster_manager/internal/control_plane/workflow"
	"fmt"
	"strings"
	"testing"
)

func partitionFunctions() []PartitionMethod {
	return []PartitionMethod{
		FullPartition,
		NoPartition,
		ConsumerBased,
	}
}

type testRunner struct {
	taskOnly    bool // will construct output only considering tasks (ignoring functions of tasks)
	dataTaskIn  map[*workflow.Task][]string
	dataTaskOut map[*workflow.Task][]string
}

func newTestRunner(taskOnly bool) *testRunner {
	return &testRunner{
		taskOnly:    taskOnly,
		dataTaskIn:  make(map[*workflow.Task][]string),
		dataTaskOut: make(map[*workflow.Task][]string),
	}
}

func isRunnable(d []string) bool {
	for _, s := range d {
		if s == "" {
			return false
		}
	}
	return true
}

func constructOutput(name string, inData []string, numOut int) []string {
	out := name + "("
	for _, data := range inData {
		out += data + ","
	}
	if len(inData) > 0 {
		out = out[:len(out)-1]
	}
	out += ")"
	outData := make([]string, numOut)
	if numOut > 1 {
		for i := 0; i < int(numOut); i++ {
			outData[i] = fmt.Sprintf("%s[%d]", out, i)
		}
	} else if numOut == 1 {
		outData[0] = out
	}
	return outData
}

func (tr *testRunner) runTaskWithFunctions(task *workflow.Task) []string {
	if len(task.Functions) == 1 {
		outData := constructOutput(task.Functions[0], tr.dataTaskIn[task], int(task.NumOut))
		tr.dataTaskOut[task] = outData
		return outData
	}

	data := make([][]string, len(task.Functions)+1)
	data[0] = tr.dataTaskIn[task]

	fProcessed := make([]bool, len(task.Functions)+1)
	fProcessed[0] = true
	allProcessed := false
	for !allProcessed {
		allProcessed = true
		dfIdx := 0

		for fIdx := 0; fIdx < len(task.Functions); fIdx++ {
			if !fProcessed[fIdx+1] {
				fInData := make([]string, task.FunctionInNum[fIdx])
				runnable := true
				for argIdx := 0; argIdx < int(task.FunctionInNum[fIdx]); argIdx++ {
					srcIdx := task.FunctionDataFlow[dfIdx+(2*argIdx)]
					outIdx := task.FunctionDataFlow[dfIdx+(2*argIdx)+1]
					if !fProcessed[srcIdx+1] {
						runnable = false
						break
					}
					fInData[argIdx] = data[srcIdx+1][outIdx]
				}
				if runnable {
					data[fIdx+1] = constructOutput(task.Functions[fIdx], fInData, int(task.FunctionOutNum[fIdx]))
					fProcessed[fIdx+1] = true
				} else {
					allProcessed = false
				}
			}
			dfIdx += 2 * int(task.FunctionInNum[fIdx])
		}
	}

	outData := make([]string, task.NumOut)
	dfIdx := len(task.FunctionDataFlow) - 2*int(task.NumOut)
	for outIdx := 0; outIdx < int(task.NumOut); outIdx++ {
		srcIdx := task.FunctionDataFlow[dfIdx]
		srcOutIdx := task.FunctionDataFlow[dfIdx+1]
		dfIdx += 2
		outData[outIdx] = data[srcIdx+1][srcOutIdx]
	}

	tr.dataTaskOut[task] = outData
	return outData
}

func (tr *testRunner) runTaskWithoutFunctions(task *workflow.Task) []string {
	taskName := strings.Split(task.Name, "_")[2]
	outData := constructOutput(taskName, tr.dataTaskIn[task], int(task.NumOut))
	tr.dataTaskOut[task] = outData
	return outData
}

func (tr *testRunner) run(wf *workflow.Workflow, inData []string) ([]string, error) {
	var queue []*workflow.Task
	for taskIdx, task := range wf.InitialTasks {
		taskData, ok := tr.dataTaskIn[task]
		if !ok {
			taskData = make([]string, task.NumIn)
			tr.dataTaskIn[task] = taskData
		}
		taskData[wf.InitialDataDstIdx[taskIdx]] = inData[wf.InitialDataSrcIdx[taskIdx]]
		tr.dataTaskIn[task] = taskData
		if isRunnable(taskData) {
			queue = append(queue, task)
		}
	}

	tasksDone := 0
	for len(queue) > 0 {
		currStmt := queue[0]
		queue = queue[1:]

		if tr.taskOnly {
			tr.runTaskWithoutFunctions(currStmt)
		} else {
			tr.runTaskWithFunctions(currStmt)
		}
		outData := tr.dataTaskOut[currStmt]
		tasksDone++

		for nextIdx, nextTask := range currStmt.ConsumerTasks {
			_, ok := tr.dataTaskIn[nextTask]
			if !ok {
				tr.dataTaskIn[nextTask] = make([]string, nextTask.NumIn)
			}
			tr.dataTaskIn[nextTask][currStmt.ConsumerDataDstIdx[nextIdx]] = outData[currStmt.ConsumerDataSrcIdx[nextIdx]]
			if isRunnable(tr.dataTaskIn[nextTask]) {
				queue = append(queue, nextTask)
			}
		}
	}

	if tasksDone != int(wf.TotalTasks) {
		return nil, fmt.Errorf("not all tasks were run (%d/%d)", tasksDone, wf.TotalTasks)
	}

	wfOutData := make([]string, len(wf.OutTasks))
	for taskIdx, task := range wf.OutTasks {
		wfOutData[taskIdx] = tr.dataTaskOut[task][wf.OutDataSrcIdx[taskIdx]]
	}

	return wfOutData, nil
}

func TestFunctionRunner(t *testing.T) {
	testTask := workflow.Task{
		Name:           "Test",
		NumIn:          2,
		NumOut:         3,
		Functions:      []string{"A", "B", "C", "D", "E", "F", "G"},
		FunctionInNum:  []int32{1, 1, 2, 2, 3, 0, 2},
		FunctionOutNum: []int32{2, 1, 2, 1, 1, 4, 0},
		FunctionDataFlow: []int32{ // pair {src function, src data idx} for every input
			-1, 0, // a
			0, 0, // b
			1, 0, 3, 0, // c
			0, 1, -1, 1, // d
			2, 1, 3, 0, 5, 0, // e
			5, 3, 5, 2, // g
			2, 0, 4, 0, 5, 1, // composition out
		},
	}

	tr := newTestRunner(false)
	tr.dataTaskIn[&testTask] = []string{"a", "b"}

	tr.runTaskWithFunctions(&testTask)
	taskOut := tr.dataTaskOut[&testTask]

	// A(a)[0], A(a)[1]
	// B(A(a)[0])
	// C(B(A(a)[0]),D(A(a)[1],b))[0], C(B(A(a)[0]),D(A(a)[1],b))[1]
	// D(A(a)[1],b)
	// E(C(B(A(a)[0]),D(A(a)[1],b))[1],D(A(a)[1],b),F()[0])
	// F()[0], F()[1], F()[2], F()[3]
	// G(F()[3],F()[2]) -> does not exist (no output)
	expectedOut := []string{
		"C(B(A(a)[0]),D(A(a)[1],b))[0]",
		"E(C(B(A(a)[0]),D(A(a)[1],b))[1],D(A(a)[1],b),F()[0])",
		"F()[1]",
	}

	for i := 0; i < len(expectedOut); i++ {
		if taskOut[i] != expectedOut[i] {
			t.Errorf("task output mismatch:\n expected: %s\n got:      %s", expectedOut[i], taskOut[i])
		}
	}

}

func TestPartition(t *testing.T) {
	inputs := []string{
		`
		(:function FunA (A) -> (B))
		(:composition c1 (InputA) -> (OutputB) (
			(FunA ((A <- InputA)) => ((OutputB := B)))
		))
		`,
		`
		(:function FunA (A B) -> (C D))
		(:function FunB (C D) -> (E F))
		(:function FunC (B D F G) -> (H))
    
		(:composition c2 (InputA InputB InputG) -> (OutputE OutputH) (
			(FunA ((A <- InputA) (B <- InputB)) => ((InterC := C) (InterD := D)))
			(FunB ((C <- InterC) (D <- InterD)) => ((OutputE := E) (InterF := F)))
			(FunC ((B <- InputB) (D <- InterD) (F <- InterF) (G <- InputG)) => ((OutputH := H)))
		))
		`,
		`
		(:function FunA (A) -> (B C))
		(:function FunB (B C) -> (D E))
		(:function FunC (D) -> (F))
		(:function FunD (E) -> (G))

		(:composition c3 (InputA) -> (OutputF OutputG) (
			(FunA ((A <- InputA)) => ((InterB := B) (InterC := C)))
			(FunB ((B <- InterB) (C <- InterC)) => ((InterD := D) (InterE := E)))
			(FunC ((D <- InterD)) => ((OutputF := F)))
			(FunD ((E <- InterE)) => ((OutputG := G)))
		))
		`,
	}
	inData := [][]string{
		{"a"},
		{"a", "b", "g"},
		{"a"},
	}
	expectedTaskOutput := [][][]string{
		{ // full partition
			{"FunA0(a)"},
			{
				"FunB1(FunA0(a,b)[0],FunA0(a,b)[1])[0]",
				"FunC2(b,FunA0(a,b)[1],FunB1(FunA0(a,b)[0],FunA0(a,b)[1])[1],g)",
			},
			{
				"FunC2(FunB1(FunA0(a)[0],FunA0(a)[1])[0])",
				"FunD3(FunB1(FunA0(a)[0],FunA0(a)[1])[1])",
			},
		},
		{ // no partition
			{"task(a)"},
			{
				"task(a,b,g)[0]",
				"task(a,b,g)[1]",
			},
			{
				"task(a)[0]",
				"task(a)[1]",
			},
		},
		{ // transformation based
			{"0(a)"},
			{
				"1(0(a,b)[0],0(a,b)[1])[0]",
				"2(b,0(a,b)[1],1(0(a,b)[0],0(a,b)[1])[1],g)",
			},
			{
				"2(0(a)[0])",
				"1(0(a)[1])",
			},
		},
	}
	expectedOutput := [][]string{
		{"FunA(a)"},
		{
			"FunB(FunA(a,b)[0],FunA(a,b)[1])[0]",
			"FunC(b,FunA(a,b)[1],FunB(FunA(a,b)[0],FunA(a,b)[1])[1],g)",
		},
		{
			"FunC(FunB(FunA(a)[0],FunA(a)[1])[0])",
			"FunD(FunB(FunA(a)[0],FunA(a)[1])[1])",
		},
	}

	for partFuncIdx, partitionFunc := range partitionFunctions() {
		for i := 0; i < len(inputs); i++ {
			parser := NewParser(bufio.NewReader(strings.NewReader(inputs[i])))
			dwf, err := parser.Parse()
			if err != nil {
				t.Errorf("Got error while parsing input: %v", err)
				return
			}
			dwf.Name = "wf"
			wfs, _, err := dwf.ExportWorkflow(partitionFunc)
			if err != nil {
				t.Errorf("Got error while exporting workflow: %v", err)
				return
			}

			if len(wfs) != 1 {
				t.Errorf("Expected to get only 1 workflow got %d", len(wfs))
				return
			}
			wf := wfs[0]

			// check partitioned tasks
			trTasks := newTestRunner(true)
			taskOutData, err := trTasks.run(wf, inData[i])
			if err != nil {
				t.Errorf("%v (partition function #%d)", err, partFuncIdx)
				return
			}
			for idx, expected := range expectedTaskOutput[partFuncIdx][i] {
				if taskOutData[idx] != expected {
					t.Errorf(
						"Got unexpected output: %v (expected: %v) (testcase #%d, partition function #%d)",
						taskOutData[idx], expected, i, partFuncIdx,
					)
				}
			}

			// check that overall execution remains the same
			trGeneral := newTestRunner(false)
			wfOutData, err := trGeneral.run(wf, inData[i])
			if err != nil {
				t.Errorf("%v (partition function #%d)", err, partFuncIdx)
				return
			}
			for idx, expected := range expectedOutput[i] {
				if wfOutData[idx] != expected {
					t.Errorf(
						"Got unexpected output: %v (expected: %v) (testcase #%d, partition function #%d)",
						wfOutData[idx], expected, i, partFuncIdx,
					)
				}
			}
		}
	}
}

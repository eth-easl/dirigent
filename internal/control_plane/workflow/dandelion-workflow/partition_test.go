package dandelion_workflow

import (
	"bufio"
	"cluster_manager/internal/control_plane/workflow"
	"fmt"
	"slices"
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

func inputToWorkflow(input string, partitionFunc PartitionMethod) (*workflow.Workflow, error) {
	parser := NewParser(bufio.NewReader(strings.NewReader(input)))
	dwf, err := parser.Parse()
	if err != nil {
		return nil, fmt.Errorf("Got error while parsing input: %v", err)
	}
	dwf.Name = "wf"
	wfs, _, err := dwf.ExportWorkflow(partitionFunc)
	if err != nil {
		return nil, fmt.Errorf("Got error while exporting workflow: %v", err)
	}

	if len(wfs) != 1 {
		return nil, fmt.Errorf("Expected to get only 1 workflow got %d", len(wfs))
	}

	return wfs[0], nil
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
	// G(F()[3],F()[2]) => does not exist (no output)
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

//-------
// Tests

func TestPartition(t *testing.T) {
	inputs := []string{
		`
		function FunA (A) => (B);
		composition c1 (InputA) => (OutputB) {
			FunA (A = all InputA) => (OutputB = B);
		}
		`,
		`
		function FunA (A, B) => (C, D);
		function FunB (C, D) => (E, F);
		function FunC (B, D, F, G) => (H);
    
		composition c2 (InputA, InputB, InputG) => (OutputE, OutputH) {
			FunA (A = all InputA, B = all InputB) => (InterC = C, InterD = D);
			FunB (C = all InterC, D = all InterD) => (OutputE = E, InterF = F);
			FunC (B = all InputB, D = all InterD, F = all InterF, G = all InputG) => (OutputH = H);
		}
		`,
		`
		function FunA (A) => (B, C);
		function FunB (B, C) => (D, E);
		function FunC (D) => (F);
		function FunD (E) => (G);

		composition c3 (InputA) => (OutputF, OutputG) {
			FunA (A = all InputA) => (InterB = B, InterC = C);
			FunB (B = all InterB, C = all InterC) => (InterD = D, InterE = E);
			FunC (D = all InterD) => (OutputF = F);
			FunD (E = all InterE) => (OutputG = G);
		}
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
		{ // consumer based
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
			// load and partition workflow
			wf, err := inputToWorkflow(inputs[i], partitionFunc)
			if err != nil {
				t.Fatalf(err.Error())
			}

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

func TestParallelPartitionSingleInput(t *testing.T) {
	keywords := []string{"all", "keyed", "each"}
	inputs := make([]string, 0, len(keywords)*len(keywords))
	for _, keyword1 := range keywords {
		for _, keyword2 := range keywords {
			input := fmt.Sprintf(
				`
				function FunA (A) => (B);
				composition c1 (In) => (Out) {
					FunA (A = %s In) => (Inter = B);
					FunA (A = %s Inter) => (Out = B);
				}`,
				keyword1, keyword2,
			)
			inputs = append(inputs, input)
		}
	}
	expectedTaskSharding := [][][]workflow.Sharding{
		{ // full partition
			{workflow.ShardingAll, workflow.ShardingAll},
			{workflow.ShardingAll, workflow.ShardingKeyed},
			{workflow.ShardingAll, workflow.ShardingEach},
			{workflow.ShardingKeyed, workflow.ShardingAll},
			{workflow.ShardingKeyed, workflow.ShardingKeyed},
			{workflow.ShardingKeyed, workflow.ShardingEach},
			{workflow.ShardingEach, workflow.ShardingAll},
			{workflow.ShardingEach, workflow.ShardingKeyed},
			{workflow.ShardingEach, workflow.ShardingEach},
		},
		{ // no partition
			{workflow.ShardingAll},
			{workflow.ShardingAll},
			{workflow.ShardingAll},
			{workflow.ShardingAll},
			{workflow.ShardingAll},
			{workflow.ShardingKeyed},
			{workflow.ShardingAll},
			{workflow.ShardingAll},
			{workflow.ShardingEach},
		},
		{ // consumer based
			{workflow.ShardingAll},
			{workflow.ShardingAll, workflow.ShardingKeyed},
			{workflow.ShardingAll, workflow.ShardingEach},
			{workflow.ShardingKeyed, workflow.ShardingAll},
			{workflow.ShardingKeyed, workflow.ShardingKeyed},
			{workflow.ShardingKeyed},
			{workflow.ShardingEach, workflow.ShardingAll},
			{workflow.ShardingEach, workflow.ShardingKeyed},
			{workflow.ShardingEach},
		},
	}

	for partFuncIdx, partitionFunc := range partitionFunctions() {
		for tcIdx := 0; tcIdx < len(inputs); tcIdx++ {
			// load and partition workflow
			wf, err := inputToWorkflow(inputs[tcIdx], partitionFunc)
			if err != nil {
				t.Fatalf(err.Error())
			}

			// check number of tasks
			if int(wf.TotalTasks) != len(expectedTaskSharding[partFuncIdx][tcIdx]) {
				t.Errorf(
					"Got wrong number of total tasks: expected %d, got %d (test case #%d, partition function #%d)",
					len(expectedTaskSharding[partFuncIdx][tcIdx]), int(wf.TotalTasks), tcIdx, partFuncIdx,
				)
				continue
			}

			// check sharding
			nextTask := wf.InitialTasks[0]
			for taskNum, expected := range expectedTaskSharding[partFuncIdx][tcIdx] {
				if nextTask.InputSharding[0] != expected {
					t.Errorf(
						"Got unexpected task sharding: expected %d, got %d (task #%d, test case #%d, partition function #%d)",
						expected, nextTask.InputSharding[0], taskNum, tcIdx, partFuncIdx,
					)
					continue
				}
				if taskNum != len(expectedTaskSharding[partFuncIdx][tcIdx])-1 {
					nextTask = nextTask.ConsumerTasks[0]
				}
			}
		}
	}
}

func TestParallelPartitionMultiInput(t *testing.T) {
	keywords := []string{"all", "keyed", "each"}
	inputs := make([]string, 0, len(keywords)*len(keywords)*len(keywords)*len(keywords))
	for _, f1in1Keyword := range keywords {
		for _, f1in2Keyword := range keywords {
			for _, f2in1Keyword := range keywords {
				for _, f2in2Keyword := range keywords {
					input := fmt.Sprintf(
						`
						function FunA (A, B) => (C, D);
						composition c1 (InA, InB) => (OutC, OutD) {
							FunA (A = %s InA, B = %s InB) => (InterA = C, InterB = D);
							FunA (A = %s InterA, B = %s InterB) => (OutC = C, OutD = D);
						}`,
						f1in1Keyword, f1in2Keyword, f2in1Keyword, f2in2Keyword,
					)
					inputs = append(inputs, input)
				}
			}
		}
	}
	sharding := []workflow.Sharding{workflow.ShardingAll, workflow.ShardingKeyed, workflow.ShardingEach}
	expectedTaskSharding := make([][][][]workflow.Sharding, 3)
	expectedTaskSharding[0] = make([][][]workflow.Sharding, 0, len(keywords)*len(keywords)*len(keywords)*len(keywords))
	expectedTaskSharding[1] = make([][][]workflow.Sharding, 0, len(keywords)*len(keywords)*len(keywords)*len(keywords))
	expectedTaskSharding[2] = make([][][]workflow.Sharding, 0, len(keywords)*len(keywords)*len(keywords)*len(keywords))
	for _, f1in1Sharding := range sharding {
		for _, f1in2Sharding := range sharding {
			for _, f2in1Sharding := range sharding {
				for _, f2in2Sharding := range sharding {
					// general case
					fullPartitionSharding := [][]workflow.Sharding{
						{f1in1Sharding, f1in2Sharding}, {f2in1Sharding, f2in2Sharding},
					}
					noPartitionSharding := [][]workflow.Sharding{
						{workflow.ShardingAll, workflow.ShardingAll},
					}
					consumerBasedPartitionSharding := [][]workflow.Sharding{
						{f1in1Sharding, f1in2Sharding}, {f2in1Sharding, f2in2Sharding},
					}

					// special cases
					if f2in1Sharding == workflow.ShardingEach && f2in2Sharding == workflow.ShardingEach {
						noPartitionSharding[0] = []workflow.Sharding{f1in1Sharding, f1in2Sharding}
						if f1in1Sharding != workflow.ShardingAll || f1in2Sharding != workflow.ShardingAll {
							consumerBasedPartitionSharding = [][]workflow.Sharding{
								{f1in1Sharding, f1in2Sharding},
							}
						}
					}

					expectedTaskSharding[0] = append(expectedTaskSharding[0], fullPartitionSharding)
					expectedTaskSharding[1] = append(expectedTaskSharding[1], noPartitionSharding)
					expectedTaskSharding[2] = append(expectedTaskSharding[2], consumerBasedPartitionSharding)
				}
			}
		}
	}
	expectedTaskSharding[2][0] = [][]workflow.Sharding{{workflow.ShardingAll, workflow.ShardingAll}}

	for partFuncIdx, partitionFunc := range partitionFunctions() {
		if partFuncIdx != 2 {
			continue
		}
		for tcIdx := 16; tcIdx < len(inputs); tcIdx++ {
			// load and partition workflow
			wf, err := inputToWorkflow(inputs[tcIdx], partitionFunc)
			if err != nil {
				t.Fatalf(err.Error())
			}

			// check number of tasks
			if int(wf.TotalTasks) != len(expectedTaskSharding[partFuncIdx][tcIdx]) {
				t.Errorf(
					"Got wrong number of total tasks: expected %d, got %d (test case #%d, partition function #%d)",
					len(expectedTaskSharding[partFuncIdx][tcIdx]), int(wf.TotalTasks), tcIdx, partFuncIdx,
				)
				continue
			}

			// check sharding
			nextTask := wf.InitialTasks[0]
			for taskNum, expected := range expectedTaskSharding[partFuncIdx][tcIdx] {
				if len(nextTask.InputSharding) != len(expected) {
					t.Errorf(
						"Got unexpected input sharding size: expected %d, got %d (task #%d, test case #%d, partition function #%d)",
						len(expected), len(nextTask.InputSharding), taskNum, tcIdx, partFuncIdx,
					)
					continue
				}
				for i := 0; i < len(expected); i++ {
					if nextTask.InputSharding[i] != expected[i] {
						t.Errorf(
							"Got unexpected task sharding for input #%d: expected %d, got %d (task #%d, test case #%d, partition function #%d)",
							i, expected[0], nextTask.InputSharding[0], taskNum, tcIdx, partFuncIdx,
						)
						continue
					}
				}
				if taskNum != len(expectedTaskSharding[partFuncIdx][tcIdx])-1 {
					nextTask = nextTask.ConsumerTasks[0]
				}
			}
		}
	}
}

func TestDataQueryWorkflow(t *testing.T) {
	input := `
		function csv_reader (option, data) => (outSchema, outBatches);
		function csv_writer (option, inSchema, inBatches) => (data);
		function filter (option, inSchema, inBatches) => (outSchema, outBatches);
		function hash_join (option, inSchema, inBatches, inSchema2, inBatches2) => (outSchema, outBatches);
		function project (option, inSchema, inBatches) => (outSchema, outBatches);
		function group_by (option, inSchema, inBatches) => (outSchema, outBatches);
		function order_by (option, inSchema, inBatches) => (outSchema, outBatches);
		composition q1 (ropt0, fopt0, ropt1, fopt1, hopt2, popt2, oopt2, gopt2, wopt2, data0, data1) => (out) {
			csv_reader (option = all ropt0, data = each data0) => (rs0 = outSchema, rb0 = outBatches);
			filter (option = all fopt0, inSchema = any rs0, inBatches = each rb0) => (fs0 = outSchema, fb0 = outBatches);

			csv_reader (option = all ropt1, data = each data1) => (rs1 = outSchema, rb1 = outBatches);
			filter (option = all fopt1, inSchema = any rs1, inBatches = each rb1) => (fs1 = outSchema, fb1 = outBatches);

			hash_join (option = all hopt2, inSchema = any fs0, inBatches = all fb0, inSchema2 = any fs1, inBatches2 = all fb1) => (hs2 = outSchema, hb2 = outBatches);
			project (option = all popt2, inSchema = any hs2, inBatches = each hb2) => (ps2 = outSchema, pb2 = outBatches);
			group_by (option = all gopt2, inSchema = any ps2, inBatches = all pb2) => (gs2 = outSchema, gb2 = outBatches);
			order_by (option = all oopt2, inSchema = any gs2, inBatches = all gb2) => (os2 = outSchema, ob2 = outBatches);
			csv_writer (option = all wopt2, inSchema = any os2, inBatches = all ob2) => (out = data);
		}
		`
	expectedTasks := [][]string{
		{"csv_reader", "filter"},
		{"csv_reader", "filter"},
		{"hash_join"},
		{"project"},
		{"group_by", "order_by", "csv_writer"},
	}

	// load and partition workflow
	wf, err := inputToWorkflow(input, ConsumerBased)
	if err != nil {
		t.Fatalf(err.Error())
	}

	// check tasks
	if wf.TotalTasks != uint32(5) {
		t.Errorf("Expected 5 tasks, got %d", wf.TotalTasks)
	}
	tasksFound := make([]bool, len(expectedTasks))
	var tasksChecked []*workflow.Task
	queue := wf.InitialTasks
	for len(queue) > 0 {
		currentTask := queue[0]
		queue = queue[1:]

		if slices.Contains(tasksChecked, currentTask) {
			continue
		}
		tasksChecked = append(tasksChecked, currentTask)

		taskFound := false
		for expectedIdx, expectedTask := range expectedTasks {
			if tasksFound[expectedIdx] {
				continue
			}
			if slices.Equal(currentTask.Functions, expectedTask) {
				tasksFound[expectedIdx] = true
				taskFound = true
				break
			}
		}
		if !taskFound {
			t.Errorf("Task not found in expected tasks: %v", currentTask)
		}

		queue = append(queue, currentTask.ConsumerTasks...)
	}
	if len(tasksChecked) != len(expectedTasks) {
		t.Errorf("Expected %d tasks, got %d", len(expectedTasks), len(tasksChecked))
	}
}

func TestDebug(t *testing.T) {
	input := `
		function csv_reader (options, inData) => (outSchema, outBatches);
		function csv_writer (options, inSchema, inBatches) => (outData);
	
		composition test (in_data, reader_options) => (out_data) {
		  csv_reader (options = all reader_options, inData = each in_data) => (schema = outSchema, batches = outBatches);
		  csv_writer (options = all reader_options, inSchema = any schema, inBatches = each batches) => (out_data = outData);
		}`
	inData := []string{"inCSV", "ropt"}
	expectedTaskOutput := []string{"task(inCSV,ropt)"}
	expectedOutput := []string{"csv_writer(ropt,csv_reader(ropt,inCSV)[0],csv_reader(ropt,inCSV)[1])"}

	// load and partition workflow
	wf, err := inputToWorkflow(input, NoPartition)
	if err != nil {
		t.Fatalf(err.Error())
	}

	// check partitioned tasks
	trTasks := newTestRunner(true)
	taskOutData, err := trTasks.run(wf, inData)
	if err != nil {
		t.Errorf("%v", err)
		return
	}
	for idx, expected := range expectedTaskOutput {
		if taskOutData[idx] != expected {
			t.Errorf(
				"Got unexpected output: %v (expected: %v)",
				taskOutData[idx], expected,
			)
		}
	}

	// check that overall execution remains the same
	trGeneral := newTestRunner(false)
	wfOutData, err := trGeneral.run(wf, inData)
	if err != nil {
		t.Errorf("%v", err)
		return
	}
	for idx, expected := range expectedOutput {
		if wfOutData[idx] != expected {
			t.Errorf(
				"Got unexpected output: %v (expected: %v)",
				wfOutData[idx], expected,
			)
		}
	}

}

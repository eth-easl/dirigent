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

func isRunnable(d []string) bool {
	for _, s := range d {
		if s == "" {
			return false
		}
	}
	return true
}

func applyFunc(task *workflow.Task, dataTracker map[*workflow.Task][]string) []string {
	inData := dataTracker[task]
	out := "("
	for _, data := range inData {
		out += data + ","
	}
	if len(inData) > 0 {
		out = out[:len(out)-1]
	}
	out += ")->" + task.Name
	outData := make([]string, task.NumOut)
	if task.NumOut > 1 {
		for i := 0; i < int(task.NumOut); i++ {
			outData[i] = fmt.Sprintf("%s[%d]", out, i)
		}
	} else if task.NumOut == 1 {
		outData[0] = out
	}
	return outData
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
		{"A"},
		{"A", "B", "G"},
		{"A"},
	}
	expectedOutput := [][][]string{
		{ // full partition
			{"(A)->wf_c1_FunA"},
			{
				"((A,B)->wf_c2_FunA[0],(A,B)->wf_c2_FunA[1])->wf_c2_FunB[0]",
				"(B,(A,B)->wf_c2_FunA[1],((A,B)->wf_c2_FunA[0],(A,B)->wf_c2_FunA[1])->wf_c2_FunB[1],G)->wf_c2_FunC",
			},
			{
				"(((A)->wf_c3_FunA[0],(A)->wf_c3_FunA[1])->wf_c3_FunB[0])->wf_c3_FunC",
				"(((A)->wf_c3_FunA[0],(A)->wf_c3_FunA[1])->wf_c3_FunB[1])->wf_c3_FunD",
			},
		},
		{ // no partition
			{"(A)->wf_c1_noPartition"},
			{
				"(A,B,G)->wf_c2_task[0]",
				"(A,B,G)->wf_c2_task[1]",
			},
			{
				"(A)->wf_c3_task[0]",
				"(A)->wf_c3_task[1]",
			},
		},
		{ // transformation based
			{"(A)->wf_c1_0"},
			{
				"((A,B)->wf_c2_0[0],(A,B)->wf_c2_0[1])->wf_c2_1[0]",
				"(B,(A,B)->wf_c2_0[1],((A,B)->wf_c2_0[0],(A,B)->wf_c2_0[1])->wf_c2_1[1],G)->wf_c2_2",
			},
			{
				"((A)->wf_c3_0[1])->wf_c3_1",
				"((A)->wf_c3_0[0])->wf_c3_2",
			},
		},
	}

	for partFuncIdx, partitionFunc := range partitionFunctions() {
		if partFuncIdx != 2 {
			continue
		}
		for i := 2; i < len(inputs); i++ {
			dataInTracker := make(map[*workflow.Task][]string)
			dataOutTracker := make(map[*workflow.Task][]string)

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

			var queue []*workflow.Task
			for taskIdx, task := range wf.InitialTasks {
				taskData, ok := dataInTracker[task]
				if !ok {
					taskData = make([]string, task.NumIn)
					dataInTracker[task] = taskData
				}
				taskData[wf.InitialDataDstIdx[taskIdx]] = inData[i][wf.InitialDataSrcIdx[taskIdx]]
				dataInTracker[task] = taskData
				if isRunnable(taskData) {
					queue = append(queue, task)
				}
			}

			tasksDone := 0
			for len(queue) > 0 {
				currStmt := queue[0]
				queue = queue[1:]

				outData := applyFunc(currStmt, dataInTracker)
				dataOutTracker[currStmt] = outData
				tasksDone++

				for nextIdx, nextTask := range currStmt.ConsumerTasks {
					_, ok := dataInTracker[nextTask]
					if !ok {
						dataInTracker[nextTask] = make([]string, nextTask.NumIn)
					}
					dataInTracker[nextTask][currStmt.ConsumerDataDstIdx[nextIdx]] = outData[currStmt.ConsumerDataSrcIdx[nextIdx]]
					if isRunnable(dataInTracker[nextTask]) {
						queue = append(queue, nextTask)
					}
				}
			}

			if tasksDone != int(wf.TotalTasks) {
				t.Errorf("Not all tasks were run (%d/%d) (partition function #%d)", tasksDone, wf.TotalTasks, partFuncIdx)
				return
			}

			wfOutData := make([]string, len(wf.OutTasks))
			for taskIdx, task := range wf.OutTasks {
				wfOutData[taskIdx] = dataOutTracker[task][wf.OutDataSrcIdx[taskIdx]]
			}

			for idx, expected := range expectedOutput[partFuncIdx][i] {
				if wfOutData[idx] != expected {
					t.Errorf("Got unexpected output: %v (expected: %v) (partition function #%d)", wfOutData[idx], expected, partFuncIdx)
				}
			}
		}
	}
}

package scheduler

import (
	"bufio"
	cp_workflow "cluster_manager/internal/control_plane/workflow"
	"cluster_manager/internal/control_plane/workflow/dandelion-workflow"
	dp_workflow "cluster_manager/internal/data_plane/workflow"
	"cluster_manager/pkg/config"
	"context"
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"
)

var testedSchedulers = []SchedulerType{
	SequentialFifo,
	ConcurrentFifo,
}

func keyToIdx(key rune) int {
	return int(key) - 97
}
func constructOutput(name string, inData []*dp_workflow.Data, numOut int, outNumItems []int, suffix string) []*dp_workflow.Data {
	out := name + "("
	for _, data := range inData {
		str := ""
		byteItems := data.GetBytes()
		sort.Slice(byteItems, func(i, j int) bool {
			b_i := byteItems[i]
			b_j := byteItems[j]
			return int(b_i[len(b_i)-1]) < int(b_j[len(b_j)-1])
		})
		for i, b := range byteItems {
			str += string(b)
			if i != len(byteItems)-1 {
				str += "&"
			}
		}
		out += str + ","
	}
	if len(inData) > 0 {
		out = out[:len(out)-1]
	}
	out += ")"
	outData := make([]*dp_workflow.Data, numOut)
	if numOut > 1 {
		for i := 0; i < numOut; i++ {
			if outNumItems[i] > 1 {
				outItems := make([][]byte, outNumItems[i])
				for j := 0; j < outNumItems[i]; j++ {
					outItems[j] = []byte(fmt.Sprintf("%s[%d][%d]%s", out, i, j, suffix))
				}
				outData[i] = dp_workflow.NewBytesData(outItems)
			} else {
				outData[i] = dp_workflow.NewBytesData([][]byte{
					[]byte(fmt.Sprintf("%s[%d]%s", out, i, suffix)),
				})
			}
		}
	} else if numOut == 1 {
		if outNumItems[0] > 1 {
			outItems := make([][]byte, outNumItems[0])
			for j := 0; j < outNumItems[0]; j++ {
				outItems[j] = []byte(fmt.Sprintf("%s[][%d]%s", out, j, suffix))
			}
			outData[0] = dp_workflow.NewBytesData(outItems)
		} else {
			outData[0] = dp_workflow.NewBytesData([][]byte{
				[]byte(fmt.Sprintf("%s%s", out, suffix)),
			})
		}
	}
	return outData
}

func dummyScheduleFunc() ScheduleTaskFunc {
	return func(to *dp_workflow.TaskOrchestrator, sTask *dp_workflow.SchedulerTask, _ context.Context) error {
		inData := to.GetInData(sTask)
		taskName := strings.Split(sTask.GetTask().Name, "_")[2]
		outNumItems := make([]int, sTask.GetTask().NumOut)
		for i := 0; i < len(outNumItems); i++ {
			outNumItems[i] = 1
		}
		suffix := ""
		if sTask.GetDataParallelism() > 1 {
			if sTask.GetTask().InputSharding[0] == dp_workflow.ShardingKeyed {
				b := inData[0].GetBytes()[0]
				suffix = fmt.Sprintf(":%d", keyToIdx(rune(b[len(b)-2])))
			} else {
				suffix = fmt.Sprintf(":%d", sTask.SubtaskIdx)
			}

		}
		outData := constructOutput(taskName, inData, int(sTask.GetTask().NumOut), outNumItems, suffix)
		_ = to.SetOutData(sTask, outData)

		time.Sleep(20 * time.Millisecond) // forces concurrent jobs to run in parallel if supported by scheduler
		return nil
	}
}

func cpToDpSharding(cpSharding []cp_workflow.Sharding) []dp_workflow.Sharding {
	out := make([]dp_workflow.Sharding, len(cpSharding))
	for i, s := range cpSharding {
		switch s {
		case cp_workflow.ShardingAll:
			out[i] = dp_workflow.ShardingAll
		case cp_workflow.ShardingKeyed:
			out[i] = dp_workflow.ShardingKeyed
		case cp_workflow.ShardingEach:
			out[i] = dp_workflow.ShardingEach
		}
	}
	return out
}
func cpToDpWorkflow(cpWf *cp_workflow.Workflow, cpWfTasks []*cp_workflow.Task) *dp_workflow.Workflow {
	nameToDpTask := make(map[string]*dp_workflow.Task, len(cpWfTasks))

	dpWf := &dp_workflow.Workflow{
		Name:              cpWf.Name,
		Tasks:             make([]*dp_workflow.Task, len(cpWfTasks)),
		NumIn:             cpWf.NumIn,
		NumOut:            cpWf.NumOut,
		InitialTasks:      make([]*dp_workflow.Task, len(cpWf.InitialTasks)),
		InitialDataSrcIdx: cpWf.InitialDataSrcIdx,
		InitialDataDstIdx: cpWf.InitialDataDstIdx,
		OutTasks:          make([]*dp_workflow.Task, len(cpWf.OutTasks)),
		OutDataSrcIdx:     cpWf.OutDataSrcIdx,
	}

	for taskIdx, cpTask := range cpWfTasks {
		dpTask := &dp_workflow.Task{
			Name:               cpTask.Name,
			NumIn:              cpTask.NumIn,
			NumOut:             cpTask.NumOut,
			InputSharding:      cpToDpSharding(cpTask.InputSharding),
			Functions:          cpTask.Functions,
			FunctionInNum:      cpTask.FunctionInNum,
			FunctionOutNum:     cpTask.FunctionOutNum,
			FunctionDataFlow:   cpTask.FunctionDataFlow,
			FunctionInSharding: cpToDpSharding(cpTask.FunctionInSharding),
			ConsumerTasks:      make([]*dp_workflow.Task, len(cpTask.ConsumerTasks)),
			ConsumerDataSrcIdx: cpTask.ConsumerDataSrcIdx,
			ConsumerDataDstIdx: cpTask.ConsumerDataDstIdx,
		}
		dpWf.Tasks[taskIdx] = dpTask
		nameToDpTask[cpTask.Name] = dpTask
	}
	for _, cpTask := range cpWfTasks {
		dpTask := nameToDpTask[cpTask.Name]
		for i, consumerTask := range cpTask.ConsumerTasks {
			dpTask.ConsumerTasks[i] = nameToDpTask[consumerTask.Name]
		}
	}

	for i, cpTask := range cpWf.InitialTasks {
		dpWf.InitialTasks[i] = nameToDpTask[cpTask.Name]
	}
	for i, cpTask := range cpWf.OutTasks {
		dpWf.OutTasks[i] = nameToDpTask[cpTask.Name]
	}

	return dpWf
}

//-------
// Tests

func TestSimple(t *testing.T) {
	wfDescription := []string{
		`; simple chain
			(:function FunA (A) -> (B))
			(:function FunB (B) -> (C))
			(:function FunC (C) -> (D))
		
			(:composition Test (InputA) -> (OutputD) (
				(FunA ((A <- InputA)) => ((InterB := B)))
				(FunB ((B <- InterB)) => ((InterC := C)))
				(FunC ((C <- InterC)) => ((OutputD := D)))
			))
		`,
		`; simple split
			(:function FunA (A) -> (B))
			(:function FunB (B) -> (C))
			(:function FunC (B) -> (D))
			(:function FunD (C D) -> (E))
		
			(:composition y (InputA) -> (OutputE) (
				(FunA ((A <- InputA)) => ((InterB := B)))
				(FunB ((B <- InterB)) => ((InterC := C)))
				(FunC ((B <- InterB)) => ((InterD := D)))
				(FunD ((C <- InterC) (D <- InterD)) => ((OutputE := E)))
			))
		`,
		`; larger example
			(:function FunA (A) -> (B))
			(:function FunB (B) -> (C))
	
			(:function FunC (C) -> (D))
			(:function FunD (D) -> (E))
			(:function FunE (E) -> (F))
	
			(:function FunF (C) -> (G))
			(:function FunG (G) -> (H))
			(:function FunH (G) -> (I))
			(:function FunI (H I) -> (J))
	
			(:function FunJ (F J) -> (K))
			(:function FunK (K) -> (L))
			(:function FunL (L) -> (M))
		
			(:composition y (InputA) -> (OutputM) (
				(FunA ((A <- InputA)) => ((InterB := B)))
				(FunB ((B <- InterB)) => ((InterC := C)))
	
				(FunC ((C <- InterC)) => ((InterD := D)))
				(FunD ((D <- InterD)) => ((InterE := E)))
				(FunE ((E <- InterE)) => ((InterF := F)))
	
				(FunF ((C <- InterC)) => ((InterG := G)))
				(FunG ((G <- InterG)) => ((InterH := H)))
				(FunH ((G <- InterG)) => ((InterI := I)))
				(FunI ((H <- InterH) (I <- InterI)) => ((InterJ := J)))
	
				(FunJ ((F <- InterF) (J <- InterJ)) => ((InterK := K)))
				(FunK ((K <- InterK)) => ((InterL := L)))
				(FunL ((L <- InterL)) => ((OutputM := M)))
			))
		`,
		`; simple data handling
			(:function FunA (A) -> (C D))
			(:function FunB (C B) -> (E))
		
			(:composition y (InputA InputB) -> (OutputD OutputE) (
				(FunA ((A <- InputA)) => ((InterC := C) (OutputD := D)))
				(FunB ((C <- InterC) (B <- InputB)) => ((OutputE := E)))
			))
		`,
	}
	wfInput := [][]string{
		{"a"},
		{"a"},
		{"a"},
		{"a", "b"},
	}
	expectedOutput := [][]string{
		{"FunC2(FunB1(FunA0(a)))"},
		{"FunD3(FunB1(FunA0(a)),FunC2(FunA0(a)))"},
		{"FunL11(FunK10(FunJ9(FunE4(FunD3(FunC2(FunB1(FunA0(a))))),FunI8(FunG6(FunF5(FunB1(FunA0(a)))),FunH7(FunF5(FunB1(FunA0(a))))))))"},
		{
			"FunA0(a)[1]",
			"FunB1(FunA0(a)[0],b)",
		},
	}

	for testIdx, desc := range wfDescription {
		parser := dandelion_workflow.NewParser(bufio.NewReader(strings.NewReader(desc)))
		dwf, err := parser.Parse()
		if err != nil {
			t.Fatalf("Got error while parsing workflow description #%d: %v", testIdx, err)
		}
		dwf.Name = "Wf"
		wfs, wfTasks, err := dwf.ExportWorkflow(dandelion_workflow.FullPartition)
		if err != nil {
			t.Fatalf("Got error while exporting workflows (#%d): %v", testIdx, err)
		}
		wf := cpToDpWorkflow(wfs[0], wfTasks[0]) // only one workflow

		inData := make([]*dp_workflow.Data, len(wfInput[testIdx]))
		for i, input := range wfInput[testIdx] {
			inData[i] = dp_workflow.NewBytesData([][]byte{[]byte(input)})
		}
		for _, sType := range testedSchedulers {
			scheduler := NewScheduler(wf, sType)
			mockDPConfig := &config.DataPlaneConfig{WorkflowPreferredWorkerParallelism: 1}
			err = scheduler.Schedule(dummyScheduleFunc(), inData, mockDPConfig, context.Background())
			if err != nil {
				t.Fatalf("Got error while scheduling workflow (testIdx: %d, schedulerType: %d): %v", testIdx, sType, err)
			}

			outData, err := scheduler.CollectOutput()
			if err != nil {
				t.Fatalf("Got error while collecting output (testIdx: %d, schedulerType: %d): %v", testIdx, sType, err)
			}

			// check number of output objects
			if len(outData) != len(expectedOutput[testIdx]) {
				t.Errorf(
					"Got invalid number of output objects from workflow (testIdx: %d, schedulerType: %d):\n expected: %d\n got:      %d",
					testIdx, sType, len(expectedOutput[testIdx]), len(outData),
				)
				continue
			}

			// check output
			for i := 0; i < len(outData); i++ {
				outBytes := outData[i].GetBytes()
				if len(outBytes) > 1 {
					t.Errorf("Got multiple output items, expected only one")
					continue
				}
				outStr := string(outBytes[0])
				if outStr != expectedOutput[testIdx][i] {
					t.Errorf(
						"Got unexpected output (testIdx: %d, schedulerType: %d):\n expected: %s\n got:      %s",
						testIdx, sType, expectedOutput[testIdx][i], outStr,
					)
					continue
				}
			}
		}
	}
}

func TestDataParallelism(t *testing.T) {
	wfDescription := []string{
		`; simple fan-out with :each
			(:function FunA (A) -> (B))
		
			(:composition Test (In) -> (Out) (
				(FunA ((:each A <- In)) => ((Inter := B)))
				(FunA ((:all A <- Inter)) => ((Out := B)))
			))
		`,
		`; simple fan-out with :keyed
			(:function FunA (A) -> (B))
		
			(:composition Test (In) -> (Out) (
				(FunA ((:keyed A <- In)) => ((Inter := B)))
				(FunA ((:all A <- Inter)) => ((Out := B)))
			))
		`,
	}
	wfInput := [][][]string{
		{{"a0", "a1", "b2", "a3", "c4"}},
		{{"a0", "a1", "b2", "a3", "c4"}},
	}
	expectedOutput := [][]string{
		{"FunA1(FunA0(a0):0&FunA0(a1):1&FunA0(b2):2&FunA0(a3):3&FunA0(c4):4)"},
		{"FunA1(FunA0(a0&a1&a3):0&FunA0(b2):1&FunA0(c4):2)"},
	}

	for testIdx, desc := range wfDescription {
		parser := dandelion_workflow.NewParser(bufio.NewReader(strings.NewReader(desc)))
		dwf, err := parser.Parse()
		if err != nil {
			t.Fatalf("Got error while parsing workflow description #%d: %v", testIdx, err)
		}
		dwf.Name = "Wf"
		wfs, wfTasks, err := dwf.ExportWorkflow(dandelion_workflow.FullPartition)
		if err != nil {
			t.Fatalf("Got error while exporting workflows (#%d): %v", testIdx, err)
		}
		wf := cpToDpWorkflow(wfs[0], wfTasks[0]) // only one workflow

		inData := make([]*dp_workflow.Data, len(wfInput[testIdx]))
		for i, input := range wfInput[testIdx] {
			items := make([][]byte, len(input))
			for j, item := range input {
				items[j] = []byte(item)
			}
			inData[i] = dp_workflow.NewBytesData(items)
		}
		for _, sType := range testedSchedulers {
			scheduler := NewScheduler(wf, sType)
			mockDPConfig := &config.DataPlaneConfig{WorkflowPreferredWorkerParallelism: 1}
			err = scheduler.Schedule(dummyScheduleFunc(), inData, mockDPConfig, context.Background())
			if err != nil {
				t.Fatalf("Got error while scheduling workflow (testIdx: %d, schedulerType: %d): %v", testIdx, sType, err)
			}

			outData, err := scheduler.CollectOutput()
			if err != nil {
				t.Fatalf("Got error while collecting output (testIdx: %d, schedulerType: %d): %v", testIdx, sType, err)
			}

			// check number of output objects
			if len(outData) != len(expectedOutput[testIdx]) {
				t.Errorf(
					"Got invalid number of output objects from workflow (testIdx: %d, schedulerType: %d):\n expected: %d\n got:      %d",
					testIdx, sType, len(expectedOutput[testIdx]), len(outData),
				)
				continue
			}

			// check output
			for i := 0; i < len(outData); i++ {
				outBytes := outData[i].GetBytes()
				outStr := ""
				for j, outItem := range outBytes {
					outStr += string(outItem)
					if j != len(outBytes)-1 {
						outStr += "&"
					}
				}
				if outStr != expectedOutput[testIdx][i] {
					t.Errorf(
						"Got unexpected output (testIdx: %d, schedulerType: %d):\n expected: %s\n got:      %s",
						testIdx, sType, expectedOutput[testIdx][i], outStr,
					)
					continue
				}
			}
		}
	}
}

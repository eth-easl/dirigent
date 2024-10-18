package scheduler

import (
	"bufio"
	cp_workflow "cluster_manager/internal/control_plane/workflow"
	"cluster_manager/internal/control_plane/workflow/dandelion-workflow"
	dp_workflow "cluster_manager/internal/data_plane/workflow"
	"fmt"
	"strings"
	"testing"
	"time"
)

const testedSchedulerType = ConcurrentFifo

func dummyScheduleFunc() ScheduleTaskFunc {
	return func(to *dp_workflow.TaskOrchestrator, task *dp_workflow.Task) error {
		inData := to.GetInData(task)
		out := "("
		for _, data := range inData {
			out += string(data.GetBytes()) + ","
		}
		if len(inData) > 0 {
			out = out[:len(out)-1]
		}
		out += ")->" + task.Name
		outData := make([]*dp_workflow.Data, task.NumOut)
		if task.NumOut > 1 {
			for i := 0; i < int(task.NumOut); i++ {
				outData[i] = dp_workflow.NewBytesData([]byte(out + fmt.Sprintf("[%d]", i)))
			}
		} else if task.NumOut == 1 {
			outData[0] = dp_workflow.NewBytesData([]byte(out))
		}
		err := to.SetOutData(task, outData)
		if err != nil {
			return err
		}

		time.Sleep(100 * time.Millisecond)
		return nil
	}
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
			Functions:          cpTask.Functions,
			FunctionInNum:      cpTask.FunctionInNum,
			FunctionOutNum:     cpTask.FunctionOutNum,
			FunctionDataFlow:   cpTask.FunctionDataFlow,
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

func TestSimpleChain(t *testing.T) {
	testWorkflow := `
		(:function FunA (A) -> (B))
    	(:function FunB (B) -> (C))
    	(:function FunC (C) -> (D))
    
		(:composition Test (InputA) -> (OutputD) (
			(FunA (
				(A <- InputA)
			) => (
				(InterB := B)
			))
		
			(FunB (
				(B <- InterB)
			) => (
				(InterC := C)
			))
			
			(FunC (
				(C <- InterC)
			) => (
				(OutputD := D)
			))
		))
	`

	parser := dandelion_workflow.NewParser(bufio.NewReader(strings.NewReader(testWorkflow)))
	dwf, err := parser.Parse()
	if err != nil {
		t.Errorf("Got error while parsing input: %v", err)
		return
	}
	dwf.Name = "Wf"
	wfs, wfTasks, err := dwf.ExportWorkflow(dandelion_workflow.FullPartition)
	if err != nil {
		t.Errorf("Got error while exporting workflows: %v", err)
		return
	}
	wf := cpToDpWorkflow(wfs[0], wfTasks[0]) // only one workflow

	inputA := dp_workflow.NewBytesData([]byte("InputA"))
	inData := []*dp_workflow.Data{inputA}
	scheduler := NewScheduler(wf, testedSchedulerType)
	err = scheduler.Schedule(dummyScheduleFunc(), inData)
	if err != nil {
		t.Errorf("Got error while scheduling workflow: %v", err)
		return
	}

	outData, err := scheduler.CollectOutput()
	if err != nil {
		t.Errorf("Got error while collecting output: %v", err)
		return
	}
	if len(outData) != 1 || string(outData[0].GetBytes()) != "(((InputA)->Wf_Test_FunA0)->Wf_Test_FunB1)->Wf_Test_FunC2" {
		t.Errorf("Got invalid output: %s", string(outData[0].GetBytes()))
	}
}

func TestSimpleSplit(t *testing.T) {
	testWorkflow := `
		(:function FunA (A) -> (B))
    	(:function FunB (B) -> (C))
    	(:function FunC (B) -> (D))
    	(:function FunD (C D) -> (E))
    
		(:composition y (InputA) -> (OutputE) (
			(FunA (
				(A <- InputA)
			) => (
				(InterB := B)
			))
		
			(FunB (
				(B <- InterB)
			) => (
				(InterC := C)
			))

			(FunC (
				(B <- InterB)
			) => (
				(InterD := D)
			))
			
			(FunD (
				(C <- InterC)
				(D <- InterD)
			) => (
				(OutputE := E)
			))
		))
	`

	parser := dandelion_workflow.NewParser(bufio.NewReader(strings.NewReader(testWorkflow)))
	dwf, err := parser.Parse()
	if err != nil {
		t.Errorf("Got error while parsing input: %v", err)
		return
	}
	dwf.Name = "x"
	wfs, wfTasks, err := dwf.ExportWorkflow(dandelion_workflow.FullPartition)
	if err != nil {
		t.Errorf("Got error while exporting workflows: %v", err)
		return
	}
	wf := cpToDpWorkflow(wfs[0], wfTasks[0]) // only one workflow

	inputA := dp_workflow.NewBytesData([]byte("InputA"))
	inData := []*dp_workflow.Data{inputA}
	scheduler := NewScheduler(wf, testedSchedulerType)
	err = scheduler.Schedule(dummyScheduleFunc(), inData)
	if err != nil {
		t.Errorf("Got error while scheduling dp_workflow: %v", err)
		return
	}

	outData, err := scheduler.CollectOutput()
	if err != nil {
		t.Errorf("Got error while collecting output: %v", err)
		return
	}
	if len(outData) != 1 || string(outData[0].GetBytes()) != "(((InputA)->x_y_FunA0)->x_y_FunB1,((InputA)->x_y_FunA0)->x_y_FunC2)->x_y_FunD3" {
		t.Errorf("Got invalid output: %s", string(outData[0].GetBytes()))
	}
}

func TestSimpleExample(t *testing.T) {
	testWorkflow := `
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
			(FunI ((I <- InterI) (H <- InterH)) => ((InterJ := J)))

			(FunJ ((F <- InterF) (J <- InterJ)) => ((InterK := K)))
			(FunK ((K <- InterK)) => ((InterL := L)))
			(FunL ((L <- InterL)) => ((OutputM := M)))
		))
	`

	parser := dandelion_workflow.NewParser(bufio.NewReader(strings.NewReader(testWorkflow)))
	dwf, err := parser.Parse()
	if err != nil {
		t.Errorf("Got error while parsing input: %v", err)
		return
	}
	dwf.Name = "x"
	wfs, wfTasks, err := dwf.ExportWorkflow(dandelion_workflow.FullPartition)
	if err != nil {
		t.Errorf("Got error while exporting workflows: %v", err)
		return
	}
	wf := cpToDpWorkflow(wfs[0], wfTasks[0]) // only one workflow

	inputA := dp_workflow.NewBytesData([]byte("InputA"))
	inData := []*dp_workflow.Data{inputA}
	scheduler := NewScheduler(wf, testedSchedulerType)
	err = scheduler.Schedule(dummyScheduleFunc(), inData)
	if err != nil {
		t.Errorf("Got error while scheduling dp_workflow: %v", err)
		return
	}

	outData, err := scheduler.CollectOutput()
	if err != nil {
		t.Errorf("Got error while collecting output: %v", err)
		return
	}
	if len(outData) != 1 || string(outData[0].GetBytes()) != "((((((((InputA)->x_y_FunA0)->x_y_FunB1)->x_y_FunC2)->x_y_FunD3)->x_y_FunE4,(((((InputA)->x_y_FunA0)->x_y_FunB1)->x_y_FunF5)->x_y_FunH7,((((InputA)->x_y_FunA0)->x_y_FunB1)->x_y_FunF5)->x_y_FunG6)->x_y_FunI8)->x_y_FunJ9)->x_y_FunK10)->x_y_FunL11" {
		t.Errorf("Got invalid output: %s", string(outData[0].GetBytes()))
	}
}

func TestDandelionSimpleExamples(t *testing.T) {
	/* TODO: what are Dandelion library functions?
	testExample1 := `
		(:function MakePNGGrayscaleS3 (S3GetResponse) -> (S3PutRequest))

		(:composition MakePNGGrayscale (S3GetRequest) -> () (
			(DandelionHTTPGet ( (:keyed Request <- S3GetRequest) ) => ( (ToProcess := Response) ))
			(MakePNGGrayscale ( (:keyed S3GetResponse <- ToProcess) ) => ( (S3PutRequest := S3PutRequest) ))
			(DandelionHTTPPut ( (:keyed Request <- S3PutRequest) ) => ( ))
		))
	`
	*/
	testExample2 := `
		(:function FunA (A B) -> (C))
		(:function FunB (A B C) -> (D))
		(:function FunC (D) -> (E))
		
		(:composition y (InputA InputB) -> (OutputE) (
			(FunA (
				(:keyed A <- InputA)
				(:keyed B <- InputB)
			) => (
				(InterC := C)
			))
		
			(FunB (
				(:keyed A <- InputA)
				(:keyed B <- InputB)
				(:keyed C <- InterC)
			) => (
				(InterD := D)
			))
			
			(FunC (
				(:all D <- InterD)
			) => (
				(OutputE := E)
			))
		))
	`

	tests := []string{testExample2}
	inData := [][]*dp_workflow.Data{
		//{&dp_workflow.Data{Data: []byte("S3GetRequest")}},
		{dp_workflow.NewBytesData([]byte("InputA")), dp_workflow.NewBytesData([]byte("InputB"))},
	}
	expectedOutput := []string{
		//"",
		"((InputA,InputB,(InputA,InputB)->x_y_FunA0)->x_y_FunB1)->x_y_FunC2",
	}

	for i, test := range tests {
		parser := dandelion_workflow.NewParser(bufio.NewReader(strings.NewReader(test)))
		dwf, err := parser.Parse()
		if err != nil {
			t.Errorf("Got error while parsing input: %v (testcase %d)", err, i)
			return
		}
		dwf.Name = "x"
		wfs, wfTasks, err := dwf.ExportWorkflow(dandelion_workflow.FullPartition)
		if err != nil {
			t.Errorf("Got error while exporting workflows: %v", err)
			return
		}
		wf := cpToDpWorkflow(wfs[0], wfTasks[0]) // only one workflow

		scheduler := NewScheduler(wf, testedSchedulerType)
		err = scheduler.Schedule(dummyScheduleFunc(), inData[i])
		if err != nil {
			t.Errorf("Got error while scheduling dp_workflow: %v (testcase %d)", err, i)
			return
		}

		outData, err := scheduler.CollectOutput()
		if err != nil {
			t.Errorf("Got error while collecting output: %v (testcase %d)", err, i)
			return
		}
		if len(outData) != 1 || string(outData[0].GetBytes()) != expectedOutput[i] {
			t.Errorf("Got invalid output: %s (testcase %d)", string(outData[0].GetBytes()), i)
		}
	}
}

/* loops not supported yet
func TestDandelionLoopExample(t *testing.T) {
	testInput := `
		(:function CompileFiles (Source) -> (Out))
		(:function LinkObjects (ObjectFile Library) -> (Binary))

		(:composition CompileMulti (SourceFiles Libraries) -> (Binaries) (
			(CompileFiles (
				(:keyed Source <- SourceFile)
			) => (
				(ObjectFiles := Out)
			))

			(LinkObjects (
				(:all Objects <- ObjectFiles)
				(Libraries <- Libraries)
			) => (
				(Binaries := Binary)
			))
		))

		(:function CompileOneFile (SourcesBefore) -> (SourcesAfter Out))

		(:composition CompileFixpoint (SourceFiles Libraries) -> (Binary) (
			(:loop (
				(:until_empty Sources <- SourceFiles) ; until_empty (until the collection is empty), until_empty_item (until the collection's only item has length zero)
			) => (
				(CompileOneFile (
					(SourcesBefore <- Sources) ; no sharding modifier: run a single function instance with all the collection's inputs
				) => (
					(SourcesAfter := SourcesAfter)
					(Out := Out)
				))
			) => (
				(:feedback Sources := SourcesAfter) ; replaces Sources at each iteration
				(ObjectFiles := Out)
			))

			(LinkObjects (
				(Objects <- ObjectFiles)
				(Libraries <- Libraries) ; no sharding modifier: broadcast libraries orchestrator all funciton calls
			) => (
				(Binary := Binary)
			))
		)) ; test comment
	`

	parser := NewParser(bufio.NewReader(strings.NewReader(testInput)))
	wf, err := parser.Parse()
	if err != nil {
		t.Errorf("Got error while parsing input: %v", err)
		return
	}

	scheduler := NewScheduler(wf, testedSchedulerType)
	err = scheduler.Schedule(dummyScheduleFunc())
	if err != nil {
		t.Errorf("Got error while scheduling dp_workflow: %v", err)
		return
	}
}
*/

func TestDataHandling(t *testing.T) {
	testWorkflow := `
		(:function FunA (A) -> (C D))
    	(:function FunB (C B) -> (E))
    
		(:composition y (InputA InputB) -> (OutputD OutputE) (
			(FunA ((A <- InputA)) => ((InterC := C) (OutputD := D)))
			(FunB ((C <- InterC) (B <- InputB)) => ((OutputE := E)))
		))
	`

	parser := dandelion_workflow.NewParser(bufio.NewReader(strings.NewReader(testWorkflow)))
	dwf, err := parser.Parse()
	if err != nil {
		t.Errorf("Got error while parsing input: %v", err)
		return
	}
	dwf.Name = "x"
	wfs, wfTasks, err := dwf.ExportWorkflow(dandelion_workflow.FullPartition)
	if err != nil {
		t.Errorf("Got error while exporting workflows: %v", err)
		return
	}
	wf := cpToDpWorkflow(wfs[0], wfTasks[0]) // only one workflow

	inputA := dp_workflow.NewBytesData([]byte("InputA"))
	inputB := dp_workflow.NewBytesData([]byte("InputB"))
	inData := []*dp_workflow.Data{inputA, inputB}
	scheduler := NewScheduler(wf, testedSchedulerType)
	err = scheduler.Schedule(dummyScheduleFunc(), inData)
	if err != nil {
		t.Errorf("Got error while scheduling dp_workflow: %v", err)
		return
	}

	outData, err := scheduler.CollectOutput()
	if err != nil {
		t.Errorf("Got error while collecting output: %v", err)
		return
	}
	if len(outData) != 2 ||
		string(outData[0].GetBytes()) != "(InputA)->x_y_FunA0[1]" ||
		string(outData[1].GetBytes()) != "((InputA)->x_y_FunA0[0],InputB)->x_y_FunB1" {
		t.Errorf("Got invalid output: %s", string(outData[0].GetBytes()))
	}
}

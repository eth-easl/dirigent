package scheduler

import (
	"bufio"
	"go-sandbox/workflow-scheduling/workflow"
	"strings"
	"testing"
	"time"
)

const testedSchedulerType = ConcurrentFifo

func dummyScheduleFunc() ScheduleTaskFunc {
	return func(string) error {
		time.Sleep(100 * time.Millisecond)
		return nil
	}
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

	parser := workflow.NewParser(bufio.NewReader(strings.NewReader(testWorkflow)))
	wf, err := parser.Parse()
	if err != nil {
		t.Errorf("Got error while parsing input: %v", err)
		return
	}

	scheduler := NewScheduler(wf, testedSchedulerType)
	err = scheduler.Schedule(dummyScheduleFunc())
	if err != nil {
		t.Errorf("Got error while scheduling workflow: %v", err)
		return
	}
}

func TestSimpleSplit(t *testing.T) {
	testWorkflow := `
		(:function FunA (A) -> (B))
    	(:function FunB (B) -> (C))
    	(:function FunC (B) -> (D))
    	(:function FunD (C D) -> (E))
    
		(:composition Test (InputA) -> (OutputE) (
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

	parser := workflow.NewParser(bufio.NewReader(strings.NewReader(testWorkflow)))
	wf, err := parser.Parse()
	if err != nil {
		t.Errorf("Got error while parsing input: %v", err)
		return
	}

	scheduler := NewScheduler(wf, testedSchedulerType)
	err = scheduler.Schedule(dummyScheduleFunc())
	if err != nil {
		t.Errorf("Got error while scheduling workflow: %v", err)
		return
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
    
		(:composition Test (InputA) -> (OutputM) (
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

	parser := workflow.NewParser(bufio.NewReader(strings.NewReader(testWorkflow)))
	wf, err := parser.Parse()
	if err != nil {
		t.Errorf("Got error while parsing input: %v", err)
		return
	}

	scheduler := NewScheduler(wf, testedSchedulerType)
	err = scheduler.Schedule(dummyScheduleFunc())
	if err != nil {
		t.Errorf("Got error while scheduling workflow: %v", err)
		return
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
		
		(:composition Test (InputA InputB) -> (OutputE) (
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

	for i, test := range tests {
		parser := workflow.NewParser(bufio.NewReader(strings.NewReader(test)))
		wf, err := parser.Parse()
		if err != nil {
			t.Errorf("Got error while parsing input: %v (testcase %d)", err, i)
			return
		}

		scheduler := NewScheduler(wf, testedSchedulerType)
		err = scheduler.Schedule(dummyScheduleFunc())
		if err != nil {
			t.Errorf("Got error while scheduling workflow: %v (testcase %d)", err, i)
			return
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
				(Libraries <- Libraries) ; no sharding modifier: broadcast libraries to all funciton calls
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
		t.Errorf("Got error while scheduling workflow: %v", err)
		return
	}
}
*/

package workflow

import (
	"bufio"
	"fmt"
	"slices"
	"strings"
	"testing"
)

func TestDeclarationMismatch(t *testing.T) {
	wrongFunctionParam := `
		(:function FunA (A) -> (B))
    
		(:composition Test (InputA) -> (OutputC) (
			(FunA ((A <- InputA)) => ((OutputC := C))) ; <- should be B not C
		))
	`
	wrongParamNumber := `
		(:function FunA (A) -> (B))
    
		(:composition Test (InputA) -> (OutputB) (
			(FunA () => ((OutputB := B))) ; <- FunA should have an argument
		))
	`
	wrongRetNumber := `
		(:function FunA (A) -> (B))
    
		(:composition Test (InputA) -> (OutputB OutputC) (
			(FunA ((A <- InputA)) => ((OutputB := B) (OutputC := C))) ; <- Too many returns
		))
	`
	missingDeclaration := `
		(:composition Test (InputA) -> (OutputB) (
			(FunA ((A <- InputA)) => ((OutputB := B))) ; <- Missing declaration
		))
	`

	tests := []string{wrongFunctionParam, wrongParamNumber, wrongRetNumber, missingDeclaration}

	for i, test := range tests {
		parser := NewParser(bufio.NewReader(strings.NewReader(test)))
		wf, err := parser.Parse()
		if err != nil {
			t.Errorf("Got error while parsing input: %v (testcase %d)", err, i)
			return
		}

		err = wf.Process()
		if err == nil {
			t.Errorf("Expected error while processing invalid input (testcase %d).", i)
			return
		} else {
			fmt.Printf("Got expected error %v (testcase %d).\n", err, i)
		}
	}
}

func TestIOMismatch(t *testing.T) {
	missingCompositionInput := `
		(:function FunA (A) -> (B))
    
		(:composition Test () -> (OutputB) ( ; <- missing InputA
			(FunA ((A <- InputA)) => ((OutputB := B)))
		))
	`
	missingCompositionOutput := `
		(:function FunA (A) -> ())
    
		(:composition Test (InputA) -> (OutputB) ( ; <- no function output results in OutputB
			(FunA ((A <- InputA)) => ())
		))
	`
	missingIntermediate := `
		(:function FunA (A) -> ())
		(:function FunB (B) -> (C))
    
		(:composition Test (InputA) -> (OutputC) (
			(FunA ((A <- InputA)) => ())
			(FunB ((B <- InterB)) => ((OutputC := C))) ; <- InterB is not defined
		))
	`
	outputDuplicate := `
		(:function FunA (A) -> (C))
		(:function FunB (B) -> (C))
    
		(:composition Test (InputA InputB) -> (OutputC) (
			(FunA ((A <- InputA)) => ((OutputC := C)))
			(FunB ((B <- InterB)) => ((OutputC := C))) ; <- OutputC is defined twice
		))
	`

	tests := []string{missingCompositionInput, missingCompositionOutput, missingIntermediate, outputDuplicate}

	for i, test := range tests {
		parser := NewParser(bufio.NewReader(strings.NewReader(test)))
		wf, err := parser.Parse()
		if err != nil {
			t.Errorf("Got error while parsing input: %v (testcase %d)", err, i)
			return
		}

		err = wf.Process()
		if err == nil {
			t.Errorf("Expected error while processing invalid input (testcase %d).", i)
			return
		} else {
			fmt.Printf("Got expected error %v (testcase %d).\n", err, i)
		}
	}

}

func TestCircularDependencies(t *testing.T) {
	loopExample1 := `
		(:function FunA (A D) -> (B))
		(:function FunB (B) -> (C))
		(:function FunC (C) -> (D E))
    
		(:composition Test (InputA) -> (OutputE) (
			(FunA ((A <- InputA) (D <- InterD)) => ((InterB := B)))
			(FunB ((B <- InterB)) => ((InterC := C)))
			(FunC ((C <- InterC)) => ((InterD := D) (OutputE := E)))
		))
	`
	loopExample2 := `
		(:function FunA (A B) -> (C))
		(:function FunB (A C) -> (B))
		(:function FunC (B C) -> (D))
    
		(:composition Test (InputA) -> (OutputD) (
			(FunA ((A <- InputA) (B <- InterB)) => ((InterC := C)))
			(FunB ((B <- InterB) (C <- InterC)) => ((InterB := B)))
			(FunC ((B <- InterB) (C <- InterC)) => ((OutputD := D)))
		))
	`

	tests := []string{loopExample1, loopExample2}

	for i, test := range tests {
		parser := NewParser(bufio.NewReader(strings.NewReader(test)))
		wf, err := parser.Parse()
		if err != nil {
			t.Errorf("Got error while parsing input: %v (testcase %d)", err, i)
			return
		}

		err = wf.Process()
		if err == nil {
			t.Errorf("Expected error while processing invalid input (testcase %d).", i)
			return
		} else {
			fmt.Printf("Got expected error %v (testcase %d).\n", err, i)
		}
	}
}

type exportTestCase struct {
	exportName string
	headIdx    int
	tailIdx    int
}

func TestExport(t *testing.T) {
	input := `
		(:function FunA (A B) -> (C))
		(:function FunB (C) -> (D))
		(:function FunC (D E) -> (F))
    
		(:composition Test (InputA InputB InputE) -> (OutputF) (
			(FunA ((A <- InputA) (B <- InputB)) => ((InterC := C)))
			(FunB ((C <- InterC)) => ((InterD := D)))
			(FunC ((D <- InterD) (E <- InputE)) => ((OutputF := F)))
		))
	`

	parser := NewParser(bufio.NewReader(strings.NewReader(input)))
	wf, err := parser.Parse()
	if err != nil {
		t.Errorf("Got error while parsing input: %v", err)
		return
	}

	testCases := []exportTestCase{
		{"Test1", 0, 2},
		{"Test2", 2, 3},
		{"Test3", 1, 2},
	}
	expectedFuncDecl := [][]string{
		{"(:function FunA(A B)->(C))", "(:function FunB(C)->(D))"},
		{"(:function FunC(D E)->(F))"},
		{"(:function FunB(C)->(D))"},
	}
	expectedCompBody := []string{
		"((FunA((A<-InputA)(B<-InputB))=>((InterC:=C)))(FunB((C<-InterC))=>((InterD:=D)))",
		"((FunC((D<-InterD)(E<-InputE))=>((OutputF:=F))))",
		"((FunB((C<-InterC))=>((InterD:=D)))",
	}
	expectedCompParam := [][]string{
		{"InputA", "InputB"},
		{"InterD", "InputE"},
		{"InterC"},
	}
	expectedCompRets := [][]string{
		{"InterD"},
		{"OutputF"},
		{"InterD"},
	}

	for i, tc := range testCases {
		composition, err := wf.ExportStatements(tc.exportName, wf.Compositions[0].statements[tc.headIdx:tc.tailIdx])
		if err != nil {
			t.Errorf("Got error while exporting composition: %v", err)
			return
		}

		for _, funcDecl := range expectedFuncDecl[i] {
			if !strings.Contains(composition, funcDecl) {
				t.Errorf("Exported composition does not contain expected function declaration: %s\n -> output: %s.", funcDecl, composition)
			}
		}
		if !strings.Contains(composition, expectedCompBody[i]) {
			t.Errorf("Exported composition does not contain expected body: %s\n -> output: %s.", expectedCompBody[i], composition)
		}

		compParser := NewParser(bufio.NewReader(strings.NewReader(composition)))
		compWF, err := compParser.Parse()
		if err != nil {
			t.Errorf("Got error while reparsing exported composition: %v", err)
			return
		}

		if compWF.Compositions[0].Name != tc.exportName {
			t.Errorf("Exported composition has wrong Name: expected: %s, got: %s", compWF.Compositions[0].Name, tc.exportName)
		}
		for _, compParam := range expectedCompParam[i] {
			if !slices.Contains(compWF.Compositions[0].params, compParam) {
				t.Errorf("Exported composition does not contain the expected parameter: %s\n -> output: %s.", compParam, composition)
			}
		}
		if len(expectedCompParam[i]) != len(compWF.Compositions[0].params) {
			t.Errorf("Exported composition contains wrong number of parameters\n -> output: %s.", composition)
		}
		for _, compRet := range expectedCompRets[i] {
			if !slices.Contains(compWF.Compositions[0].returns, compRet) {
				t.Errorf("Exported composition does not contain the expected return: %s\n -> output: %s.", compRet, composition)
			}
		}
		if len(expectedCompRets[i]) != len(compWF.Compositions[0].returns) {
			t.Errorf("Exported composition contains wrong number of parameters\n -> output: %s.", composition)
		}

	}

}

func TestDataMapping(t *testing.T) {
	input := `
		(:function FunA (A B) -> (C D))
		(:function FunB (C D) -> (E F))
		(:function FunC (B D F G) -> (H))
    
		(:composition Test (InputA InputB InputG) -> (OutputE OutputH) (
			(FunA ((A <- InputA) (B <- InputB)) => ((InterC := C) (InterD := D)))
			(FunB ((C <- InterC) (D <- InterD)) => ((OutputE := E) (InterF := F)))
			(FunC ((B <- InputB) (D <- InterD) (F <- InterF) (G <- InputG)) => ((OutputH := H)))
		))
	`

	parser := NewParser(bufio.NewReader(strings.NewReader(input)))
	wf, err := parser.Parse()
	if err != nil {
		t.Errorf("Got error while parsing input: %v", err)
		return
	}
	err = wf.Process()
	if err != nil {
		t.Errorf("Got error while processing workflow: %v", err)
		return
	}

	inputA := &Data{[]byte("InputA")}
	inputB := &Data{[]byte("InputB")}
	interC := &Data{[]byte("InterC")}
	interD := &Data{[]byte("InterD")}
	outputE := &Data{[]byte("OutputE")}
	interF := &Data{[]byte("InterF")}
	inputG := &Data{[]byte("InputG")}
	outputH := &Data{[]byte("OutputH")}

	stmts, err := wf.Compositions[0].GetInitialRunnable([]*Data{inputA, inputB, inputG})
	if err != nil {
		t.Errorf("Got error while getting initial runnable: %v", err)
	}

	for len(stmts) > 0 {
		currStmt := stmts[0]

		var nextStmts []*Statement
		switch currStmt.Name {
		case "FunA":
			stmtInData := currStmt.GetInData()
			if stmtInData[0] != inputA || stmtInData[1] != inputB {
				t.Errorf("Got incorrect input data for FunA")
			}
			err = currStmt.SetOutData([]*Data{interC, interD})
			nextStmts = currStmt.SetDone()

		case "FunB":
			stmtInData := currStmt.GetInData()
			if stmtInData[0] != interC || stmtInData[1] != interD {
				t.Errorf("Got incorrect input data for FunB")
			}
			err = currStmt.SetOutData([]*Data{outputE, interF})
			nextStmts = currStmt.SetDone()

		case "FunC":
			stmtInData := currStmt.GetInData()
			if stmtInData[0] != inputB || stmtInData[1] != interD || stmtInData[2] != interF || stmtInData[3] != inputG {
				t.Errorf("Got incorrect input data for FunC")
			}
			err = currStmt.SetOutData([]*Data{outputH})
			nextStmts = currStmt.SetDone()
		}

		if err != nil {
			t.Errorf("Got error while getting next runnable statements: %v", err)
		}
		for _, nextStmt := range nextStmts {
			stmts = append(stmts, nextStmt)
		}

		stmts = stmts[1:]
	}

	outData := wf.Compositions[0].CollectOutData()
	if outData[0] != outputE || outData[1] != outputH {
		t.Errorf("Got incorrect output data for composition")
	}

}

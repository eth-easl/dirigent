package dandelion_workflow

import (
	"bufio"
	"fmt"
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

	c := wf.Compositions[0]

	// check composition input Consumers
	for consumerIdx, consumer := range c.Consumers {
		switch consumer.Name {
		case "FunA":
			if c.ConsumerArgIdx[consumerIdx] == 0 && c.ConsumerOutIdx[consumerIdx] == 0 {
				continue
			} else if c.ConsumerArgIdx[consumerIdx] == 1 && c.ConsumerOutIdx[consumerIdx] == 1 {
				continue
			}
			t.Errorf("Invalid data mapping for composition consumer FunA (idx: %d)", consumerIdx)
		case "FunC":
			if c.ConsumerArgIdx[consumerIdx] == 0 && c.ConsumerOutIdx[consumerIdx] == 1 {
				continue
			} else if c.ConsumerArgIdx[consumerIdx] == 3 && c.ConsumerOutIdx[consumerIdx] == 2 {
				continue
			}
			t.Errorf("Invalid data mapping for composition consumer FunC (idx: %d)", consumerIdx)
		default:
			t.Errorf("Invalid data mapping for composition to consumer %s", consumer.Name)
		}
	}

	// check Consumers of statements
	for _, stmt := range c.Statements {
		switch stmt.Name {
		case "FunA":
			for retIdx, ret := range stmt.Rets {
				for destIdx, dest := range ret.DestStmt {
					switch dest.Name {
					case "FunB":
						if retIdx == 0 && ret.DestStmtInIdx[destIdx] == 0 {
							continue
						} else if retIdx == 1 && ret.DestStmtInIdx[destIdx] == 1 {
							continue
						}
						t.Errorf("Invalid data mapping for FunA and destination FunB (retIdx: %d, destInIdx: %d)", retIdx, ret.DestStmtInIdx[destIdx])
					case "FunC":
						if retIdx == 1 && ret.DestStmtInIdx[destIdx] == 1 {
							continue
						}
						t.Errorf("Invalid data mapping for FunA and destination FunC (retIdx: %d, destInIdx: %d)", retIdx, ret.DestStmtInIdx[destIdx])
					default:
						t.Errorf("Invalid data mapping for FunA to destination %s", dest.Name)
					}
				}
			}

		case "FunB":
			for retIdx, ret := range stmt.Rets {
				for destIdx, dest := range ret.DestStmt {
					switch dest.Name {
					case "FunC":
						if retIdx == 1 && ret.DestStmtInIdx[destIdx] == 2 {
							continue
						}
						t.Errorf("Invalid data mapping for FunB and destination FunC (retIdx: %d, destInIdx: %d)", retIdx, ret.DestStmtInIdx[destIdx])
					default:
						t.Errorf("Invalid data mapping for FunB to destination %s", dest.Name)
					}
				}
			}

		case "FunC":
			for _, ret := range stmt.Rets {
				if len(ret.DestStmt) > 0 {
					t.Errorf("Found unexpected consumer for Func")
				}
			}

		default:
			t.Errorf("Unexpected statment %s", stmt.Name)
		}
	}

	// check output of composition
	for stmtIdx, stmt := range c.outStmts {
		switch stmt.Name {
		case "FunB":
			if stmtIdx == 0 && c.outStmtRetIdx[stmtIdx] == 0 {
				continue
			}
			t.Errorf("Invalid data mapping for composition output from FunB (idx: %d)", stmtIdx)
		case "FunC":
			if stmtIdx == 1 && c.outStmtRetIdx[stmtIdx] == 0 {
				continue
			}
			t.Errorf("Invalid data mapping for composition output from FunC (idx: %d)", stmtIdx)
		default:
			t.Errorf("Invalid data mapping for composition output from %s", stmt.Name)
		}
	}

}

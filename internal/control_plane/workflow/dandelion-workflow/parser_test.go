package dandelion_workflow

import (
	"bufio"
	"strings"
	"testing"
)

func TestFunctionDeclaration(t *testing.T) {
	testInput := `
		function FunctionName (Param1, Param2) => (Ret1, Ret2);
	`

	parser := NewParser(bufio.NewReader(strings.NewReader(testInput)))
	output, err := parser.Parse()

	if err != nil {
		t.Errorf("Got error while parsing input: %v", err)
		return
	}

	if output == nil {
		t.Errorf("Got nil output")
		return
	}
	if output.Compositions != nil {
		t.Errorf("Got unexpected Compositions")
	}
	if output.FunctionDecls == nil {
		t.Errorf("Got nil FunctionDecls")
		return
	}
	if output.FunctionDecls[0].name != "FunctionName" {
		t.Errorf("Wrong function Name")
	}
	if output.FunctionDecls[0].params[0] != "Param1" {
		t.Errorf("Wrong param Name")
	}
	if output.FunctionDecls[0].params[1] != "Param2" {
		t.Errorf("Wrong param Name")
	}
	if output.FunctionDecls[0].returns[0] != "Ret1" {
		t.Errorf("Wrong return Name")
	}
	if output.FunctionDecls[0].returns[1] != "Ret2" {
		t.Errorf("Wrong return Name")
	}

}

func TestFunctionApplication(t *testing.T) {
	testInput := `
		function F (A, B) => (C);
		composition C (InA, InB) => (OutA) {
			F (A = all InA, B = all InB) => (OutA = C);
		}
	`

	parser := NewParser(bufio.NewReader(strings.NewReader(testInput)))
	output, err := parser.Parse()

	if err != nil {
		t.Errorf("Got error while parsing input: %v", err)
		return
	}

	if output == nil {
		t.Errorf("Got nil output")
		return
	}
	if len(output.FunctionDecls) != 1 {
		t.Errorf("Got != 1 function declarations")
		return
	}
	if len(output.Compositions) != 1 {
		t.Errorf("Got != 1 Compositions")
		return
	}

}

func TestComment(t *testing.T) {
	testInput := `
		function F (A) => (B, C); # some comment that should be ignored
		function G (A, B, C) => (D); # some other comment at end of file
	`

	parser := NewParser(bufio.NewReader(strings.NewReader(testInput)))
	output, err := parser.Parse()

	if err != nil {
		t.Errorf("Got error while parsing input: %v", err)
		return
	}

	if output == nil {
		t.Errorf("Got nil output")
		return
	}
	if len(output.FunctionDecls) != 2 {
		t.Errorf("Got != 2 function declarations")
		return
	}
	if len(output.Compositions) != 0 {
		t.Errorf("Got != 0 Compositions")
		return
	}
}

func TestDandelionExample1(t *testing.T) {
	testInput := `
		function CompileFiles (Source) => (Out);
		function LinkObjects (ObjectFile, Library) => (Binary);
		
		composition CompileMulti (SourceFiles, Libraries) => (Binaries) {
			CompileFiles (Source = keyed SourceFile) => (ObjectFiles = Out);
			LinkObjects (Objects = all ObjectFiles, Libraries = all Libraries) => (Binaries = Binary);
		}
		# end of file comment
	`

	parser := NewParser(bufio.NewReader(strings.NewReader(testInput)))
	_, err := parser.Parse()

	if err != nil {
		t.Errorf("Got error while parsing input: %v", err)
		return
	}
}

func TestDandelionExample2(t *testing.T) {
	testInput := `
		function MakePNGGrayscaleS3 (S3GetResponse) => (S3PutRequest);
		
		composition MakePNGGrayscale (S3GetRequest) => () {
			DandelionHTTPGet (Request = keyed S3GetRequest) => (ToProcess = Response);
			MakePNGGrayscale (S3GetResponse = keyed ToProcess) => (S3PutRequest = S3PutRequest);
			DandelionHTTPPut (Request = keyed S3PutRequest) => ();
		}
	`

	parser := NewParser(bufio.NewReader(strings.NewReader(testInput)))
	_, err := parser.Parse()

	if err != nil {
		t.Errorf("Got error while parsing input: %v", err)
		return
	}
}

func TestDandelionExample3(t *testing.T) {
	testInput := `
		function FunA (A, B) => (C);
		function FunB (A, B, C) => (D);
		function FunC (D) => (E);
		
		composition Test (InputA, InputB) => (OutputE) {
			FunA (A = keyed InputA, B = keyed InputB) => (InterC = C);
			FunB (A = keyed InputA, B = keyed InputB, C = keyed InterC) => (InterD = D);
			FunC (D = all InterD) => (OutputE = E);			
		}
	`

	parser := NewParser(bufio.NewReader(strings.NewReader(testInput)))
	_, err := parser.Parse()

	if err != nil {
		t.Errorf("Got error while parsing input: %v", err)
		return
	}
}

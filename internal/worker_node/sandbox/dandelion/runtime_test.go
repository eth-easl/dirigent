package dandelion

import (
	"cluster_manager/proto"
	"testing"
)

func TestExportFunctionComposition(t *testing.T) {
	testTask := proto.WorkflowTaskInfo{
		Name:           "Test",
		NumIn:          2,
		NumOut:         3,
		Functions:      []string{"a", "b", "c", "d", "e", "f", "g"},
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
		FunctionInSharding: make([]uint32, 11),
	}

	export := exportFunctionComposition(&testTask)

	expected := "function a (in0) => (out0, out1);" +
		"function b (in0) => (out0);" +
		"function c (in0, in1) => (out0, out1);" +
		"function d (in0, in1) => (out0);" +
		"function e (in0, in1, in2) => (out0);" +
		"function f () => (out0, out1, out2, out3);" +
		"function g (in0, in1) => (); " +
		"composition Test (cIn0, cIn1) => (f2d0, f4d0, f5d1) {" +
		"a (in0 = all cIn0) => (f0d0 = out0, f0d1 = out1); " +
		"b (in0 = all f0d0) => (f1d0 = out0); " +
		"c (in0 = all f1d0, in1 = all f3d0) => (f2d0 = out0, f2d1 = out1); " +
		"d (in0 = all f0d1, in1 = all cIn1) => (f3d0 = out0); " +
		"e (in0 = all f2d1, in1 = all f3d0, in2 = all f5d0) => (f4d0 = out0); " +
		"f () => (f5d0 = out0, f5d1 = out1, f5d2 = out2, f5d3 = out3); " +
		"g (in0 = all f5d3, in1 = all f5d2) => ();" +
		"}"

	if export != expected {
		t.Errorf("Export does not match the expectation: \nexport:   %s\nexpected: %s", export, expected)
	}
}

func TestExportFunctionCompositionMultiUse(t *testing.T) {
	testTask := proto.WorkflowTaskInfo{
		Name:           "Test",
		NumIn:          2,
		NumOut:         0,
		Functions:      []string{"a", "b", "b", "c"},
		FunctionInNum:  []int32{2, 1, 1, 1},
		FunctionOutNum: []int32{1, 1, 1, 0},
		FunctionDataFlow: []int32{ // pair {src function, src data idx} for every input
			-1, 0, -1, 1, // a
			0, 0, // b
			1, 0, // b
			2, 0, // c
		},
		FunctionInSharding: make([]uint32, 5),
	}

	export := exportFunctionComposition(&testTask)

	expected := "function a (in0, in1) => (out0);" +
		"function b (in0) => (out0);" +
		"function c (in0) => (); " +
		"composition Test (cIn0, cIn1) => () {" +
		"a (in0 = all cIn0, in1 = all cIn1) => (f0d0 = out0); " +
		"b (in0 = all f0d0) => (f1d0 = out0); " +
		"b (in0 = all f1d0) => (f2d0 = out0); " +
		"c (in0 = all f2d0) => ();" +
		"}"

	if export != expected {
		t.Errorf("Export does not match the expectation: \nexport:   %s\nexpected: %s", export, expected)
	}
}

func TestExportFunctionCompositionSharding(t *testing.T) {
	testTask := proto.WorkflowTaskInfo{
		Name:           "Test",
		NumIn:          2,
		NumOut:         1,
		Functions:      []string{"a", "a", "b", "c"},
		FunctionInNum:  []int32{2, 2, 1, 1},
		FunctionOutNum: []int32{1, 1, 1, 1},
		FunctionDataFlow: []int32{ // pair {src function, src data idx} for every input
			-1, 0, -1, 1, // a
			0, 0, -1, 1, // a
			1, 0, // b
			2, 0, // c
			3, 0, // composition out
		},
		FunctionInSharding: []uint32{ // value for every input
			0, 0, // a
			1, 0, // a
			2, // b
			0, // c
		},
	}

	export := exportFunctionComposition(&testTask)

	expected := "function a (in0, in1) => (out0);" +
		"function b (in0) => (out0);" +
		"function c (in0) => (out0); " +
		"composition Test (cIn0, cIn1) => (f3d0) {" +
		"a (in0 = all cIn0, in1 = all cIn1) => (f0d0 = out0); " +
		"a (in0 = keyed f0d0, in1 = all cIn1) => (f1d0 = out0); " +
		"b (in0 = each f1d0) => (f2d0 = out0); " +
		"c (in0 = all f2d0) => (f3d0 = out0);" +
		"}"

	if export != expected {
		t.Errorf("Export does not match the expectation: \nexport:   %s\nexpected: %s", export, expected)
	}
}

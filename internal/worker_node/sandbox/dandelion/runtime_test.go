package dandelion

import (
	"cluster_manager/pkg/config"
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

	dummyRuntime := Runtime{dandelionConfig: &config.DandelionConfig{AddOperatorLoadAndStore: false}}
	export := dummyRuntime.exportFunctionComposition(&testTask)

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

	dummyRuntime := Runtime{dandelionConfig: &config.DandelionConfig{AddOperatorLoadAndStore: false}}
	export := dummyRuntime.exportFunctionComposition(&testTask)

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

	dummyRuntime := Runtime{dandelionConfig: &config.DandelionConfig{AddOperatorLoadAndStore: false}}
	export := dummyRuntime.exportFunctionComposition(&testTask)

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

func TestExportFunctionCompositionLoadStore1(t *testing.T) {
	testTask := proto.WorkflowTaskInfo{
		Name:           "Test",
		NumIn:          3,
		NumOut:         2,
		Functions:      []string{"csv_reader", "filter"},
		FunctionInNum:  []int32{2, 3},
		FunctionOutNum: []int32{2, 2},
		FunctionDataFlow: []int32{ // pair {src function, src data idx} for every input
			-1, 0, -1, 2, // csv_reader
			-1, 1, 0, 0, 0, 1, // filter
			1, 0, 1, 1, // composition out
		},
		FunctionInSharding: []uint32{ // value for every input
			0, 2, // csv_reader
			0, 0, 2, // filter
		},
	}

	dummyRuntime := Runtime{dandelionConfig: &config.DandelionConfig{AddOperatorLoadAndStore: false}}
	exportNoLoadAndStore := dummyRuntime.exportFunctionComposition(&testTask)
	expectedNoLoadAndStore := "function csv_reader (in0, in1) => (out0, out1);" +
		"function filter (in0, in1, in2) => (out0, out1); " +
		"composition Test (cIn0, cIn1, cIn2) => (f1d0, f1d1) {" +
		"csv_reader (in0 = all cIn0, in1 = each cIn2) => (f0d0 = out0, f0d1 = out1); " +
		"filter (in0 = all cIn1, in1 = all f0d0, in2 = each f0d1) => (f1d0 = out0, f1d1 = out1);" +
		"}"
	if exportNoLoadAndStore != expectedNoLoadAndStore {
		t.Errorf("Export does not match the expectation: \nexport:   %s\nexpected: %s", exportNoLoadAndStore, expectedNoLoadAndStore)
	}

	dummyRuntime.dandelionConfig.AddOperatorLoadAndStore = true
	exportWithLoadAndStore := dummyRuntime.exportFunctionComposition(&testTask)
	expectedWithLoadAndStore := "function fetch_requests(url) => (getRequest);" +
		"function HTTP(request) => (response);" +
		"function csv_reader_op (in0, in1) => (out0, out1);" +
		"function filter_op (in0, in1, in2) => (out0, out1);" +
		"function store_requests(inUrl, data) => (putRequest, outUrl); " +
		"composition Test (cIn0, cIn1, cIn2) => (outSchemaUrl, outBatchesUrl) {" +
		"fetch_requests(url = each cIn2) => (inDataReq = getRequest); " +
		"HTTP(request = each inDataReq) => (fetched2 = response); " +
		"csv_reader_op (in0 = all cIn0, in1 = each fetched2) => (f0d0 = out0, f0d1 = out1); " +
		"filter_op (in0 = all cIn1, in1 = all f0d0, in2 = each f0d1) => (f1d0 = out0, f1d1 = out1); " +
		"store_requests(inUrl = all cIn2, data = each f1d0) => (outSchemaReq = putRequest, outSchemaUrl = outUrl); " +
		"HTTP(request = each outSchemaReq) => (_0 = response); " +
		"store_requests(inUrl = all cIn2, data = each f1d1) => (outBatchesReq = putRequest, outBatchesUrl = outUrl); " +
		"HTTP(request = each outBatchesReq) => (_1 = response);" +
		"}"
	if exportWithLoadAndStore != expectedWithLoadAndStore {
		t.Errorf("Export does not match the expectation: \nexport:   %s\nexpected: %s", exportWithLoadAndStore, expectedWithLoadAndStore)
	}

}

func TestExportFunctionCompositionLoadStore2(t *testing.T) {
	testTask := proto.WorkflowTaskInfo{
		Name:           "Test",
		NumIn:          7,
		NumOut:         1,
		Functions:      []string{"hash_join", "project", "csv_writer"},
		FunctionInNum:  []int32{5, 3, 3},
		FunctionOutNum: []int32{2, 2, 1},
		FunctionDataFlow: []int32{ // pair {src function, src data idx} for every input
			-1, 0, -1, 3, -1, 4, -1, 5, -1, 6, // hash_join
			-1, 1, 0, 0, 0, 1, // project
			-1, 2, 1, 0, 1, 1, // csv_writer
			2, 0, // composition out
		},
		FunctionInSharding: []uint32{ // value for every input
			0, 0, 0, 0, 0, // hash_join
			0, 0, 2, // project
			0, 0, 2, // csv_writer
		},
	}

	dummyRuntime := Runtime{dandelionConfig: &config.DandelionConfig{AddOperatorLoadAndStore: false}}
	exportNoLoadAndStore := dummyRuntime.exportFunctionComposition(&testTask)
	expectedNoLoadAndStore := "function hash_join (in0, in1, in2, in3, in4) => (out0, out1);" +
		"function project (in0, in1, in2) => (out0, out1);" +
		"function csv_writer (in0, in1, in2) => (out0); " +
		"composition Test (cIn0, cIn1, cIn2, cIn3, cIn4, cIn5, cIn6) => (f2d0) {" +
		"hash_join (in0 = all cIn0, in1 = all cIn3, in2 = all cIn4, in3 = all cIn5, in4 = all cIn6) => (f0d0 = out0, f0d1 = out1); " +
		"project (in0 = all cIn1, in1 = all f0d0, in2 = each f0d1) => (f1d0 = out0, f1d1 = out1); " +
		"csv_writer (in0 = all cIn2, in1 = all f1d0, in2 = each f1d1) => (f2d0 = out0);" +
		"}"
	if exportNoLoadAndStore != expectedNoLoadAndStore {
		t.Errorf("Export does not match the expectation: \nexport:   %s\nexpected: %s", exportNoLoadAndStore, expectedNoLoadAndStore)
	}

	dummyRuntime.dandelionConfig.AddOperatorLoadAndStore = true
	exportWithLoadAndStore := dummyRuntime.exportFunctionComposition(&testTask)
	expectedWithLoadAndStore := "function fetch_requests(url) => (getRequest);" +
		"function HTTP(request) => (response);" +
		"function hash_join_op (in0, in1, in2, in3, in4) => (out0, out1);" +
		"function project_op (in0, in1, in2) => (out0, out1);" +
		"function csv_writer_op (in0, in1, in2) => (out0);" +
		"function store_requests(inUrl, data) => (putRequest, outUrl); " +
		"composition Test (cIn0, cIn1, cIn2, cIn3, cIn4, cIn5, cIn6) => (outDataUrl) {" +
		"fetch_requests(url = each cIn3) => (inReq3 = getRequest); " +
		"HTTP(request = each inReq3) => (fetched3 = response); " +
		"fetch_requests(url = each cIn4) => (inReq4 = getRequest); " +
		"HTTP(request = each inReq4) => (fetched4 = response); " +
		"fetch_requests(url = each cIn5) => (inReq5 = getRequest); " +
		"HTTP(request = each inReq5) => (fetched5 = response); " +
		"fetch_requests(url = each cIn6) => (inReq6 = getRequest); " +
		"HTTP(request = each inReq6) => (fetched6 = response); " +
		"hash_join_op (in0 = all cIn0, in1 = all fetched3, in2 = all fetched4, in3 = all fetched5, in4 = all fetched6) => (f0d0 = out0, f0d1 = out1); " +
		"project_op (in0 = all cIn1, in1 = all f0d0, in2 = each f0d1) => (f1d0 = out0, f1d1 = out1); " +
		"csv_writer_op (in0 = all cIn2, in1 = all f1d0, in2 = each f1d1) => (f2d0 = out0); " +
		"store_requests(inUrl = all cIn4, data = each f2d0) => (outDataReq = putRequest, outDataUrl = outUrl); " +
		"HTTP(request = each outDataReq) => (_0 = response);" +
		"}"
	if exportWithLoadAndStore != expectedWithLoadAndStore {
		t.Errorf("Export does not match the expectation: \nexport:   %s\nexpected: %s", exportWithLoadAndStore, expectedWithLoadAndStore)
	}

}

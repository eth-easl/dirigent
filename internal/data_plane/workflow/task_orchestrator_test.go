package workflow

import (
	"fmt"
	"slices"
	"testing"
)

func compareDataIndexes(dataIdxs [][]int, expected [][]int) error {
	if len(dataIdxs) != len(expected) {
		return fmt.Errorf("expected %d indexes, got %d", len(expected), len(dataIdxs))
	}
	for i, idx := range dataIdxs {
		if len(idx) != len(expected[i]) {
			return fmt.Errorf("expected %d item indexes for data index %d, got %d", len(expected[i]), i, len(idx))
		}
		for _, itmIdx := range idx {
			if !slices.Contains(expected[i], itmIdx) {
				return fmt.Errorf("item index %d for data index %d is not in expected set %v", itmIdx, i, expected[i])
			}
		}
	}
	return nil
}

func TestGetSchedulerTasks(t *testing.T) {
	testTasks := []Task{
		{
			Name:          "Parallel",
			NumIn:         4,
			InputSharding: []Sharding{ShardingEach, ShardingAll, ShardingKeyed, ShardingAll},
		},
		{
			Name:          "NonParallel",
			NumIn:         4,
			InputSharding: []Sharding{ShardingAll, ShardingAll, ShardingAll, ShardingAll},
		},
	}
	inputStr := [][]string{{"a0", "a2"}, {"b0", "b1"}, {"c0", "d1", "c2", "e3"}, {"f0", "f1"}}
	input := make([]*Data, len(inputStr))
	for i, setStr := range inputStr {
		bytes := make([][]byte, len(setStr))
		for j, str := range setStr {
			bytes[j] = []byte(str)
		}
		input[i] = NewBytesData(bytes)
	}

	workerSubtasksList := []int{1, 2, 3, 4}
	expectedDataIdxs := [][][][][]int{
		{ // task = Parallel
			{ // workerSubtasks = 1
				{{0}, {}, {0, 2}, {}},
				{{0}, {}, {1}, {}},
				{{0}, {}, {3}, {}},
				{{1}, {}, {0, 2}, {}},
				{{1}, {}, {1}, {}},
				{{1}, {}, {3}, {}},
			},
			{ // workerSubtasks = 2
				{{0, 1}, {}, {0, 2}, {}},
				{{0, 1}, {}, {1}, {}},
				{{0, 1}, {}, {3}, {}},
			},
			{ // workerSubtasks = 3
				{{0}, {}, {0, 1, 2, 3}, {}},
				{{1}, {}, {0, 1, 2, 3}, {}},
			},
			{ // workerSubtasks = 4
				{{0}, {}, {0, 1, 2, 3}, {}},
				{{1}, {}, {0, 1, 2, 3}, {}},
			},
		},
		{ // task = NonParallel
			{ // workerSubtasks = 1
				{{}, {}, {}, {}},
			},
			{ // workerSubtasks = 2
				{{}, {}, {}, {}},
			},
			{ // workerSubtasks = 3
				{{}, {}, {}, {}},
			},
			{ // workerSubtasks = 4
				{{}, {}, {}, {}},
			},
		},
	}

	for taskIdx, task := range testTasks {
		if taskIdx == 0 {
			continue
		}
		for wsIdx, workerSubtasks := range workerSubtasksList {
			schedulerTasks, err := getSchedulerTasks(&task, input, workerSubtasks)
			if err != nil {
				t.Fatalf("getSchedulerTasks failed: %v", err)
			}

			if len(schedulerTasks) != len(expectedDataIdxs[taskIdx][wsIdx]) {
				t.Errorf(
					"expected %d scheduler tasks, got %d (task #%d, #workerSubtasks: %d)",
					len(expectedDataIdxs[taskIdx][wsIdx]), len(schedulerTasks), taskIdx, workerSubtasks,
				)
				continue
			}
			checked := make([]bool, len(schedulerTasks))
			for sTaskIdx, sTask := range schedulerTasks {
				var cmpErr error
				for i, expectedIdx := range expectedDataIdxs[taskIdx][wsIdx] {
					if !checked[i] {
						cmpErr = compareDataIndexes(sTask.dataIdxs, expectedIdx)
						if cmpErr == nil {
							checked[i] = true
							break
						}
					}
				}
				if cmpErr != nil {
					t.Errorf(
						"no expected dataIdx set fit schedulerTask #%d (task #%d, #workerSubtasks: %d)",
						sTaskIdx, taskIdx, workerSubtasks,
					)
					continue
				}
			}
		}
	}
}

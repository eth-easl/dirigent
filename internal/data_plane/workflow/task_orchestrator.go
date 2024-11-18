package workflow

import (
	"cluster_manager/pkg/config"
	"fmt"
	"github.com/sirupsen/logrus"
	"sync"
	"sync/atomic"
)

type SchedulerTask struct {
	taskPtr         *Task
	subtasksLeft    *atomic.Int32
	dataIdxs        [][]int
	dataParallelism int

	SubtaskIdx int
}

type TaskData struct {
	inData       []*Data
	inDataMutex  sync.RWMutex
	outData      []*Data
	outDataMutex sync.RWMutex
}

type TaskOrchestrator struct {
	wf       *Workflow
	taskData map[*Task]*TaskData
	dataType DataType
	pWorker  int // preferred worker parallelism
}

func (s *SchedulerTask) GetTask() *Task {
	return s.taskPtr
}
func (s *SchedulerTask) GetDataParallelism() int {
	return s.dataParallelism
}

// getSchedulerTasks collects data parallelism information for each input data object and creates a scheduler task
// for all parallel data object item combinations (cartesian product) while also best-effort combining some items
// to allow for parallelism on the worker runtime
func getSchedulerTasks(t *Task, inData []*Data, pWorker int) ([]*SchedulerTask, error) {
	// collect data item index mappings
	dataIdxMap := make([][][]int, t.NumIn)
	numCombinations := 1
	for i := int(t.NumIn) - 1; i >= 0; i-- {
		dataIdxMap[i] = inData[i].GetDataParallelism(t.InputSharding[i])
		if dataIdxMap[i] == nil {
			return nil, fmt.Errorf("failed to get data parallelism for input #%d of task %s", i, t.Name)
		}
		numCombinations *= max(len(dataIdxMap[i]), 1)
	}

	// find best index over which we combine different data item indexes (-> allows for parallelization on worker)
	combIdx := -1
	combSetP := 0
	for i := 0; i < int(t.NumIn); i++ {
		currSetP := len(dataIdxMap[i])
		if currSetP == 0 {
			continue
		}
		if currSetP%pWorker == 0 {
			combIdx = i
			combSetP = currSetP
			break
		}
		if currSetP > combSetP {
			combIdx = i
			combSetP = currSetP
		}
	}

	// get number of splits over the combined index and total number of scheduler tasks
	combSetNumSplits := 1
	numSchedulerTasks := 1
	if combIdx != -1 {
		combSetNumSplits = combSetP / pWorker
		if combSetP%pWorker != 0 {
			combSetNumSplits++
		}
		numSchedulerTasks = numCombinations / combSetP * combSetNumSplits
	}

	subtaskCounter := atomic.Int32{}
	subtaskCounter.Store(int32(numSchedulerTasks))
	outTasks := make([]*SchedulerTask, numSchedulerTasks)
	for setIdx := 0; setIdx < numSchedulerTasks; setIdx++ {
		outTasks[setIdx] = &SchedulerTask{
			taskPtr:         t,
			subtasksLeft:    &subtaskCounter,
			dataIdxs:        make([][]int, t.NumIn),
			dataParallelism: numSchedulerTasks,
			SubtaskIdx:      setIdx,
		}
	}
	repeat := 1
	for setIdx := 0; setIdx < int(t.NumIn); setIdx++ {
		if len(dataIdxMap[setIdx]) == 0 { // no parallelization of this set
			continue
		}

		// set over which some items are combined
		if setIdx == combIdx {
			combSetIdxMap := make([][]int, combSetNumSplits)
			idx := 0
			for i := 0; i < combSetNumSplits; i++ {
				for j := 0; j < pWorker; j++ {
					if idx == combSetP {
						break
					}
					combSetIdxMap[i] = append(combSetIdxMap[i], dataIdxMap[setIdx][idx]...)
					idx++
				}
			}
			sTaskIdx := 0
			for sTaskIdx < numSchedulerTasks {
				for cIdx := 0; cIdx < len(combSetIdxMap); cIdx++ {
					for i := 0; i < repeat; i++ {
						outTasks[sTaskIdx].dataIdxs[setIdx] = combSetIdxMap[cIdx]
						sTaskIdx++
					}
				}
			}
			repeat *= combSetNumSplits
			continue
		}

		// default case
		sTaskIdx := 0
		for sTaskIdx < numSchedulerTasks {
			for setMapIdx := 0; setMapIdx < len(dataIdxMap[setIdx]); setMapIdx++ {
				for i := 0; i < repeat; i++ {
					outTasks[sTaskIdx].dataIdxs[setIdx] = dataIdxMap[setIdx][setMapIdx]
					sTaskIdx++
				}
			}
		}
		repeat *= len(dataIdxMap[setIdx])
	}
	return outTasks, nil
}

func GetInitialRunnable(wf *Workflow, inData []*Data, dpConfig *config.DataPlaneConfig) (*TaskOrchestrator, []*SchedulerTask, error) {
	if len(inData) != int(wf.NumIn) {
		return nil, nil, fmt.Errorf("got %d input data objects, composition has %d params", len(inData), wf.NumIn)
	}

	// initialize task orchestrator
	taskData := make(map[*Task]*TaskData)
	for _, t := range wf.Tasks {
		taskData[t] = &TaskData{
			inData: make([]*Data, t.NumIn),
		}
	}
	dataType := Unknown
	if len(inData) > 0 {
		dataType = inData[0].dType
	}
	to := &TaskOrchestrator{
		wf:       wf,
		taskData: taskData,
		dataType: dataType,
		pWorker:  dpConfig.WorkflowPreferredWorkerParallelism,
	}

	// get initial runnable tasks
	var tasksRunnable []*SchedulerTask
	for i, initTask := range wf.InitialTasks {
		to.taskData[initTask].inData[wf.InitialDataDstIdx[i]] = inData[wf.InitialDataSrcIdx[i]]

		allTrue := true
		for _, arg := range to.taskData[initTask].inData {
			if arg == nil {
				allTrue = false
				break
			}
		}
		if allTrue {
			nextTasks, err := getSchedulerTasks(initTask, to.taskData[initTask].inData, to.pWorker)
			if err != nil {
				return nil, nil, fmt.Errorf("error getting scheduler tasks: %v", err)
			}
			tasksRunnable = append(tasksRunnable, nextTasks...)
		}
	}
	return to, tasksRunnable, nil
}

func (to *TaskOrchestrator) GetInData(st *SchedulerTask) []*Data {
	taskData := to.taskData[st.taskPtr].inData

	if st.dataParallelism == 1 {
		return taskData
	}

	subtaskData := make([]*Data, len(taskData))
	for i := 0; i < len(taskData); i++ {
		if taskData[i] == nil { // -> whole input set (:all)
			subtaskData[i] = taskData[i]
		} else { // -> partial input set (:keyed / :each)
			subtaskData[i] = taskData[i].GetItems(st.dataIdxs[i])
		}
	}
	return subtaskData
}

func (to *TaskOrchestrator) SetOutData(st *SchedulerTask, data []*Data) error {
	if to.dataType == Unknown && len(data) > 0 {
		to.dataType = data[0].dType
	}

	if st.dataParallelism == 1 { // -> one execution context
		if len(data) == int(st.taskPtr.NumOut) {
			to.taskData[st.taskPtr].outData = data
		} else {
			logrus.Warnf(
				"got %d output data objects, statement has %d returns -> excess data will be cut, using empty data object where data is missing",
				len(data), st.taskPtr.NumOut,
			)
			taskData := make([]*Data, st.taskPtr.NumOut)
			for i := 0; i < int(st.taskPtr.NumOut); i++ {
				if i < len(data) {
					taskData[i] = data[i]
				} else {
					taskData[i] = NewEmptyData(to.dataType)
				}
			}
			to.taskData[st.taskPtr].outData = taskData
		}
	} else { // -> multiple contexts for the same task (dataParallelism > 1)
		// check outData is initialized
		to.taskData[st.taskPtr].outDataMutex.Lock()
		outData := to.taskData[st.taskPtr].outData
		if outData == nil {
			outData = make([]*Data, st.taskPtr.NumOut)
			for i := 0; i < int(st.taskPtr.NumOut); i++ {
				outData[i] = NewEmptyData(to.dataType)
			}
			to.taskData[st.taskPtr].outData = outData
		}
		to.taskData[st.taskPtr].outDataMutex.Unlock()

		// add items from subtask
		if len(data) != int(st.taskPtr.NumOut) {
			logrus.Warnf(
				"got %d output data objects, statement has %d returns -> excess data will be cut, using empty data object where data is missing",
				len(data), st.taskPtr.NumOut,
			)
		}
		for i := 0; i < min(int(st.taskPtr.NumOut), len(data)); i++ {
			err := outData[i].AddItems(data[i]) // AddItems is thread safe -> no need to acquire a lock
			if err != nil {
				return fmt.Errorf("error adding data items: %v", err)
			}
		}
	}
	return nil
}

// SetDone assumes that the output data has been set already using SetOutData
func (to *TaskOrchestrator) SetDone(st *SchedulerTask) (bool, []*SchedulerTask) {
	st.subtasksLeft.Add(-1)
	if st.subtasksLeft.Load() != int32(0) {
		return false, nil
	}

	tData := to.taskData[st.taskPtr]
	var tasksRunnable []*SchedulerTask
	for i, consumer := range st.taskPtr.ConsumerTasks {
		consumerData := to.taskData[consumer]
		consumerData.inDataMutex.Lock()

		consumerData.inData[st.taskPtr.ConsumerDataDstIdx[i]] = tData.outData[st.taskPtr.ConsumerDataSrcIdx[i]]

		allTrue := true
		for _, arg := range consumerData.inData {
			if arg == nil {
				allTrue = false
				break
			}
		}
		if allTrue {
			nextTasks, err := getSchedulerTasks(consumer, to.taskData[consumer].inData, to.pWorker)
			if err != nil {
				logrus.Errorf("error getting scheduler tasks: %v", err)
			}
			tasksRunnable = append(tasksRunnable, nextTasks...)
		}

		consumerData.inDataMutex.Unlock()
	}

	return true, tasksRunnable
}

func (to *TaskOrchestrator) CollectOutData() []*Data {
	outData := make([]*Data, to.wf.NumOut)
	for i, outStmt := range to.wf.OutTasks {
		outData[i] = to.taskData[outStmt].outData[to.wf.OutDataSrcIdx[i]]
	}
	return outData
}

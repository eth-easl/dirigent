package workflow

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"sync"
)

type DataType int

const (
	BytesData DataType = iota
	DandelionData
)

type Data struct {
	dType DataType
	bytes []byte
	ddPtr InputSet
}

func (d *Data) GetType() DataType {
	return d.dType
}
func (d *Data) GetBytes() []byte {
	return d.bytes
}
func (d *Data) GetInputSet() *InputSet {
	return &d.ddPtr
}
func NewBytesData(bs []byte) *Data {
	return &Data{
		dType: BytesData,
		bytes: bs,
	}
}
func NewDandelionData(iPtr InputSet) *Data {
	return &Data{
		dType: DandelionData,
		ddPtr: iPtr,
	}
}
func NewEmptyData() *Data {
	return &Data{
		bytes: []byte{},
		ddPtr: EmptyInputSet(),
	}
}

type TaskData struct {
	inData      []*Data
	inDataMutex sync.RWMutex
	outData     []*Data
}

type TaskOrchestrator struct {
	wf       *Workflow
	taskData map[*Task]*TaskData
}

func GetInitialRunnable(wf *Workflow, inData []*Data) (*TaskOrchestrator, []*Task, error) {
	if len(inData) != int(wf.NumIn) {
		return nil, nil, fmt.Errorf("got %d input data objects, composition has %d params", len(inData), wf.NumIn)
	}

	d := &TaskOrchestrator{
		wf:       wf,
		taskData: make(map[*Task]*TaskData),
	}
	for _, t := range wf.Tasks {
		d.taskData[t] = &TaskData{
			inData: make([]*Data, t.NumIn),
		}
	}

	var tasksRunnable []*Task
	for i, initTask := range wf.InitialTasks {
		d.taskData[initTask].inData[wf.InitialDataDstIdx[i]] = inData[wf.InitialDataSrcIdx[i]]

		allTrue := true
		for _, arg := range d.taskData[initTask].inData {
			if arg == nil {
				allTrue = false
				break
			}
		}
		if allTrue {
			tasksRunnable = append(tasksRunnable, initTask)
		}
	}
	return d, tasksRunnable, nil
}

func (d *TaskOrchestrator) GetInData(t *Task) []*Data {
	return d.taskData[t].inData
}

func (d *TaskOrchestrator) SetOutData(t *Task, data []*Data) error {
	if len(data) != int(t.NumOut) {
		logrus.Warnf("got %d output data objects, statement has %d returns -> excess data will be cut, using empty data object where data is missing", len(data), t.NumOut)
		taskData := make([]*Data, t.NumOut)
		for i := 0; i < int(t.NumOut); i++ {
			if i < len(data) {
				taskData[i] = data[i]
			} else {
				taskData[i] = NewEmptyData()
			}
		}
		d.taskData[t].outData = taskData
		return nil
	}
	d.taskData[t].outData = data
	return nil
}

// SetDone assumes that the output data of this task has been set already
func (d *TaskOrchestrator) SetDone(t *Task) []*Task {
	tData := d.taskData[t]
	var tasksRunnable []*Task
	for i, consumer := range t.ConsumerTasks {
		consumerData := d.taskData[consumer]
		consumerData.inDataMutex.Lock()

		consumerData.inData[t.ConsumerDataDstIdx[i]] = tData.outData[t.ConsumerDataSrcIdx[i]]

		allTrue := true
		for _, arg := range consumerData.inData {
			if arg == nil {
				allTrue = false
				break
			}
		}
		if allTrue {
			tasksRunnable = append(tasksRunnable, consumer)
		}

		consumerData.inDataMutex.Unlock()
	}

	return tasksRunnable
}

func (d *TaskOrchestrator) CollectOutData() []*Data {
	outData := make([]*Data, d.wf.NumOut)
	for i, outStmt := range d.wf.OutTasks {
		outData[i] = d.taskData[outStmt].outData[d.wf.OutDataSrcIdx[i]]
	}
	return outData
}

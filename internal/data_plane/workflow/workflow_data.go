package workflow

import (
	"fmt"
	"sync"
)

type DataType int

const (
	Unknown DataType = iota
	BytesData
	DandelionData
)

type Data struct {
	dType DataType
	bytes [][]byte
	dData InputSet
	sync.RWMutex
}

func (d *Data) GetType() DataType {
	return d.dType
}
func (d *Data) GetBytes() [][]byte {
	d.RLock()
	defer d.RUnlock()
	return d.bytes
}
func (d *Data) GetInputSet() *InputSet {
	d.RLock()
	defer d.RUnlock()
	return &d.dData
}

func NewBytesData(bs [][]byte) *Data {
	return &Data{
		dType: BytesData,
		bytes: bs,
	}
}
func NewDandelionData(set InputSet) *Data {
	return &Data{
		dType: DandelionData,
		dData: set,
	}
}
func NewEmptyData(dType DataType) *Data {
	switch dType {
	case BytesData:
		return &Data{
			dType: BytesData,
			bytes: [][]byte{},
		}
	case DandelionData:
		return &Data{
			dType: DandelionData,
			dData: EmptyInputSet(),
		}
	default:
		return nil
	}
}

func (d *Data) GetDataParallelism(s Sharding) [][]int {
	switch d.dType {
	case BytesData:
		switch s {
		case ShardingAll, ShardingAny:
			return [][]int{}
		case ShardingKeyed:
			keyMap := make(map[rune][]int)
			for i, b := range d.bytes {
				keyMap[rune(b[len(b)-2])] = append(keyMap[rune(b[len(b)-2])], i)
			}
			out := make([][]int, 0, len(keyMap))
			for _, key := range keyMap {
				out = append(out, key)
			}
			return out
		case ShardingEach:
			out := make([][]int, len(d.bytes))
			for i := 0; i < len(d.bytes); i++ {
				out[i] = []int{i}
			}
			return out
		default:
			return nil
		}
	case DandelionData:
		return d.dData.GetDataParallelism(s)
	default:
		return nil
	}
}

func (d *Data) AddItems(data *Data) error {
	if d.dType != data.dType {
		return fmt.Errorf("source data type does not match target data type")
	}

	d.Lock()
	defer d.Unlock()
	switch d.dType {
	case BytesData:
		d.bytes = append(d.bytes, data.bytes...)
	case DandelionData:
		d.dData.Items = append(d.dData.Items, data.dData.Items...)
	default:
		return fmt.Errorf("invalid data type")
	}

	return nil
}

func (d *Data) GetItems(idx []int) *Data {
	d.RLock()
	defer d.RUnlock()

	if len(idx) == 0 {
		return d
	}

	switch d.dType {
	case BytesData:
		outBytes := make([][]byte, len(idx))
		for i := 0; i < len(idx); i++ {
			outBytes[i] = d.bytes[idx[i]]
		}
		return NewBytesData(outBytes)

	case DandelionData:
		outSet := InputSet{
			Identifier: d.dData.Identifier,
			Items:      make([]InputItem, len(idx)),
		}
		for i := 0; i < len(idx); i++ {
			outSet.Items[i] = d.dData.Items[idx[i]]
		}
		return NewDandelionData(outSet)

	default:
		return nil
	}
}

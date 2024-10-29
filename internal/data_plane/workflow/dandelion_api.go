package workflow

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
)

// dandelion data structure

type InputItem struct {
	Identifier string `bson:"identifier"`
	Key        int64  `bson:"key"`
	Data       []byte `bson:"data"`
}
type InputSet struct {
	Identifier string      `bson:"identifier"`
	Items      []InputItem `bson:"items"`
}

func EmptyInputSet() InputSet {
	return InputSet{
		Identifier: "empty",
		Items:      []InputItem{},
	}
}

func (s *InputSet) GetDataParallelism(sharding Sharding) [][]int {
	switch sharding {
	case ShardingAll:
		return [][]int{}
	case ShardingKeyed: // item keys: [0, 1, 0, 2] -> [[0,2],[1],[3]]
		keyMap := make(map[int64][]int)
		for itmIdx, itm := range s.Items {
			keyMap[itm.Key] = append(keyMap[itm.Key], itmIdx)
		}
		out := make([][]int, 0, len(keyMap))
		for _, key := range keyMap {
			out = append(out, key)
		}
		return out
	case ShardingEach: // -> [[0],[1],...]
		out := make([][]int, 0, len(s.Items))
		for i := 0; i < len(s.Items); i++ {
			out = append(out, []int{i})
		}
	default:

	}
	logrus.Errorf("Got invalid sharding value %d.", sharding)
	return nil
}

func (s *InputSet) GetItemsWithIdx(indexes []int) []InputItem {
	items := make([]InputItem, len(indexes))
	for itmIdx, idx := range indexes {
		items[itmIdx] = s.Items[idx]
	}
	return items
}

// dandelion requests / responses

type RequestBody struct {
	Name string     `bson:"name"`
	Sets []InputSet `bson:"sets"`
}
type ResponseBody struct {
	Sets []InputSet `bson:"sets"`
}

func InvocationBody(funcName string, data []*Data) ([]byte, error) {
	var bodySets []InputSet
	if len(data) == 0 {
		bodySets = []InputSet{}
	} else {
		bodySets = make([]InputSet, len(data))
		for i, d := range data {
			s := d.GetInputSet()
			if s == nil {
				return nil, fmt.Errorf("dandelion invocation body expects DandelionData objects")
			}
			bodySets[i] = *s
		}
	}

	request := RequestBody{
		Name: funcName,
		Sets: bodySets,
	}

	reqBody, err := bson.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("error encoding dandelion invocation request body - %v", err)
	}

	return reqBody, nil
}

func GetResponseBody(outData []*Data) ([]byte, error) {
	outSets := make([]InputSet, len(outData))
	for i, data := range outData {
		set := data.GetInputSet()
		if set == nil {
			return nil, fmt.Errorf("dandelion response body expects dandelionData objects")
		}
		outSets[i] = *set
	}
	request := ResponseBody{outSets}

	reqBody, err := bson.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("error encoding dandelion response body - %v", err)
	}

	return reqBody, nil
}

func DeserializeRequestToData(reqData []byte) ([]*Data, error) {
	if len(reqData) == 0 {
		return nil, nil
	}

	var reqBody RequestBody
	err := bson.Unmarshal(reqData, &reqBody)
	if err != nil {
		return nil, fmt.Errorf("error deserializing request body - %v", err)
	}

	outData := make([]*Data, len(reqBody.Sets))
	for i, set := range reqBody.Sets {
		outData[i] = NewDandelionData(set)
	}

	return outData, nil
}

func DeserializeResponseToData(respData []byte) ([]*Data, error) {
	var respBody ResponseBody
	err := bson.Unmarshal(respData, &respBody)
	if err != nil {
		return nil, fmt.Errorf("error deserializing response body - %v", err)
	}

	outData := make([]*Data, len(respBody.Sets))
	for i, set := range respBody.Sets {
		outData[i] = NewDandelionData(set)
	}

	return outData, nil
}

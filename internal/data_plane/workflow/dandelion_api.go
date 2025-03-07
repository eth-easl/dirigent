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
		return out
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
	Sets       []InputSet `bson:"sets"`
	Timestamps string     `bson:"timestamps"`
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
			if len(s.Items) > 0 {
				for itmIdx, itm := range s.Items {
					logrus.Tracef("input set %d, item, %d -> size=%d", i, itmIdx, len(itm.Data))
					if len(itm.Data) < 120 { // only print data content of small sets to not overfill the log
						logrus.Tracef("  -> data: %s", string(itm.Data))
					}
				}
			} else {
				logrus.Tracef("input set %d -> empty", i)
			}
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
	request := ResponseBody{outSets, ""}

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

func DeserializeResponseToData(respData []byte) ([]*Data, string, error) {
	var respBody ResponseBody
	err := bson.Unmarshal(respData, &respBody)
	if err != nil {
		return nil, "", fmt.Errorf("error deserializing response body - %v", err)
	}

	timestamps := respBody.Timestamps

	if len(respBody.Sets) == 0 {
		return []*Data{}, timestamps, nil
	}

	// expect stdio as last set if it is part of the output by dandelion
	respSets := respBody.Sets
	lastSet := respBody.Sets[len(respBody.Sets)-1]
	if lastSet.Identifier == "stdio" {
		for _, itm := range lastSet.Items {
			if itm.Identifier == "stderr" && len(itm.Data) > 0 {
				logrus.Warnf("Invocation returned error (stderr):\n%s", itm.Data)
			} else {
				msg := " empty"
				if len(itm.Data) > 0 {
					msg = "\n" + string(itm.Data)
				}
				logrus.Debugf("Invocation returned output (%s):%s", itm.Identifier, msg)
			}
		}
		respSets = respSets[:len(respSets)-1]
	}

	outData := make([]*Data, len(respSets))
	for i, set := range respSets {
		outData[i] = NewDandelionData(set)
	}

	return outData, timestamps, nil
}

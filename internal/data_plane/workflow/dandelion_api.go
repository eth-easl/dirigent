package workflow

import (
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
)

type InputItem struct {
	Identifier string `bson:"identifier"`
	Key        int64  `bson:"key"`
	Data       []byte `bson:"data"`
}
type InputSet struct {
	Identifier string      `bson:"identifier"`
	Items      []InputItem `bson:"items"`
}
type RequestBody struct {
	Name string     `bson:"name"`
	Sets []InputSet `bson:"sets"`
}
type ResponseBody struct {
	Sets []InputSet `bson:"sets"`
}

func EmptyInputSet() InputSet {
	return InputSet{
		Identifier: "empty",
		Items:      []InputItem{},
	}
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

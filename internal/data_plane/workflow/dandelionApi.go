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

type DandelionData struct {
	Data InputSet
}

func (d *DandelionData) GetType() DataType {
	return DandelionSets
}
func (d *DandelionData) GetData() []byte {
	// concatenate all items into one output byte array
	var outData []byte
	for _, itemData := range d.Data.Items {
		outData = append(outData, itemData.Data...)
	}
	return outData
}
func (d *DandelionData) GetDandelionData() (*InputSet, error) {
	return &d.Data, nil
}

func BusyLoopInvocationBody(funcName string, data []Data) ([]byte, error) {
	if len(data) != 2 {
		return nil, fmt.Errorf("busy loop body expects 2 data items, got %d", len(data))
	}
	header, err := data[0].GetDandelionData()
	if err != nil {
		return nil, fmt.Errorf("busy loop body expects dandelionData objects: %v", err)
	}
	body, err := data[1].GetDandelionData()
	if err != nil {
		return nil, fmt.Errorf("busy loop body expects dandelionData objects: %v", err)
	}

	request := RequestBody{
		Name: funcName,
		Sets: []InputSet{
			*header,
			*body,
		},
	}

	reqBody, err := bson.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("error encoding dandelion busy loop request body - %v", err)
	}

	return reqBody, nil
}

func GetResponseBody(outData []Data) ([]byte, error) {
	outSets := make([]InputSet, len(outData))
	for i, data := range outData {
		set, err := data.GetDandelionData()
		if err != nil {
			return nil, fmt.Errorf("dandelion response body expects dandelionData objects: %v", err)
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

func DeserializeRequestToData(reqData []byte) ([]Data, error) {
	var reqBody RequestBody
	err := bson.Unmarshal(reqData, &reqBody)
	if err != nil {
		return nil, fmt.Errorf("error deserializing request body - %v", err)
	}

	outData := make([]Data, len(reqBody.Sets))
	for i, set := range reqBody.Sets {
		outData[i] = &DandelionData{Data: set}
	}

	return outData, nil
}

func DeserializeResponseToData(respData []byte) ([]Data, error) {
	var respBody ResponseBody
	err := bson.Unmarshal(respData, &respBody)
	if err != nil {
		return nil, fmt.Errorf("error deserializing response body - %v", err)
	}

	outData := make([]Data, len(respBody.Sets))
	for i, set := range respBody.Sets {
		outData[i] = &DandelionData{Data: set}
	}

	return outData, nil
}

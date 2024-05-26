package request_persistence

import (
	"cluster_manager/internal/data_plane/proxy/requests"
	"encoding/json"
	"github.com/sirupsen/logrus"
	"io"
)

type BufferedRequestWrapper struct {
	*requests.BufferedRequest

	Body    string `json:"Body,omitempty"`
	GetBody string `json:"GetBody,omitempty"`
	Cancel  string `json:"Cancel,omitempty"`
}

func CreateWrapper(request *requests.BufferedRequest) *BufferedRequestWrapper {
	body := ""

	if request.Body != nil {
		b, err := io.ReadAll(request.Body)
		if err != nil {
			logrus.Errorf("Failed to read body of the buffered request.")
			return nil
		}

		body = string(b)
	}

	return &BufferedRequestWrapper{
		Body:            body,
		BufferedRequest: request,
	}
}

func ExtractRequest(rawData []byte) *requests.BufferedRequest {
	var brw BufferedRequestWrapper
	err := json.Unmarshal(rawData, &brw)
	if err != nil {
		logrus.Errorf("Failed to unmarshal wrapped buffered request - %v", err)
		return nil
	}

	return brw.BufferedRequest
}

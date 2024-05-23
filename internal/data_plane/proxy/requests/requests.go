package requests

import (
	"context"
	"github.com/google/uuid"
	"io"
	"net/http"
	"strings"
	"time"
)

type BufferedRequest struct {
	http.Request

	// Async parameter
	Code                  string
	NumberTries           int
	Start                 time.Time
	SerializationDuration time.Duration
	PersistenceDuration   time.Duration
}

type BufferedResponse struct {
	StatusCode           int
	Body                 string
	Timestamp            time.Time
	UniqueCodeIdentifier string
	E2ELatency           time.Duration
}

func BufferedRequestFromRequest(request *http.Request, code string) *BufferedRequest {
	return &BufferedRequest{
		Request:     *request.Clone(context.Background()),
		Code:        code,
		NumberTries: 0,
		Start:       time.Now(),
	}
}

func GetUniqueRequestCode() string {
	return uuid.New().String()
}

func FillResponseWithBufferedResponse(rw http.ResponseWriter, response *BufferedResponse) error {
	_, err := io.Copy(rw, strings.NewReader(response.Body))
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		return err
	}

	rw.WriteHeader(response.StatusCode)

	return nil
}

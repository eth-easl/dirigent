package requests

import (
	"bytes"
	"github.com/google/uuid"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// TODO: Find minimal set of values we need to replicate
type BufferedRequest struct {
	Method           string
	URL              *url.URL
	Proto            string
	ProtoMajor       int
	ProtoMinor       int
	Header           http.Header
	Body             string
	ContentLength    int64
	TransferEncoding []string
	Close            bool
	Host             string
	Form             url.Values
	PostForm         url.Values
	MultipartForm    *multipart.Form
	Trailer          http.Header
	RemoteAddr       string
	// Async parameter
	Code                  string
	NumberTries           int
	Start                 time.Time
	SerializationDuration time.Duration
	PersistenceDuration   time.Duration
}

type BufferedResponse struct {
	StatusCode int
	Body       string
	Timestamp  time.Time
	Code       string
}

func BufferedRequestFromRequest(request *http.Request, code string) *BufferedRequest {
	buf := new(bytes.Buffer)
	buf.ReadFrom(request.Body)

	return &BufferedRequest{
		Method:           request.Method,
		URL:              request.URL,
		Proto:            request.Proto,
		ProtoMajor:       request.ProtoMajor,
		ProtoMinor:       request.ProtoMinor,
		Header:           request.Header,
		Body:             buf.String(),
		ContentLength:    request.ContentLength,
		TransferEncoding: request.TransferEncoding,
		Close:            request.Close,
		Host:             request.Host,
		Form:             request.Form,
		PostForm:         request.PostForm,
		MultipartForm:    request.MultipartForm,
		Trailer:          request.Trailer,
		RemoteAddr:       request.RemoteAddr,
		Code:             code,
		NumberTries:      0,
		Start:            time.Now(),
	}
}

func RequestFromBufferedRequest(bufferedRequest *BufferedRequest) *http.Request {
	return &http.Request{
		Method:           bufferedRequest.Method,
		URL:              bufferedRequest.URL,
		Proto:            bufferedRequest.Proto,
		ProtoMajor:       bufferedRequest.ProtoMajor,
		ProtoMinor:       bufferedRequest.ProtoMinor,
		Header:           bufferedRequest.Header,
		Body:             io.NopCloser(strings.NewReader(bufferedRequest.Body)),
		ContentLength:    bufferedRequest.ContentLength,
		TransferEncoding: bufferedRequest.TransferEncoding,
		Close:            bufferedRequest.Close,
		Host:             bufferedRequest.Host,
		Form:             bufferedRequest.Form,
		PostForm:         bufferedRequest.PostForm,
		MultipartForm:    bufferedRequest.MultipartForm,
		Trailer:          bufferedRequest.Trailer,
		RemoteAddr:       bufferedRequest.RemoteAddr,
	}
}

func GetUniqueRequestCode() string {
	return uuid.New().String()
}

func FillResponseWithBufferedResponse(rw http.ResponseWriter, response *BufferedResponse) error {
	rw.WriteHeader(response.StatusCode)

	if _, err := io.Copy(rw, strings.NewReader(response.Body)); err != nil {
		return err
	}

	return nil
}

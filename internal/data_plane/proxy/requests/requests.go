/*
 * MIT License
 *
 * Copyright (c) 2024 EASL
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */

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

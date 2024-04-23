package main

import (
	"cluster_manager/clients/register_service"
	"context"
	"crypto/tls"
	"fmt"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/http2"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func AsyncRequest() {

	logrus.Info("Invocating async function")

	////////////////////////////////////
	// INVOKE FUNCTION
	////////////////////////////////////

	client := http.Client{
		Timeout: 5 * time.Second,
		Transport: &http2.Transport{
			AllowHTTP: true,
			DialTLSContext: func(ctx context.Context, network, addr string, cfg *tls.Config) (net.Conn, error) {
				return net.Dial(network, addr)
			},
			DisableCompression: true,
		},
	}

	req, err := http.NewRequest("GET", "http://localhost:8080", nil)
	if err != nil {
		panic(err.Error())
	}

	req.Host = "test"

	req.Header.Set("workload", "empty")
	req.Header.Set("requested_cpu", strconv.Itoa(1))
	req.Header.Set("requested_memory", strconv.Itoa(1))
	req.Header.Set("multiplier", strconv.Itoa(0))

	resp, err := client.Do(req)
	if err != nil {
		logrus.Fatalf("Failed to establish a HTTP connection - %s\n", err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil || resp == nil || resp.StatusCode != http.StatusOK {
		msg := ""
		if err != nil {
			msg = err.Error()
		} else if resp == nil {
			msg = fmt.Sprintf("empty response - status code: %d", resp.StatusCode)
		} else if resp.StatusCode != http.StatusOK {
			msg = fmt.Sprintf("%s - status code: %d", body, resp.StatusCode)
		}
		logrus.Debugf("HTTP timeout for function")
		logrus.Fatalf(msg)
	}

	code := string(body[:])

	logrus.Infof("Unique code : %s", code)

	time.Sleep(1500 * time.Millisecond)

	req, err = http.NewRequest("GET", "http://localhost:8082", strings.NewReader(code))
	if err != nil {
		logrus.Fatalf(err.Error())
	}

	resp, err = client.Do(req)
	if err != nil {
		logrus.Fatalf(err.Error())
	}

	responseInBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		logrus.Fatalf(err.Error())
	}

	logrus.Infof("Response from server: %s", string(responseInBytes[:]))
}

func main() {
	register_service.Deployservice()
	AsyncRequest()
}

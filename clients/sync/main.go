package main

import (
	"cluster_manager/clients/register_service"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/net/http2"
)

func SyncRequest() {

	logrus.Info("Invocating sync function")

	////////////////////////////////////
	// INVOKE FUNCTION
	////////////////////////////////////

	client := http.Client{
		Timeout: 10 * time.Second,
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

	logrus.Infof("Response : %s", string(body[:]))
}

func main() {
	register_service.Deployservice()
	SyncRequest()
}

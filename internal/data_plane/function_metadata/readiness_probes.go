package function_metadata

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/http2"
	"net"
	"net/http"
	"sync"
	"time"
)

func createProbingDialer(network, addr string) (net.Conn, error) {
	dialer := &net.Dialer{
		DualStack: true,
		KeepAlive: 5 * time.Second,
		Timeout:   10 * time.Millisecond,
	}

	return dialer.Dial(network, addr)
}

var httpProbingClient = http.Client{
	Timeout: 10 * time.Millisecond,
	Transport: &http2.Transport{
		AllowHTTP: true,
		DialTLSContext: func(ctx context.Context, network, addr string, cfg *tls.Config) (net.Conn, error) {
			return createProbingDialer(network, addr)
		},
		DisableCompression: true,
	},
}

func SendReadinessProbe(url string) (time.Duration, bool) {
	wg := &sync.WaitGroup{}
	wg.Add(1)

	// empirically determined to make 20 probes before failure
	/*	1 :  0.015
		2 :  0.02535
		3 :  0.032955000000000005
		4 :  0.042841500000000005
		5 :  0.05569395000000001
		6 :  0.07240213500000002
		7 :  0.09412277550000003
		8 :  0.12235960815000003
		9 :  0.15906749059500003
		10 :  0.20678773777350007
		11 :  0.26882405910555013
		12 :  0.3494712768372152
		13 :  0.45431265988837977
		14 :  0.5906064578548937
		15 :  0.7677883952113619
		16 :  0.9981249137747704
		17 :  1.2975623879072016
		18 :  1.6868311042793622
		19 :  2.1928804355631715
		20 :  2.8507445662321222 */
	expBackoff := ExponentialBackoff{
		Interval:        0.015,
		ExponentialRate: 1.3,
		RetryNumber:     0,
		MaxDifference:   1,
	}

	passed := true
	start := time.Now()

	go func() {
		defer wg.Done()

		for {
			res, err := httpProbingClient.Get(fmt.Sprintf("http://%s/health", url))
			if err != nil || res == nil || (res != nil && res.StatusCode != http.StatusOK) {
				toSleep := expBackoff.Next()
				if toSleep < 0 {
					logrus.Error("Failed to pass readiness probe for ", url, ".")

					passed = false
					break
				}

				time.Sleep(time.Duration(int(toSleep*1000)) * time.Millisecond)

				continue
			}

			break
		}
	}()

	wg.Wait()

	elapsed := time.Since(start)

	logrus.Trace("Passed readiness probe for ", url, " in ", elapsed.Milliseconds(), " ms")
	return elapsed, passed
}

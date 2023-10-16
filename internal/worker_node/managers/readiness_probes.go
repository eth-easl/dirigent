package managers

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

	// empirically determined
	/*	1 :  0.02
		2 :  0.045
		3 :  0.0675
		4 :  0.10125
		5 :  0.151875
		6 :  0.2278125
		7 :  0.34171875
		8 :  0.512578125
		9 :  0.7688671875
		10 :  1.15330078125
		11 :  1.729951171875
		12 :  2.5949267578125
		13 :  -1
		14 :  -1
		15 :  -1
		16 :  -1
		17 :  -1
		18 :  -1
		19 :  -1
		20 :  -1 */
	expBackoff := ExponentialBackoff{
		Interval:        0.020,
		ExponentialRate: 1.5,
		RetryNumber:     0,
		MaxDifference:   1,
	}

	passed := true
	start := time.Now()

	tries := 0
	go func() {
		defer wg.Done()

		for {
			tries++

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

	logrus.Trace("Passed readiness probe for ", url, " from ", tries, " attempt in ", elapsed.Milliseconds(), " ms")
	return elapsed, passed
}

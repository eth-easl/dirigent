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

package managers

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"io"
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
	Transport: &http.Transport{
		DialContext: (&net.Dialer{
			Timeout: 2 * time.Second,
		}).DialContext,
		DisableCompression:  true,
		IdleConnTimeout:     2 * time.Second,
		MaxIdleConns:        3000,
		MaxIdleConnsPerHost: 3000,
	},
}

func SendReadinessProbe(url string) (time.Duration, bool) {
	wg := &sync.WaitGroup{}
	wg.Add(1)

	// empirically determined
	/*	1 :  0.01
		2 :  0.018225
		3 :  0.024603750000000004
		4 :  0.03321506250000001
		5 :  0.04484033437500002
		6 :  0.06053445140625002
		7 :  0.08172150939843753
		8 :  0.11032403768789069
		9 :  0.14893745087865243
		10 :  0.2010655586861808
		11 :  0.2714385042263441
		12 :  0.36644198070556455
		13 :  0.4946966739525122
		14 :  0.6678405098358916
		15 :  0.9015846882784535
		16 :  1.2171393291759125
		17 :  1.643138094387482
		18 :  2.2182364274231006
		19 :  2.994619177021186
		20 :  4.042735888978601 */
	expBackoff := ExponentialBackoff{
		Interval:        0.01,
		ExponentialRate: 1.35,
		RetryNumber:     0,
		MaxDifference:   2,
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
				handleBodyClosing(res)

				toSleep := expBackoff.Next()
				if toSleep < 0 {
					passed = false
					break
				}

				time.Sleep(time.Duration(int(toSleep*1000)) * time.Millisecond)

				continue
			} else {
				handleBodyClosing(res)
			}

			break
		}
	}()

	wg.Wait()

	elapsed := time.Since(start)

	if passed {
		logrus.Trace("Passed readiness probe for ", url, " from ", tries, " attempt in ", elapsed.Milliseconds(), " ms")
	} else {
		logrus.Error("Failed to pass readiness probe for ", url, ".")
	}

	return elapsed, passed
}

func handleBodyClosing(response *http.Response) {
	if response == nil || response.Body == nil {
		return
	}

	_, err := io.Copy(io.Discard, response.Body)
	if err != nil {
		logrus.Errorf("Error reading the response body - %v", err)
	}

	err = response.Body.Close()
	if err != nil {
		logrus.Errorf("Error closing the response body - %v", err)
	}
}

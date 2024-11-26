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
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestGetUniqueValue(t *testing.T) {
	ipm := &IPManager{}
	wg := &sync.WaitGroup{}

	tries := 1000
	var sum uint32 = 0

	wg.Add(tries)
	for i := 0; i < tries; i++ {
		go func() {
			atomic.AddUint32(&sum, ipm.getUniqueCounterValue())
			wg.Done()
		}()
	}

	wg.Wait()

	computedSum := atomic.LoadUint32(&sum)
	expectedSum := uint32(tries * (tries - 1) / 2)
	if computedSum != expectedSum {
		t.Error("Bad IP manager. Concurrency error.")
	}
}

func TestIPAddressGeneration(t *testing.T) {
	rand.Seed(time.Now().UnixNano())

	ipm := &IPManager{}
	barrier := &sync.WaitGroup{}

	tries := 256 + rand.Intn(32)*256

	barrier.Add(tries)
	for i := 0; i < tries; i++ {
		go func() {
			c, d := ipm.generateRawGatewayIP()

			if c > 255 || d > 255 {
				t.Errorf("Invalid IP address received - x.x.%d.%d", c, d)
			}

			barrier.Done()
		}()
	}

	barrier.Wait()
}

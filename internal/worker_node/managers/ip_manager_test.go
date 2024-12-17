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

func TestCIDRToPrefix(t *testing.T) {
	if CIDRToPrefix("10.11.0.0/16") != "10.11" {
		t.Error("Expected 10.11")
	}
}

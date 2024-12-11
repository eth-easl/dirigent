package ipam

import (
	"sync"
	"testing"
)

func TestIPAMTakeAndReturn(t *testing.T) {
	m := NewIPAM("dynamic")
	allocated := map[string]bool{}

	// take 8 CIDRs
	for i := 0; i < 8; i++ {
		cidr, err := m.ReserveCIDR()
		if err != nil {
			t.Error(err)
		}

		allocated[cidr] = true
	}

	// return 8 CIDRs
	for cidr := range allocated {
		m.ReleaseCIDR(cidr)
	}

	// take 16 CIDRs
	expected := map[string]bool{
		"11.0.0.0/16": false, "11.1.0.0/16": false, "11.2.0.0/16": false, "11.3.0.0/16": false,
		"11.4.0.0/16": false, "11.5.0.0/16": false, "11.6.0.0/16": false, "11.7.0.0/16": false,
		"11.8.0.0/16": false, "11.9.0.0/16": false, "11.10.0.0/16": false, "11.11.0.0/16": false,
		"11.12.0.0/16": false, "11.13.0.0/16": false, "11.14.0.0/16": false, "11.15.0.0/16": false,
	}

	for i := 0; i < 16; i++ {
		cidr, err := m.ReserveCIDR()
		if err != nil {
			t.Error(err)
		}

		if _, ok := expected[cidr]; !ok {
			t.Error("Got an unexpected CIDR range.")
		}
	}
}

func TestPoolExhaustion(t *testing.T) {
	m := NewIPAM("dynamic")
	wg := sync.WaitGroup{}

	wg.Add(MaximumClusterSize)
	for i := 0; i < MaximumClusterSize; i++ {
		go func() {
			defer wg.Done()

			_, err := m.ReserveCIDR()
			if err != nil {
				t.Error(err)
			}
		}()
	}

	wg.Wait()

	_, err := m.ReserveCIDR()
	if err == nil {
		t.Error("Error should have happened")
	}
}

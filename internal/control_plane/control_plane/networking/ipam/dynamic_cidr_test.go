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
		cidr, err := m.GetUnallocatedCIDR()
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
		cidr, err := m.GetUnallocatedCIDR()
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

			_, err := m.GetUnallocatedCIDR()
			if err != nil {
				t.Error(err)
			}
		}()
	}

	wg.Wait()

	_, err := m.GetUnallocatedCIDR()
	if err == nil {
		t.Error("Error should have happened")
	}
}

func TestCIDRToCounter(t *testing.T) {
	if cidrToCounter("11.0.0.0/16") != 0 ||
		cidrToCounter("12.0.0.0/16") != 256 ||
		cidrToCounter("12.12.0.0/16") != 268 ||
		cidrToCounter("99.255.0.0/16") != 22783 {
		t.Error("Unexpected value.")
	}
}

func TestCIDRReconstruction(t *testing.T) {
	tests := []struct {
		testName       string
		input          []string
		expectedOutput []string
	}{
		{
			testName:       "ipam_reconstruction_blank",
			input:          []string{},
			expectedOutput: []string{},
		},
		{
			testName:       "ipam_reconstruction_sequential",
			input:          []string{"11.0.0.0/16", "11.1.0.0/16", "11.2.0.0/16"},
			expectedOutput: []string{},
		},
		{
			testName: "ipam_reconstruction_non_sequential",
			input:    []string{"11.0.0.0/16", "11.6.0.0/16", "11.7.0.0/16"},
			expectedOutput: []string{
				"11.1.0.0/16",
				"11.2.0.0/16",
				"11.3.0.0/16",
				"11.4.0.0/16",
				"11.5.0.0/16",
			},
		},
		{
			testName: "ipam_reconstruction_non_sequential_15",
			input:    []string{"11.15.0.0/16"},
			expectedOutput: []string{
				"11.0.0.0/16",
				"11.1.0.0/16",
				"11.2.0.0/16",
				"11.3.0.0/16",
				"11.4.0.0/16",
				"11.5.0.0/16",
				"11.6.0.0/16",
				"11.7.0.0/16",
				"11.8.0.0/16",
				"11.9.0.0/16",
				"11.10.0.0/16",
				"11.11.0.0/16",
				"11.12.0.0/16",
				"11.13.0.0/16",
				"11.14.0.0/16",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			m := NewDynamicCIDRManager()
			m.ReserveCIDRs(tt.input)

			m.disableFurtherPopulation()

			if len(tt.expectedOutput) != m.poolLength() {
				t.Error("Unexpected length of CIDR pool.")
			}

			for i := 0; i < len(tt.expectedOutput); i++ {
				cidr, _ := m.GetUnallocatedCIDR()

				if tt.expectedOutput[i] != cidr {
					t.Errorf("Unexpected CIDR value - got: %s - expected: %s", cidr, tt.expectedOutput[i])
				}
			}
		})
	}
}

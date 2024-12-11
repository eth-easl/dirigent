package ipam

import "testing"

func TestStaticCIDR(t *testing.T) {
	cidr := "10.20.0.0./16"

	m := NewStaticCIDRManager(cidr)

	got, err := m.ReserveCIDR()
	if err != nil {
		t.Error("Unexpected error from static CIDR manager")
	} else if got != cidr {
		t.Error("Got unexpected CIDR from static CIDR manager")
	}
}

package common

import "testing"

func TestDifference(t *testing.T) {
	s1 := []string{"a", "b", "c"}
	s2 := []string{"b", "c", "d"}

	lr := difference(s1, s2)
	if lr == nil || len(lr) != 1 || lr[0] != "a" {
		t.Error("s1 \\ s2 yielded a wrong result.")
	}

	rr := difference(s2, s1)
	if rr == nil || len(rr) != 1 || rr[0] != "d" {
		t.Error("s2 \\ s1 yielded a wrong result.")
	}
}

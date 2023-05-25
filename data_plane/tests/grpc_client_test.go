package tests

import (
	"testing"
)

func TestInvocationProxying(t *testing.T) {
	err := FireInvocation(t, "localhost", "8080")
	if err != nil {
		t.Error("Invocation failed.")
	}
}

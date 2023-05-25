package tests

import (
	"github.com/sirupsen/logrus"
	"testing"
)

func TestInvocationProxying(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)

	err := FireInvocation(t, "localhost", "8080")
	if err != nil {
		t.Error("Invocation failed.")
	}
}

func Test_100Invocations(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)

	invocationCount := 100
	ch := make(chan struct{})

	for i := 0; i < invocationCount; i++ {
		go func() {
			err := FireInvocation(t, "localhost", "8080")
			if err != nil {
				t.Error("Invocation failed.")
			}

			ch <- struct{}{}
		}()
	}

	for i := 0; i < invocationCount; i++ {
		<-ch
	}
}

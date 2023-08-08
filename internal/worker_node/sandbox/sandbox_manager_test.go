package sandbox

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSandboxManager(t *testing.T) {
	i := 0
	manager := NewSandboxManager()

	for ; i <= 1000; i++ {
		manager.AddSandbox(strconv.Itoa(i), &Metadata{})
		assert.Equal(t, len(manager.Metadata), i+1)
	}

	for ; i >= 0; i-- {
		manager.DeleteSandbox(strconv.Itoa(i))
		assert.Equal(t, len(manager.Metadata), i)
	}
}

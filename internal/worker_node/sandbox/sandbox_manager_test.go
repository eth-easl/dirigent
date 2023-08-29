package sandbox

import (
	"github.com/stretchr/testify/assert"
	"strconv"
	"testing"
)

func TestSimpleAddDelete(t *testing.T) {
	manager := NewSandboxManager()

	manager.AddSandbox("test", &Metadata{})
	manager.DeleteSandbox("test")
}

func TestSandboxManager(t *testing.T) {
	i := 0
	manager := NewSandboxManager()

	for ; i <= 1000; i++ {
		manager.AddSandbox(strconv.Itoa(i), &Metadata{})
		assert.Equal(t, manager.Metadata.Len(), i+1)
	}

	i--

	for ; i >= 0; i-- {
		manager.DeleteSandbox(strconv.Itoa(i))
		assert.Equal(t, manager.Metadata.Len(), i)
	}
}

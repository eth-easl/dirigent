package sandbox

import (
	"github.com/stretchr/testify/assert"
	"strconv"
	"testing"
)

const (
	mockHost string = "mockHost"
	mockkey  string = "mockKey"
)

func TestSimpleAddDelete(t *testing.T) {
	manager := NewSandboxManager(mockHost)

	manager.AddSandbox(mockkey, &Metadata{})
	manager.DeleteSandbox(mockkey)
}

func TestSandboxManager(t *testing.T) {
	i := 0
	manager := NewSandboxManager(mockHost)

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

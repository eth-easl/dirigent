package control_plane

import (
	"cluster_manager/pkg/tracing"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestEvictionPolicyEmpty(t *testing.T) {
	emptyList := make([]*Endpoint, 0)

	toEvict, currentState := EvictionPolicy(emptyList)

	assert.Nil(t, toEvict, "Evicted point should be nil")
	assert.Len(t, currentState, 0, "No object should ne present in the list")
}

func TestEvictionPolicy(t *testing.T) {
	list := make([]*Endpoint, 0)

	list = append(list, &Endpoint{
		SandboxID:       "",
		URL:             "",
		Node:            nil,
		HostPort:        0,
		CreationHistory: tracing.ColdStartLogEntry{},
	})

	list = append(list, &Endpoint{
		SandboxID:       "",
		URL:             "",
		Node:            nil,
		HostPort:        0,
		CreationHistory: tracing.ColdStartLogEntry{},
	})

	list = append(list, &Endpoint{
		SandboxID:       "",
		URL:             "",
		Node:            nil,
		HostPort:        0,
		CreationHistory: tracing.ColdStartLogEntry{},
	})

	toEvict, currentState := EvictionPolicy(list)

	assert.NotNil(t, toEvict, "Evicted point should not be nil")
	assert.Len(t, currentState, 2, "Two objects should be present in the list")
}

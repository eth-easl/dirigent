package sandbox

import (
	"github.com/containerd/containerd"
	"sync"
)

type Manager struct {
	sync.Mutex
	metadata map[string]*Metadata
}

type Metadata struct {
	Task        containerd.Task
	Container   containerd.Container
	ExitChannel <-chan containerd.ExitStatus
	IP          string
	NetNs       string
}

func (m *Manager) AddSandbox(key string, metadata *Metadata) {
	m.Lock()
	defer m.Unlock()

	m.metadata[key] = metadata
}

func (m *Manager) DeleteSandbox(key string) *Metadata {
	m.Lock()
	defer m.Unlock()

	res := m.metadata[key]
	delete(m.metadata, key)

	return res
}

package sandbox

import (
	"sync"

	"github.com/containerd/containerd"
)

type Manager struct {
	sync.Mutex
	Metadata map[string]*Metadata
}

type Metadata struct {
	Task        containerd.Task
	Container   containerd.Container
	ExitChannel <-chan containerd.ExitStatus
	HostPort    int
	IP          string
	GuestPort   int
	NetNs       string
}

func NewSandboxManager() *Manager {
	return &Manager{
		Metadata: make(map[string]*Metadata),
	}
}

func (m *Manager) AddSandbox(key string, metadata *Metadata) {
	m.Lock()
	defer m.Unlock()

	m.Metadata[key] = metadata
}

func (m *Manager) DeleteSandbox(key string) *Metadata {
	m.Lock()
	defer m.Unlock()

	res := m.Metadata[key]
	delete(m.Metadata, key)

	return res
}

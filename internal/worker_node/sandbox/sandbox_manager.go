package sandbox

import (
	"cluster_manager/pkg/atomic_map"
	"github.com/containerd/containerd"
)

type Manager struct {
	Metadata atomic_map.AtomicMap[string, *Metadata]
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
		Metadata: atomic_map.NewAtomicMap[string, *Metadata](),
	}
}

func (m *Manager) AddSandbox(key string, metadata *Metadata) {
	m.Metadata.Set(key, metadata)
}

func (m *Manager) DeleteSandbox(key string) *Metadata {
	res, ok := m.Metadata.Get(key)
	if !ok {
		panic("value not present in map")
	}

	m.Metadata.Delete(key)

	return res
}

package sandbox

import (
	"cluster_manager/api/proto"
	"cluster_manager/pkg/atomic_map"
	"github.com/containerd/containerd"
	"sync"
)

type Manager struct {
	Metadata atomic_map.AtomicMap[string, *Metadata]

	nodeName string
}

type Metadata struct {
	ServiceName string

	Task        containerd.Task
	Container   containerd.Container
	ExitChannel <-chan containerd.ExitStatus
	HostPort    int
	IP          string
	GuestPort   int
	NetNs       string

	SignalKillBySystem sync.WaitGroup
}

func NewSandboxManager(nodeName string) *Manager {
	return &Manager{
		Metadata: atomic_map.NewAtomicMap[string, *Metadata](),
		nodeName: nodeName,
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

func (m *Manager) ListEndpoints() (*proto.EndpointsList, error) {
	list := &proto.EndpointsList{}

	keys, values := m.Metadata.KeyValues()
	for i := 0; i < len(keys); i++ {
		list.Endpoint = append(list.Endpoint, &proto.Endpoint{
			SandboxID:   keys[i],
			URL:         values[i].IP,
			NodeName:    m.nodeName,
			ServiceName: values[i].ServiceName,
			HostPort:    int32(values[i].HostPort),
		})
	}

	return list, nil
}

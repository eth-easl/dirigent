package managers

import (
	"cluster_manager/pkg/atomic_map"
	"cluster_manager/proto"
	"fmt"
)

type SandboxManager struct {
	Metadata atomic_map.AtomicMap[string, *Metadata]

	nodeName string
}

type RuntimeMetadata interface{}

type Metadata struct {
	ServiceName string

	RuntimeMetadata RuntimeMetadata

	IP        string
	GuestPort int
	NetNs     string

	ExitStatusChannel chan uint32
}

func NewSandboxManager(nodeName string) *SandboxManager {
	return &SandboxManager{
		Metadata: *atomic_map.NewAtomicMap[string, *Metadata](),
		nodeName: nodeName,
	}
}

func (m *SandboxManager) AddSandbox(key string, metadata *Metadata) {
	m.Metadata.Set(key, metadata)
}

func (m *SandboxManager) DeleteSandbox(key string) *Metadata {
	res, ok := m.Metadata.Get(key)
	if !ok {
		return nil
	}

	m.Metadata.Delete(key)

	return res
}

func (m *SandboxManager) ListEndpoints() (*proto.EndpointsList, error) {
	list := &proto.EndpointsList{}

	// TODO: check for consistency with containerd client list
	keys, values := m.Metadata.KeyValues()
	for i := 0; i < len(keys); i++ {
		list.Endpoint = append(list.Endpoint, &proto.Endpoint{
			SandboxID:   keys[i],
			URL:         fmt.Sprintf("%s:%d", values[i].IP, values[i].GuestPort),
			NodeName:    m.nodeName,
			ServiceName: values[i].ServiceName,
		})
	}

	return list, nil
}

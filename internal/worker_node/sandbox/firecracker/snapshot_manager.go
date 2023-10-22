package firecracker

import (
	"cluster_manager/internal/worker_node/managers"
)

type SnapshotManager struct {
	managers.RootFsManager[*SnapshotMetadata]
}

type SnapshotMetadata struct {
	MemoryPath   string
	SnapshotPath string

	HostDevName string
	MacAddress  string
	GatewayIP   string
	VMIP        string
}

func NewFirecrackerSnapshotManager() *SnapshotManager {
	return &SnapshotManager{}
}

func (sm *SnapshotManager) Exists(serviceName string) bool {
	_, ok := sm.Get(serviceName)

	return ok
}

func (sm *SnapshotManager) AddSnapshot(serviceID string, metadata *SnapshotMetadata) {
	sm.LoadOrStore(serviceID, metadata)
}

func (sm *SnapshotManager) FindSnapshot(serviceName string) (*SnapshotMetadata, bool) {
	return sm.Get(serviceName)
}

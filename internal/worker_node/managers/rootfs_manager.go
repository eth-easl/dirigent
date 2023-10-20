package managers

import (
	"cluster_manager/pkg/atomic_map"
)

type RootFsManager[K interface{}] struct {
	atomic_map.AtomicMap[string, K]
}

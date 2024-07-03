package image_storage

import (
	"cluster_manager/internal/control_plane/control_plane/core"
)

type ImageInfo struct {
	Count uint64
	Size  uint64
}

type ImageStorage interface {
	RegisterNoFetch(image string, size uint64, node core.WorkerNodeInterface)
	RegisterWithFetch(image string, node core.WorkerNodeInterface) error
	Get(url string) (ImageInfo, bool)
	Lock()
	Unlock()
}

func ParseImageStorage(storageType string) ImageStorage {
	switch storageType {
	case "oci":
		return NewOCIImageStorage()
	default:
		return NewDefaultImageStorage()
	}
}

package image_storage

import (
	"cluster_manager/internal/control_plane/control_plane/core"
	"cluster_manager/pkg/synchronization"

	"github.com/sirupsen/logrus"
)

type DefaultImageStorage struct {
	ImageStorage

	imageMap synchronization.SyncStructure[string, ImageInfo]
}

func NewDefaultImageStorage() *DefaultImageStorage {
	return &DefaultImageStorage{
		imageMap: synchronization.NewControlPlaneSyncStructure[string, ImageInfo](),
	}
}

func (s *DefaultImageStorage) registerImage(image string, size uint64) {
	if imageInfo, ok := s.imageMap.Get(image); ok {
		imageInfo.Count += 1
		s.imageMap.Set(image, imageInfo)
	} else {
		s.imageMap.Set(image, ImageInfo{
			Count: 1,
			Size:  size,
		})
	}
}

func (s *DefaultImageStorage) RegisterNoFetch(image string, size uint64, node core.WorkerNodeInterface) {
	if node.AddImage(image) {
		s.registerImage(image, size)
		logrus.Tracef("Registered image %s of size %d for node %s", image, size, node.GetName())
	}
}

func (s *DefaultImageStorage) RegisterWithFetch(image string, node core.WorkerNodeInterface) error {
	logrus.Tracef("Default image storage is being used")
	s.RegisterNoFetch(image, 0, node)
	return nil
}

func (s *DefaultImageStorage) Get(url string) (ImageInfo, bool) {
	return s.imageMap.Get(url)
}

func (s *DefaultImageStorage) Lock() {
	s.imageMap.Lock()
	logrus.Tracef("Image storage locked now")
}

func (s *DefaultImageStorage) Unlock() {
	s.imageMap.Unlock()
	logrus.Tracef("Image storage unlocked now")
}

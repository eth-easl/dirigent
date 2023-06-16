package sandbox

import (
	"context"
	"github.com/containerd/containerd"
	"sync"
)

type ImageManager struct {
	sync.RWMutex
	imageCache map[string]containerd.Image
}

func NewImageManager() *ImageManager {
	return &ImageManager{
		imageCache: make(map[string]containerd.Image),
	}
}

func (m *ImageManager) GetImage(ctx context.Context, containerdClient *containerd.Client, url string) (containerd.Image, error) {
	m.RLock()
	if _, ok := m.imageCache[url]; !ok {
		m.RUnlock()

		m.Lock()
		defer m.Unlock()

		return FetchImage(ctx, containerdClient, url)
	}

	image := m.imageCache[url]
	m.RUnlock()

	return image, nil
}

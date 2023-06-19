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

		image, err := FetchImage(ctx, containerdClient, url)
		if err == nil {
			// cache image to avoid subsequent pulls
			m.imageCache[url] = image
		}

		return image, err
	}

	image := m.imageCache[url]
	m.RUnlock()

	return image, nil
}

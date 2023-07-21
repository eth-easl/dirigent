package sandbox

import (
	"context"
	"sync"
	"time"

	"github.com/containerd/containerd"
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

func (m *ImageManager) GetImage(ctx context.Context, containerdClient *containerd.Client, url string) (containerd.Image, error, time.Duration) {
	start := time.Now()

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

		return image, err, time.Since(start)
	}

	image := m.imageCache[url]
	m.RUnlock()

	return image, nil, time.Since(start)
}

package sandbox

import (
	"cluster_manager/pkg/atomic_map"
	"context"
	"time"

	"github.com/containerd/containerd"
)

type ImageManager struct {
	imageCache atomic_map.AtomicMap[string, containerd.Image]
}

func NewImageManager() *ImageManager {
	return &ImageManager{
		imageCache: atomic_map.NewAtomicMap[string, containerd.Image](),
	}
}

func (m *ImageManager) GetImage(ctx context.Context, containerdClient *containerd.Client, url string) (containerd.Image, error, time.Duration) {
	start := time.Now()

	image, ok := m.imageCache.Get(url)
	if !ok {
		image, err := FetchImage(ctx, containerdClient, url)
		if err != nil {
			return image, err, time.Since(start)
		}

		m.imageCache.Set(url, image)

		return image, nil, time.Since(start)
	}

	return image, nil, time.Since(start)
}

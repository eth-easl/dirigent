package containerd

import (
	"cluster_manager/internal/worker_node/managers"
	"context"
	"time"

	"github.com/containerd/containerd"
)

type ImageManager struct {
	managers.RootFsManager[containerd.Image]
}

func NewContainerdImageManager() *ImageManager {
	return &ImageManager{}
}

func (m *ImageManager) GetImage(ctx context.Context, containerdClient *containerd.Client, url string) (containerd.Image, error, time.Duration) {
	start := time.Now()

	image, ok := m.Get(url)
	if !ok {
		var err error

		image, err = FetchImage(ctx, containerdClient, url)
		if err != nil {
			return image, err, time.Since(start)
		}

		m.Set(url, image)

		return image, nil, time.Since(start)
	}

	return image, nil, time.Since(start)
}

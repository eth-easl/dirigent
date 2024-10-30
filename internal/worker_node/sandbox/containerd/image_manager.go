package containerd

import (
	"cluster_manager/internal/worker_node/managers"
	"cluster_manager/pkg/atomic_map"
	"context"
	"time"

	"github.com/containerd/containerd"
	"github.com/sirupsen/logrus"
)

type ImageManager struct {
	managers.RootFsManager[containerd.Image]

	fetching atomic_map.AtomicMap[string, chan error]
}

func NewContainerdImageManager() *ImageManager {
	return &ImageManager{}
}

func (m *ImageManager) GetImage(ctx context.Context, containerdClient *containerd.Client, url string, opts ...containerd.RemoteOpt) (containerd.Image, error, time.Duration) {
	start := time.Now()

	image, ok := m.Get(url)
	if !ok {
		// The image is not cached in our image manager.
		var err error

		fetched, loaded := m.fetching.GetOrSet(url, make(chan error, 1))
		if loaded {
			// Image is being fetched from another goroutine.
			logrus.Debugf("Image %s is already being fetched, waiting...", url)

			err, failed := <-fetched
			if failed {
				// The other goroutine has sent the error in the channel,
				// meaning that fetching failed.
				logrus.Warnf("Image %s failed to fetch (other thread): %s", url, err.Error())
				fetched <- err
				return nil, err, time.Since(start)
			}

			// Channel has been closed, meaning that our image is fetched now.
			logrus.Debugf("Image %s has been fetched (other thread).", url)
			image, ok = m.Get(url)
			if !ok {
				logrus.Fatal("Image removed from cache right after fetching")
			}
		} else {
			// Nobody else is fetching the image right now, so let's fetch it.
			defer m.fetching.RemoveKey(url)
			logrus.Debugf("Fetching image %s...", url)
			image, err = FetchImage(ctx, containerdClient, url, opts...)

			if err != nil {
				// Let everyone else know there has been an error while
				// fetching.
				logrus.Warnf("Image %s failed to fetch (fetcher): %s", url, err.Error())
				fetched <- err
				return image, err, time.Since(start)
			}

			// Store the fetched image and let everyone else know it's there.
			logrus.Debugf("Image %s has been fetched (fetcher).", url)
			m.Set(url, image)
			close(fetched)
		}

		return image, nil, time.Since(start)
	}

	return image, nil, time.Since(start)
}

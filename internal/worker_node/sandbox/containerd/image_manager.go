/*
 * MIT License
 *
 * Copyright (c) 2024 EASL
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */

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

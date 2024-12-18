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
	"math/rand"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/go-cni"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

const cniConfigPath = "../../../../configs/cni.conf"

func TestCreateAContainer(t *testing.T) {
	t.Skip()
	logrus.SetLevel(logrus.TraceLevel)

	// fails to expose networking to the container
	rand.Seed(time.Now().UnixNano())

	client, err := containerd.New("/run/containerd/containerd.sock")
	assert.NoError(t, err, "Failed to create a containerd client")

	network, err := cni.New(cni.WithConfFile(cniConfigPath))
	assert.NoError(t, err, "Failed to open cni configuration")

	ctx := namespaces.WithNamespace(context.Background(), "default")

	start := time.Now()
	image, _ := FetchImage(ctx, client, "docker.io/cvetkovic/dirigent_empty_function:latest")

	logrus.Info("Image fetching - ", time.Since(start).Microseconds(), "μs")

	start = time.Now()
	container, err, _ := CreateContainer(ctx, client, image)
	assert.NoError(t, err, "Failed to create container")
	logrus.Info("Create container - ", time.Since(start).Microseconds(), "μs")

	start = time.Now()
	task, _, ip, netns, err, _, _ := StartContainer(ctx, container, network)
	assert.NoError(t, err, "Failed to start container")
	logrus.Info("Start container - ", time.Since(start).Microseconds(), "μs")

	sm := &managers.Metadata{
		RuntimeMetadata: ContainerdMetadata{
			Task:      task,
			Container: container,
		},

		IP:    ip,
		NetNs: netns,
	}

	go WatchExitChannel(nil, sm, func(metadata *managers.Metadata) string {
		return metadata.RuntimeMetadata.(ContainerdMetadata).Container.ID()
	})

	err = DeleteContainer(ctx, network, sm)
	assert.NoError(t, err, "Failed to delete container")
}

func TestParallelCreation(t *testing.T) {
	t.Skip()
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.TextFormatter{TimestampFormat: time.StampMilli, FullTimestamp: true})

	client, err := containerd.New("/run/containerd/containerd.sock")
	assert.NoError(t, err, "Failed to create a containerd client")

	network, err := cni.New(cni.WithConfFile(cniConfigPath))
	assert.NoError(t, err)

	ctx := namespaces.WithNamespace(context.Background(), "default")
	image, _ := FetchImage(ctx, client, "docker.io/cvetkovic/dirigent_empty_function:latest")

	REPETITIONS := 1
	for rep := 0; rep < REPETITIONS; rep++ {
		wg := sync.WaitGroup{}

		for i := 0; i < 5; i++ {
			wg.Add(1)

			go func() {
				container, err, _ := CreateContainer(ctx, client, image)
				assert.NoError(t, err, "Failed to create a container")

				start := time.Now()
				task, _, ip, netns, err, _, _ := StartContainer(ctx, container, network)
				assert.NoError(t, err, "Failed to start a container")

				sm := &managers.Metadata{
					RuntimeMetadata: ContainerdMetadata{
						Task:      task,
						Container: container,
					},

					IP:    ip,
					NetNs: netns,
				}

				logrus.Debug("Sandbox creation took: ", time.Since(start).Milliseconds(), " ms")
				time.Sleep(2 * time.Second)

				err = DeleteContainer(ctx, network, sm)
				assert.NoError(t, err, "Failed to delete container")

				wg.Done()
			}()
		}

		wg.Wait()
	}
}

func TestContainerFailureHandlerTriggering(t *testing.T) {
	t.Skip()
	logrus.SetLevel(logrus.TraceLevel)

	// fails to expose networking to the container
	rand.Seed(time.Now().UnixNano())
	pm := managers.NewProcessMonitor()

	client, err := containerd.New("/run/containerd/containerd.sock")
	assert.NoError(t, err, "Failed to create a containerd client")

	network, err := cni.New(cni.WithConfFile(cniConfigPath))
	assert.NoError(t, err, "Failed to open cni configuration")

	ctx := namespaces.WithNamespace(context.Background(), "default")

	start := time.Now()
	image, _ := FetchImage(ctx, client, "docker.io/cvetkovic/dirigent_empty_function:latest")

	logrus.Info("Image fetching - ", time.Since(start).Microseconds(), "μs")

	start = time.Now()
	container, err, _ := CreateContainer(ctx, client, image)
	assert.NoError(t, err, "Failed to create container")
	logrus.Info("Create container - ", time.Since(start).Microseconds(), "μs")

	start = time.Now()
	task, _, ip, netns, err, _, _ := StartContainer(ctx, container, network)
	assert.NoError(t, err, "Failed to start container")
	logrus.Info("Start container - ", time.Since(start).Microseconds(), "μs")

	sm := &managers.Metadata{
		RuntimeMetadata: ContainerdMetadata{
			Task:      task,
			Container: container,
		},

		IP:    ip,
		NetNs: netns,

		ExitStatusChannel: make(chan uint32),
	}
	pm.AddChannel(task.Pid(), sm.ExitStatusChannel)

	go WatchExitChannel(nil, sm, func(metadata *managers.Metadata) string {
		return metadata.RuntimeMetadata.(ContainerdMetadata).Container.ID()
	})

	// wait until 'WatchExitChannel' is ready
	time.Sleep(3 * time.Second)

	task.Kill(ctx, syscall.SIGKILL, containerd.WithKillAll)

	// fault handler otherwise won't be called
	time.Sleep(5 * time.Second)
}

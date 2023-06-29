package sandbox

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestDockerClient(t *testing.T) {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	assert.NoError(t, err)

	containerConfig := &container.Config{
		Image: "docker.io/cvetkovic/empty_function:latest",
		ExposedPorts: nat.PortSet{
			"80/tcp": struct{}{},
		},
	}

	hostConfig := &container.HostConfig{
		PortBindings: nat.PortMap{
			"80/tcp": []nat.PortBinding{
				{
					HostIP:   "0.0.0.0",
					HostPort: "10000",
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := cli.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, "")
	assert.NoError(t, err)

	err = cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{})
	assert.NoError(t, err)
}

func CreateSingle(t *testing.T, cli *client.Client) string {
	containerConfig := &container.Config{
		Image: "docker.io/cvetkovic/empty_function:latest",
	}
	hostConfig := &container.HostConfig{}

	id, err := CreateSandbox(cli, hostConfig, containerConfig)
	assert.NoError(t, err)

	return id
}

func TestCreateSandboxWithTeardown(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)

	cli := GetDockerClient()

	id := CreateSingle(t, cli)

	err := DeleteSandbox(cli, id)
	assert.NoError(t, err)
}

func TestParallelSandboxCreation(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.TextFormatter{TimestampFormat: time.StampMilli, FullTimestamp: true})

	wg := sync.WaitGroup{}

	for i := 0; i < 10; i++ {
		wg.Add(1)

		go func() {
			start := time.Now()
			cli := GetDockerClient()

			logrus.Debug("Create Docker client: ", time.Since(start).Microseconds(), " μs")

			start = time.Now()

			CreateSingle(t, cli)

			logrus.Debug("Sandbox creation took: ", time.Since(start).Milliseconds(), " ms")

			wg.Done()
		}()
	}

	wg.Wait()
}

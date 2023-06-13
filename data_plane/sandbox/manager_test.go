package sandbox

import (
	"context"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/sirupsen/logrus"
	"sync"
	"testing"
	"time"
)

func TestDockerClient(t *testing.T) {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		logrus.Fatal(err)
	}

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
	if err != nil {
		logrus.Fatal(err)
	}

	err = cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{})
	if err != nil {
		logrus.Fatal(err)
	}
}

func CreateSingle(t *testing.T, cli *client.Client) string {
	containerConfig := &container.Config{
		Image: "docker.io/cvetkovic/empty_function:latest",
	}
	hostConfig := &container.HostConfig{}

	id, err := CreateSandbox(cli, hostConfig, containerConfig)
	if err != nil {
		t.Fatal(err)
	}

	return id
}

func TestCreateSandboxWithTeardown(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)
	cli := GetDockerClient()

	id := CreateSingle(t, cli)

	err := DeleteSandbox(cli, id)
	if err != nil {
		t.Error(err)
	}
}

func TestParallelSandboxCreation(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)

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

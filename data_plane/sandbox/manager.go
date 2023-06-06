package sandbox

import (
	"context"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/sirupsen/logrus"
	"time"
)

const requestTimeout = 30 * time.Second

func GetDockerClient() *client.Client {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		logrus.Fatal("Failed to create a Docker client - ", err)
	}

	return cli
}

func CreateSandbox(cli *client.Client, hostConfig *container.HostConfig, containerConfig *container.Config) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()

	resp, err := cli.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, "")
	if err != nil {
		return "", err
	}

	err = cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{})
	if err != nil {
		return "", err
	}

	return resp.ID, nil
}

func DeleteSandbox(cli *client.Client, sandboxID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()

	// TODO: think about graceful shutdown
	return cli.ContainerRemove(ctx, sandboxID, types.ContainerRemoveOptions{Force: true})
}

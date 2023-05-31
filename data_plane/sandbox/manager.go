package sandbox

import (
	"context"
	"github.com/containerd/containerd"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/sirupsen/logrus"
	"log"
	"time"
)

func GetDockerClient() *client.Client {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		logrus.Fatal("Failed to create a Docker client - ", err)
	}

	return cli
}

func GetContainerdClient() *containerd.Client {
	cli, err := containerd.New("/run/containerd/containerd.sock")
	if err != nil {
		log.Fatal("Failed to create a containerd client")
	}

	return cli
}

func CreateSandbox(cli *client.Client, sandboxName string, hostConfig *container.HostConfig, containerConfig *container.Config) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	//containerName := fmt.Sprintf("%s-%d", sandboxName, rand.Int())
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

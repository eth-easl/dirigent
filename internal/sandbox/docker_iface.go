package sandbox

import (
	"context"
	"io"
	"os"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/sirupsen/logrus"
)

const requestTimeout = 30 * time.Second

func GetDockerClient() *client.Client {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		logrus.Fatal("Failed to create a Docker client - ", err)
	}

	return cli
}

func resolveImage(cli *client.Client, image string) error {
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()

	start := time.Now()

	reader, err := cli.ImagePull(ctx, image, types.ImagePullOptions{})
	if err != nil {
		return err
	}
	defer reader.Close()

	io.Copy(os.Stdout, reader)

	logrus.Debug("Image pull took: ", time.Since(start).Microseconds(), " μs")

	return nil
}

func isImageMissing(err error) bool {
	noSuchImageString := "No such image"
	return strings.Contains(err.Error(), noSuchImageString)
}

func tryFetchImage(cli *client.Client, containerImage string) error {
	logrus.Debug("Image not found. Fetching...")

	return resolveImage(cli, containerImage)
}

func CreateSandbox(cli *client.Client, hostConfig *container.HostConfig, containerConfig *container.Config) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()

	var r container.CreateResponse

	for {
		resp, err := cli.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, "")
		r = resp

		if isImageMissing(err) {
			err = tryFetchImage(cli, containerConfig.Image)
		}

		if err != nil {
			return "", err
		} else {
			break
		}
	}

	err := cli.ContainerStart(ctx, r.ID, types.ContainerStartOptions{})
	if err != nil {
		return "", err
	}

	return r.ID, nil
}

func DeleteSandbox(cli *client.Client, sandboxID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()

	// TODO: think about graceful shutdown
	return cli.ContainerRemove(ctx, sandboxID, types.ContainerRemoveOptions{Force: true})
}

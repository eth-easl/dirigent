package sandbox

import (
	"context"
	"fmt"
	"github.com/containerd/containerd"
	"github.com/containerd/containerd/cio"
	"github.com/containerd/containerd/oci"
	"github.com/containerd/go-cni"
	"github.com/sirupsen/logrus"
	"log"
	"math/rand"
	"syscall"
)

func GetContainerdClient(containerdSocket string) *containerd.Client {
	client, err := containerd.New(containerdSocket)
	if err != nil {
		log.Fatal("Failed to create a containerd client - ", err)
	}

	return client
}

func GetCNIClient(configFile string) cni.CNI {
	network, err := cni.New(cni.WithConfFile(configFile))
	if err != nil {
		logrus.Fatal("Failed to create a CNI client - ", err)
	}

	return network
}

func FetchImage(ctx context.Context, client *containerd.Client, imageURL string) (containerd.Image, error) {
	image, err := client.Pull(ctx, imageURL, containerd.WithPullUnpack)
	if err != nil {
		return nil, err
	}

	return image, nil
}

func CreateContainer(ctx context.Context, client *containerd.Client, name string, image containerd.Image) (containerd.Container, error) {
	containerName := fmt.Sprintf("%s-%d", name, rand.Int())

	container, err := client.NewContainer(ctx, containerName,
		containerd.WithImage(image),
		containerd.WithNewSnapshot(containerName, image),
		containerd.WithNewSpec(oci.WithImageConfig(image)),
	)

	if err != nil {
		return nil, err
	}

	return container, nil
}

func StartContainer(ctx context.Context, container containerd.Container, network cni.CNI) (containerd.Task, <-chan containerd.ExitStatus, string, string, error) {
	task, err := container.NewTask(ctx, cio.NewCreator(cio.WithStdio))
	if err != nil {
		return nil, nil, "", "", err
	}

	statusChannel, err := task.Wait(ctx)
	if err != nil {
		return nil, nil, "", "", err
	}

	netns := fmt.Sprintf("/proc/%v/ns/net", task.Pid())
	result, err := network.Setup(ctx, container.ID(), netns)
	if err != nil {
		return nil, nil, "", "", err
	}

	ip := result.Interfaces["eth0"].IPConfigs[0].IP.String()
	logrus.Debug("Container ", container.ID(), " has been allocated IP = ", ip)

	err = task.Start(ctx)
	if err != nil {
		return nil, nil, "", "", err
	}

	return task, statusChannel, ip, netns, nil
}

func DeleteContainer(ctx context.Context, network cni.CNI, metadata *Metadata) error {
	if err := network.Remove(ctx, metadata.Container.ID(), metadata.NetNs); err != nil {
		return err
	}

	// non-graceful shutdown
	if err := metadata.Task.Kill(ctx, syscall.SIGKILL); err != nil {
		return err
	}
	status := <-metadata.ExitChannel

	if _, err := metadata.Task.Delete(ctx); err != nil {
		return err
	}

	if err := metadata.Container.Delete(ctx, containerd.WithSnapshotCleanup); err != nil {
		return err
	}

	logrus.Debug("Container ", metadata.Container.ID(), " exited with status ", status)
	return nil
}

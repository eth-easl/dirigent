package common

import (
	"context"
	"fmt"
	"github.com/containerd/containerd"
	"github.com/containerd/containerd/cio"
	"github.com/containerd/containerd/oci"
	"github.com/containerd/containerd/third_party/k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"math/rand"
	"net"
	"syscall"
	"time"
)

func FetchImage(ctx context.Context, client *containerd.Client, imageURL string) containerd.Image {
	image, err := client.Pull(ctx, imageURL, containerd.WithPullUnpack)
	if err != nil {
		logrus.Warn("Failed to fetch container image")
	}

	return image
}

func CreateContainerOld(ctx context.Context, client *containerd.Client, name string, image containerd.Image) containerd.Container {
	container, err := client.NewContainer(ctx, fmt.Sprintf("%s-%d", name, rand.Int()),
		containerd.WithSnapshot(name+"-snapshot"),
		//containerd.WithNewSnapshot(name+"-snapshot", image),
		containerd.WithNewSpec(oci.WithImageConfig(image)),
	)

	if err != nil {
		logrus.Warn("Failed to create a container")
	}

	return container
}

func StartContainer(ctx context.Context, container containerd.Container) (containerd.Task, <-chan containerd.ExitStatus) {
	task, err := container.NewTask(ctx, cio.NewCreator(cio.WithStdio))
	if err != nil {
		logrus.Warn("Failed to create a task from a container")
	}

	statusChannel, err := task.Wait(ctx)
	if err != nil {
		logrus.Warn("Error getting a exit status channel from the container")
	}

	err = task.Start(ctx)
	if err != nil {
		logrus.Warn("Error starting a container")
	}

	return task, statusChannel
}

func DeleteContainer(ctx context.Context, task containerd.Task, exitChannel <-chan containerd.ExitStatus) {
	if err := task.Kill(ctx, syscall.SIGTERM); err != nil {
		logrus.Fatal(err)
	}

	status := <-exitChannel
	code, _, err := status.Result()
	if err != nil {
		logrus.Debug(err)
	}
	logrus.Debug(code)
}

////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////

const ContainerdSocket = "/run/containerd/containerd.sock"

func dialer(ctx context.Context, addr string) (net.Conn, error) {
	return (&net.Dialer{}).DialContext(ctx, "unix", addr)
}

func getDialOpts() []grpc.DialOption {
	return []grpc.DialOption{
		grpc.WithInsecure(),
		grpc.WithContextDialer(dialer),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(1024 * 1024 * 16)),
	}
}

func CreateRuntimeClient() v1alpha2.RuntimeServiceClient {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, ContainerdSocket, getDialOpts()...)
	if err != nil {
		logrus.Fatal(err)
	}

	return v1alpha2.NewRuntimeServiceClient(conn)
}

func CreateImageClient() v1alpha2.ImageServiceClient {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, ContainerdSocket, getDialOpts()...)
	if err != nil {
		logrus.Fatal(err)
	}

	return v1alpha2.NewImageServiceClient(conn)
}

func CreateSandbox(runtimeClient v1alpha2.RuntimeServiceClient) string {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := runtimeClient.RunPodSandbox(ctx, &v1alpha2.RunPodSandboxRequest{
		Config: &v1alpha2.PodSandboxConfig{
			Metadata: &v1alpha2.PodSandboxMetadata{
				Name:      fmt.Sprintf("test-%d", rand.Int()),
				Namespace: "default",
			},
			/*PortMappings: []*v1alpha2.PortMapping{
				{
					Protocol:      v1alpha2.Protocol_TCP,
					ContainerPort: 80,
					HostPort:      10000,
					HostIp:        "",
				},
			},*/
		},
	})

	if err != nil {
		logrus.Warn(err)
	}

	return resp.PodSandboxId
}

func CreateContainer(runtimeClient v1alpha2.RuntimeServiceClient, imageClient v1alpha2.ImageServiceClient, podSandboxID string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	image, err := imageClient.PullImage(ctx, &v1alpha2.PullImageRequest{
		Image: &v1alpha2.ImageSpec{
			Image: "docker.io/cvetkovic/empty_function:latest",
		},
	})
	if err != nil {
		logrus.Fatal(err)
	}

	resp, err := runtimeClient.CreateContainer(ctx, &v1alpha2.CreateContainerRequest{
		//PodSandboxId: podSandboxID,
		Config: &v1alpha2.ContainerConfig{
			Metadata: &v1alpha2.ContainerMetadata{
				Name: "function",
			},
			Image: &v1alpha2.ImageSpec{Image: image.ImageRef},
		},
		SandboxConfig: nil,
	})

	if err != nil {
		logrus.Warn(err)
	}

	return resp.ContainerId
}

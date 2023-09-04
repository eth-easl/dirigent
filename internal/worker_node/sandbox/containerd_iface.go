package sandbox

import (
	"cluster_manager/api/proto"
	"context"
	"fmt"
	"log"
	"math/rand"
	"syscall"
	"time"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/cio"
	"github.com/containerd/containerd/oci"
	"github.com/containerd/go-cni"
	"github.com/sirupsen/logrus"
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

func CreateContainer(ctx context.Context, client *containerd.Client, image containerd.Image) (containerd.Container, error, time.Duration) {
	start := time.Now()

	containerName := fmt.Sprintf("workload-%d", rand.Int())

	container, err := client.NewContainer(ctx, containerName,
		containerd.WithImage(image),
		containerd.WithNewSnapshot(containerName, image),
		containerd.WithNewSpec(oci.WithImageConfig(image)),
	)

	if err != nil {
		return nil, err, time.Since(start)
	}

	return container, nil, time.Since(start)
}

func StartContainer(ctx context.Context, container containerd.Container, network cni.CNI) (containerd.Task, <-chan containerd.ExitStatus, string, string, error, time.Duration, time.Duration) {
	start := time.Now()

	task, err := container.NewTask(ctx, cio.NewCreator())
	if err != nil {
		return nil, nil, "", "", err, 0, 0
	}

	statusChannel, err := task.Wait(ctx)
	if err != nil {
		return nil, nil, "", "", err, 0, 0
	}

	//////////////////////////////////////////
	// CNI
	//////////////////////////////////////////
	cniStart := time.Now()
	netns := fmt.Sprintf("/proc/%v/ns/net", task.Pid())
	result, err := network.Setup(ctx, container.ID(), netns)

	if err != nil {
		return nil, nil, "", "", err, 0, 0
	}

	ip := result.Interfaces["eth0"].IPConfigs[0].IP.String()
	logrus.Debug("Container ", container.ID(), " has been allocated IP = ", ip)

	durationCNI := time.Since(cniStart)
	//////////////////////////////////////////
	//////////////////////////////////////////
	//////////////////////////////////////////

	err = task.Start(ctx)
	if err != nil {
		return nil, nil, "", "", err, 0, 0
	}

	return task, statusChannel, ip, netns, nil, time.Since(start) - durationCNI, durationCNI
}

func ListenOnExitChannel(cpApi proto.CpiInterfaceClient, metadata *Metadata) {
	status := <-metadata.ExitChannel
	logrus.Debug("Container ", metadata.Container.ID(), " exited with status ", status.ExitCode())

	_, _ = metadata.Task.Delete(context.Background(), containerd.WithProcessKill)
	_ = metadata.Container.Delete(context.Background(), containerd.WithSnapshotCleanup)

	switch status.ExitCode() {
	case 137: // SIGKILL signal -- sent by 'Task.Kill' from 'DeleteContainer'
		logrus.Debug("Container killed by SIGKILL.")
		metadata.SignalKillBySystem.Done()
	default: // termination not caused by a signal
		if cpApi == nil {
			return // for tests
		}

		_, err := cpApi.ReportFailure(context.Background(), &proto.Failure{
			Type:        proto.FailureType_CONTAINER_FAILURE,
			ServiceName: metadata.ServiceName,
			ContainerID: metadata.Container.ID(),
		})
		if err != nil {
			logrus.Warn("Failed to report container failure to the control plane for '" + metadata.ServiceName + "'.")
		}
	}
}

func DeleteContainer(ctx context.Context, network cni.CNI, metadata *Metadata) error {
	if err := network.Remove(ctx, metadata.Container.ID(), metadata.NetNs); err != nil {
		return err
	}

	// non-graceful shutdown
	if err := metadata.Task.Kill(ctx, syscall.SIGKILL, containerd.WithKillAll); err != nil {
		return err
	}

	metadata.SignalKillBySystem.Wait()

	return nil
}

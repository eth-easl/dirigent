package containerd

import (
	"cluster_manager/internal/worker_node/managers"
	"cluster_manager/proto"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"syscall"
	"time"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/cio"
	"github.com/containerd/containerd/oci"
	"github.com/containerd/containerd/remotes/docker"
	"github.com/containerd/go-cni"
	"github.com/sirupsen/logrus"
)

const (
	SIGKILL uint32 = 137
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

func FetchImage(ctx context.Context, client *containerd.Client, imageURL string, opts ...containerd.RemoteOpt) (containerd.Image, error) {
	image, err := client.GetImage(ctx, imageURL)
	if err == nil {
		// Image already pulled
		return image, nil
	}
	resolver := docker.NewResolver(docker.ResolverOptions{
		PlainHTTP: true,
	})
	finalOpts := append(opts, containerd.WithPullUnpack, containerd.WithResolver(resolver))
	image, err = client.Pull(ctx, imageURL, finalOpts...)
	if err != nil {
		return nil, err
	}

	return image, nil
}

func composeEnvironmentSetting(cfg *proto.SandboxConfiguration) []string {
	if cfg == nil {
		return []string{}
	}

	return []string{
		fmt.Sprintf("ITERATIONS_MULTIPLIER=%d", cfg.IterationMultiplier),
		fmt.Sprintf("COLD_START_BUSY_LOOP_MS=%d", cfg.ColdStartBusyLoopMs),
	}
}

func generateContainerName() string {
	return fmt.Sprintf("workload-%d", rand.Int())
}

func CreateContainer(ctx context.Context, client *containerd.Client, image containerd.Image, configuration *proto.ServiceInfo, CPUConstraints bool) (containerd.Container, error, time.Duration) {
	start := time.Now()

	options := []oci.SpecOpts{oci.WithImageConfig(image), oci.WithEnv(composeEnvironmentSetting(configuration.SandboxConfiguration))}
	if CPUConstraints {
		options = append(options, oci.WithCPUs(strconv.FormatUint(configuration.RequestedCpu, 10)))
	}
	options = append(options, oci.WithEnv(configuration.EnvironmentVariables))

	containerName := generateContainerName()
	container, err := client.NewContainer(ctx, containerName,
		containerd.WithImage(image),
		containerd.WithNewSnapshot(containerName, image),
		containerd.WithNewSpec(options...),
	)
	if err != nil {
		return nil, err, time.Since(start)
	}

	// For reference see - containerd repo - func WithProcessArgs(config *runtime.ContainerConfig, image *imagespec.ImageConfig) oci.SpecOpts
	if len(configuration.ProgramArguments) > 1 {
		newSpec, err := addProcessArgs(ctx, container, configuration.ProgramArguments)
		if err != nil {
			return nil, err, time.Since(start)
		}

		err = container.Delete(ctx)
		if err != nil {
			return nil, err, time.Since(start)
		}

		containerName = generateContainerName()
		container, err = client.NewContainer(ctx, containerName,
			containerd.WithImage(image),
			containerd.WithNewSnapshot(containerName, image),
			containerd.WithSpec(newSpec),
		)
		if err != nil {
			return nil, err, time.Since(start)
		}
	}

	return container, nil, time.Since(start)
}

func addProcessArgs(ctx context.Context, oldContainer containerd.Container, newArgs []string) (*oci.Spec, error) {
	info, err := oldContainer.Info(ctx)
	if err != nil {
		logrus.Errorf("Failed to extract information from a container spec.")
		return nil, err
	}

	var newSpec oci.Spec
	err = json.Unmarshal(info.Spec.GetValue(), &newSpec)
	if err != nil {
		logrus.Errorf("Failed to unmarshall oci.Spec struct.")
		return nil, err
	}

	newSpec.Process.Args = append(newSpec.Process.Args, newArgs...)
	logrus.Debugf("Edited program to %s", newSpec.Process.Args)

	return &newSpec, nil
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

func WatchExitChannel(cpApi proto.CpiInterfaceClient, metadata *managers.Metadata, extractContainerName func(*managers.Metadata) string) {
	exitCode := <-metadata.ExitStatusChannel
	containerID := extractContainerName(metadata)

	switch exitCode {
	case SIGKILL: // sent by 'Task.Kill' from 'DeleteContainer'
		logrus.Debug("Sandbox '", containerID, "' terminated by the control plane with exit code ", exitCode)
	default: // termination not caused by a signal
		if cpApi == nil {
			// TODO: in tests create fake cpApi
			return // for tests as cpApi is null
		}

		_, err := cpApi.ReportFailure(context.Background(), &proto.Failure{
			Type:        proto.FailureType_SANDBOX_FAILURE,
			ServiceName: metadata.ServiceName,
			SandboxIDs:  []string{containerID},
		})
		if err != nil {
			logrus.Warn("Failed to report container failure to the control plane for '" + metadata.ServiceName + "'.")
		}

		logrus.Debug("Control plane has been notified of failure of sandbox '", containerID, "' (exit code: ", exitCode, ")")
	}
}

func DeleteContainer(ctx context.Context, network cni.CNI, metadata *managers.Metadata) error {
	containerMetadata := (*metadata).RuntimeMetadata.(ContainerdMetadata)

	// TODO: what happens with CNI and container metadata if the container fails -- memory leak
	if err := network.Remove(ctx, containerMetadata.Container.ID(), metadata.NetNs); err != nil {
		return err
	}

	// non-graceful shutdown
	if err := containerMetadata.Task.Kill(ctx, syscall.SIGKILL, containerd.WithKillAll); err != nil {
		return err
	}

	// containerd detection failure is useless as wait on exit channel needs to be called after KILL,
	// otherwise it is useless
	exitStatus, err := containerMetadata.Task.Delete(ctx, containerd.WithProcessKill)
	if err != nil {
		return err
	}
	logrus.Debug("Sandbox terminated by SIGKILL (status code :", exitStatus.ExitCode(), ")")

	if err := containerMetadata.Container.Delete(ctx, containerd.WithSnapshotCleanup); err != nil {
		return err
	}

	return nil
}

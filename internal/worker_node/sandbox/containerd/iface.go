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
	"cluster_manager/api/proto"
	"cluster_manager/internal/worker_node/managers"
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

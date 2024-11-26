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
	"cluster_manager/internal/worker_node/sandbox"
	"cluster_manager/pkg/config"
	"context"
	"github.com/containerd/containerd"
	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/go-cni"
	"github.com/coreos/go-iptables/iptables"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/emptypb"
	"time"
)

type ContainerdRuntime struct {
	sandbox.RuntimeInterface

	cpApi proto.CpiInterfaceClient

	ContainerdClient *containerd.Client
	CNIClient        cni.CNI
	IPT              *iptables.IPTables

	ImageManager   *ImageManager
	SandboxManager *managers.SandboxManager
	ProcessMonitor *managers.ProcessMonitor
}

type ContainerdMetadata struct {
	managers.RuntimeMetadata

	Task      containerd.Task
	Container containerd.Container
}

func NewContainerdRuntime(cpApi proto.CpiInterfaceClient, config config.WorkerNodeConfig, sandboxManager *managers.SandboxManager) *ContainerdRuntime {
	containerdClient := GetContainerdClient(config.CRIPath)

	imageManager := NewContainerdImageManager()
	cniClient := GetCNIClient(config.CNIConfigPath)
	ipt, err := managers.NewIptablesUtil()

	if err != nil {
		logrus.Fatal("Error while accessing iptables - ", err)
	}

	if config.PrefetchImage {
		ctx := namespaces.WithNamespace(context.Background(), "default")

		_, err, _ = imageManager.GetImage(ctx, containerdClient, "docker.io/cvetkovic/dirigent_empty_function:latest")
		if err != nil {
			logrus.Errorf("Failed to prefetch the image")
		}

		_, err, _ = imageManager.GetImage(ctx, containerdClient, "docker.io/cvetkovic/dirigent_trace_function:latest")
		if err != nil {
			logrus.Errorf("Failed to prefetch the image")
		}
	}

	return &ContainerdRuntime{
		cpApi: cpApi,

		ContainerdClient: containerdClient,
		CNIClient:        cniClient,
		IPT:              ipt,

		ImageManager:   imageManager,
		SandboxManager: sandboxManager,
		ProcessMonitor: managers.NewProcessMonitor(),
	}
}

func (cr *ContainerdRuntime) CreateSandbox(grpcCtx context.Context, in *proto.ServiceInfo) (*proto.SandboxCreationStatus, error) {
	logrus.Debug("Create sandbox for service = '", in.Name, "'")

	start := time.Now()

	ctx := namespaces.WithNamespace(grpcCtx, "cm")
	image, err, durationFetch := cr.ImageManager.GetImage(ctx, cr.ContainerdClient, in.Image)

	if err != nil {
		logrus.Warn("Failed fetching image - ", err)
		return &proto.SandboxCreationStatus{Success: false}, err
	}

	container, err, durationContainerCreation := CreateContainer(ctx, cr.ContainerdClient, image)
	if err != nil {
		logrus.Warn("Failed creating a container - ", err)
		return &proto.SandboxCreationStatus{Success: false}, err
	}

	task, _, ip, netNs, err, durationContainerStart, durationCNI := StartContainer(ctx, container, cr.CNIClient)
	if err != nil {
		logrus.Warn("Failed starting a container - ", err)
		return &proto.SandboxCreationStatus{Success: false}, err
	}

	startConfigureMonitoring := time.Now()
	metadata := &managers.Metadata{
		ServiceName: in.Name,

		RuntimeMetadata: ContainerdMetadata{
			Task:      task,
			Container: container,
		},

		HostPort:  AssignRandomPort(),
		IP:        ip,
		GuestPort: int(in.PortForwarding.GuestPort),
		NetNs:     netNs,

		ExitStatusChannel: make(chan uint32),
	}

	cr.ProcessMonitor.AddChannel(task.Pid(), metadata.ExitStatusChannel)
	cr.SandboxManager.AddSandbox(container.ID(), metadata)
	configureMonitoringDuration := time.Since(startConfigureMonitoring)

	logrus.Debug("Sandbox creation took ", time.Since(start).Microseconds(), " μs (", container.ID(), ")")

	startIptables := time.Now()

	managers.AddRules(cr.IPT, metadata.HostPort, metadata.IP, metadata.GuestPort)

	durationIptables := time.Since(startIptables)

	logrus.Debug("IP tables configuration (add rule(s)) took ", durationIptables.Microseconds(), " μs")

	in.PortForwarding.HostPort = int32(metadata.HostPort)

	go WatchExitChannel(cr.cpApi, metadata, func(metadata *managers.Metadata) string {
		return metadata.RuntimeMetadata.(ContainerdMetadata).Container.ID()
	})

	return &proto.SandboxCreationStatus{
		Success:      true,
		ID:           container.ID(),
		PortMappings: in.PortForwarding,
		LatencyBreakdown: &proto.SandboxCreationBreakdown{
			Total:               durationpb.New(time.Since(start)),
			ImageFetch:          durationpb.New(durationFetch),
			SandboxCreate:       durationpb.New(durationContainerCreation),
			NetworkSetup:        durationpb.New(durationCNI),
			SandboxStart:        durationpb.New(durationContainerStart),
			Iptables:            durationpb.New(durationIptables),
			ReadinessProbing:    durationpb.New(0),
			SnapshotCreation:    durationpb.New(0),
			ConfigureMonitoring: durationpb.New(configureMonitoringDuration),
			FindSnapshot:        durationpb.New(0),
		},
	}, nil
}

func (cr *ContainerdRuntime) DeleteSandbox(grpcCtx context.Context, in *proto.SandboxID) (*proto.ActionStatus, error) {
	logrus.Debug("RemoveKey sandbox with ID = '", in.ID, "'")

	ctx := namespaces.WithNamespace(grpcCtx, "cm")
	metadata := cr.SandboxManager.DeleteSandbox(in.ID)

	if metadata == nil {
		logrus.Warn("Tried to delete non-existing sandbox ", in.ID)
		return &proto.ActionStatus{Success: false}, nil
	}

	start := time.Now()

	managers.DeleteRules(cr.IPT, metadata.HostPort, metadata.IP, metadata.GuestPort)
	UnassignPort(metadata.HostPort)
	logrus.Debug("IP tables configuration (remove rule(s)) took ", time.Since(start).Microseconds(), " μs")

	start = time.Now()
	err := DeleteContainer(ctx, cr.CNIClient, metadata)

	if err != nil {
		logrus.Warn(err)
		return &proto.ActionStatus{Success: false}, err
	}

	logrus.Debug("Sandbox deletion took ", time.Since(start).Microseconds(), " μs")

	return &proto.ActionStatus{Success: true}, nil
}

func (cr *ContainerdRuntime) ListEndpoints(_ context.Context, _ *emptypb.Empty) (*proto.EndpointsList, error) {
	return cr.SandboxManager.ListEndpoints()
}

func (cr *ContainerdRuntime) ValidateHostConfig() bool {
	return true
}

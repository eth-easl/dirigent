package fcctr

import (
	"cluster_manager/internal/worker_node/managers"
	ctrmanagers "cluster_manager/internal/worker_node/sandbox/containerd"
	"cluster_manager/internal/worker_node/sandbox/firecracker"
	"cluster_manager/proto"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/cio"
	"github.com/containerd/containerd/mount"
	"github.com/containerd/containerd/oci"
	"github.com/containerd/containerd/snapshots"
	"github.com/coreos/go-iptables/iptables"
	fcctr "github.com/firecracker-microvm/firecracker-containerd/firecracker-control/client"
	ctrproto "github.com/firecracker-microvm/firecracker-containerd/proto"
	"github.com/firecracker-microvm/firecracker-containerd/runtime/firecrackeroci"
	"github.com/opencontainers/image-spec/identity"
	"github.com/sirupsen/logrus"
)

const (
	baseKernelArgs    = "ro noapic reboot=k panic=1 pci=off nomodule systemd.log_color=false systemd.unit=firecracker.target init=/sbin/overlay-init tsc=reliable quiet i8042.noaux ipv6.disable=1"
	noDebugKernelArgs = baseKernelArgs + " i8042.nokbd 8250.nr_uarts=0"
	debugKernelArgs   = baseKernelArgs + " console=ttyS0 systemd.journald.forward_to_console"
)

type SandboxControlStructure struct {
	managers.RuntimeMetadata

	VMConfig             *ctrproto.CreateVMRequest
	Image                containerd.Image
	NetworkConfiguration *firecracker.NetworkConfig
	SandboxConfiguration *proto.SandboxConfiguration

	SandboxID string

	HostPort  int
	GuestPort int
}

func CreateNetworkConfig(networkManager *firecracker.NetworkPoolManager) (*firecracker.NetworkConfig, time.Duration, error) {
	start := time.Now()

	config := networkManager.GetOneConfig()
	if config == nil {
		return nil, time.Since(start), errors.New("error getting a network interface")
	}

	return config, time.Since(start), nil
}

func GetNetNs(scs *SandboxControlStructure) string {
	return fmt.Sprintf("/var/run/netns/%s", scs.NetworkConfiguration.NetNS)
}

func CreateVM(ctx context.Context, fcctrClient *fcctr.Client, scs *SandboxControlStructure, snapshot *firecracker.SnapshotMetadata, vmDebugMode bool) (time.Duration, error) {
	start := time.Now()

	kernelArgs := noDebugKernelArgs
	if vmDebugMode {
		kernelArgs = debugKernelArgs
	}
	opts := &ctrproto.CreateVMRequest{
		VMID:           scs.SandboxID,
		TimeoutSeconds: 100,
		KernelArgs:     kernelArgs,
		MachineCfg: &ctrproto.FirecrackerMachineConfiguration{
			MemSizeMib: 2048,
			VcpuCount:  1,
			HtEnabled:  false,
		},
		NetworkInterfaces: []*ctrproto.FirecrackerNetworkInterface{{
			StaticConfig: &ctrproto.StaticNetworkConfiguration{
				MacAddress:  scs.NetworkConfiguration.TapMAC,
				HostDevName: scs.NetworkConfiguration.TapDeviceName,
				IPConfig: &ctrproto.IPConfiguration{
					PrimaryAddr: fmt.Sprintf("%s/30", scs.NetworkConfiguration.TapInternalIP),
					GatewayAddr: scs.NetworkConfiguration.TapExternalIP,
				},
			},
		}},
		NetNS: GetNetNs(scs),
	}
	if snapshot != nil {
		opts.LoadSnapshot = true
		opts.MemFilePath = snapshot.MemoryPath
		opts.SnapshotPath = snapshot.SnapshotPath
		opts.ContainerSnapshotPath = snapshot.ContainerSnapshotPath
	}
	_, err := fcctrClient.CreateVM(ctx, opts)

	return time.Since(start), err
}

func CreateContainer(ctx context.Context, ctrClient *containerd.Client, scs *SandboxControlStructure, verbosity string) (containerd.Container, time.Duration, error) {
	start := time.Now()

	container, err := ctrClient.NewContainer(
		ctx,
		scs.SandboxID,
		containerd.WithSnapshotter("devmapper"),
		containerd.WithNewSnapshot(scs.SandboxID, scs.Image),
		containerd.WithNewSpec(
			oci.WithImageConfig(scs.Image),
			firecrackeroci.WithVMID(scs.SandboxID),
			firecrackeroci.WithVMNetwork,
			oci.WithEnv(composeEnvironmentSetting(scs.SandboxConfiguration, verbosity)),
		),
		containerd.WithRuntime("aws.firecracker", nil),
	)

	return container, time.Since(start), err
}

func StartContainer(ctx context.Context, container containerd.Container) (containerd.Task, time.Duration, error) {
	start := time.Now()
	stdWrite := logrus.StandardLogger().Writer()
	task, err := container.NewTask(ctx, cio.NewCreator(cio.WithStreams(os.Stdin, stdWrite, stdWrite)))
	if err != nil {
		return nil, time.Since(start), err
	}
	_, err = task.Wait(ctx)
	if err != nil {
		return nil, time.Since(start), err
	}
	err = task.Start(ctx)
	if err != nil {
		return nil, time.Since(start), err
	}
	return task, time.Since(start), nil
}

func ConfigureMonitoring(sandboxManager *managers.SandboxManager, cpApi proto.CpiInterfaceClient, serviceInfo *proto.ServiceInfo, scs *SandboxControlStructure) (*managers.Metadata, time.Duration) {
	start := time.Now()

	metadata := &managers.Metadata{
		ServiceName:     serviceInfo.Name,
		RuntimeMetadata: *scs,

		IP:        scs.NetworkConfiguration.ExposedIP,
		HostPort:  ctrmanagers.AssignRandomPort(),
		GuestPort: int(serviceInfo.PortForwarding.GuestPort),
		NetNs:     GetNetNs(scs),

		ExitStatusChannel: make(chan uint32),
	}
	sandboxManager.AddSandbox(scs.SandboxID, metadata)
	serviceInfo.PortForwarding.HostPort = int32(metadata.HostPort)

	go ctrmanagers.WatchExitChannel(cpApi, metadata, func(metadata *managers.Metadata) string {
		return metadata.RuntimeMetadata.(SandboxControlStructure).SandboxID
	})

	return metadata, time.Since(start)
}

func SetupIptables(ipt *iptables.IPTables, metadata *managers.Metadata) time.Duration {
	startIptables := time.Now()
	managers.AddRules(ipt, metadata.HostPort, metadata.IP, metadata.GuestPort)
	durationIptables := time.Since(startIptables)
	logrus.Debugf("IP tables configuration (add rule(s)) took %d μs", durationIptables.Microseconds())
	return durationIptables
}

func CreateSnapshot(ctx context.Context, ctrClient *containerd.Client, fcctrClient *fcctr.Client, snapshotManager *firecracker.SnapshotManager, scs *SandboxControlStructure) (time.Duration, error) {
	logrus.Debugf("Creating snapshot for sandbox %s", scs.SandboxID)
	start := time.Now()

	_, err := fcctrClient.PauseVM(ctx, &ctrproto.PauseVMRequest{VMID: scs.SandboxID})
	if err != nil {
		logrus.WithError(err).Errorf("Failed to pause virtual machine %s", scs.SandboxID)
		return time.Since(start), err
	}

	snapshotDir := filepath.Join(os.TempDir(), "snapshots")
	err = os.MkdirAll(snapshotDir, os.ModePerm)
	if err != nil {
		logrus.WithError(err).Error("Error creating a directory for snapshot storage")
		return time.Since(start), err
	}

	diffIDs, err := scs.Image.RootFS(ctx)
	if err != nil {
		logrus.WithError(err).Errorf("Failed to retrieve rootfs of image %s", scs.Image.Name())
		return time.Since(start), err
	}
	snapshotId := identity.ChainID(diffIDs).String()
	activeSnapshotId := fmt.Sprintf("%s-active", snapshotId)
	snapshotter := ctrClient.SnapshotService("devmapper")
	stat, _ := snapshotter.Stat(ctx, activeSnapshotId)
	var mounts []mount.Mount
	if stat.Kind != snapshots.KindActive {
		logrus.Debugf("Preparing snapshot %s", snapshotId)
		mounts, err = snapshotter.Prepare(ctx, activeSnapshotId, snapshotId)
	} else {
		mounts, err = snapshotter.Mounts(ctx, activeSnapshotId)
	}
	if err != nil {
		logrus.WithError(err).Errorf("Failed to prepare snapshot %s", snapshotId)
		return time.Since(start), err
	}

	snapshotPaths := &firecracker.SnapshotMetadata{
		MemoryPath:            filepath.Join(snapshotDir, fmt.Sprintf("memory-firecracker-containerd-%s", scs.SandboxID)),
		SnapshotPath:          filepath.Join(snapshotDir, fmt.Sprintf("snapshot-firecracker-containerd-%s", scs.SandboxID)),
		ContainerSnapshotPath: mounts[0].Source,
	}
	_, err = fcctrClient.CreateSnapshot(ctx, &ctrproto.CreateSnapshotRequest{
		VMID:         scs.SandboxID,
		MemFilePath:  snapshotPaths.MemoryPath,
		SnapshotPath: snapshotPaths.SnapshotPath,
	})
	if err != nil {
		logrus.WithError(err).Errorf("Failed to create a snapshot out of virtual machine %s", scs.SandboxID)
		return time.Since(start), err
	}

	snapshotManager.AddSnapshot(scs.Image.Name(), snapshotPaths)
	logrus.Debugf("Snapshot for sandbox %s created", scs.SandboxID)

	_, err = fcctrClient.ResumeVM(ctx, &ctrproto.ResumeVMRequest{VMID: scs.SandboxID})
	if err != nil {
		logrus.WithError(err).Errorf("Failed to resume virtual machine %s", scs.SandboxID)
		return time.Since(start), err
	}

	return time.Since(start), nil
}

func FindSnapshot(snapshotManager *firecracker.SnapshotManager, imageName string) (*firecracker.SnapshotMetadata, time.Duration) {
	start := time.Now()
	snapshot, _ := snapshotManager.FindSnapshot(imageName)
	if snapshot != nil {
		logrus.Infof("Snapshot for image %s found", imageName)
	}
	duration := time.Since(start)
	return snapshot, duration
}

func StopVM(ctx context.Context, fcctrClient *fcctr.Client, scs *SandboxControlStructure) error {
	_, err := fcctrClient.StopVM(ctx, &ctrproto.StopVMRequest{VMID: scs.SandboxID})
	return err
}

func composeEnvironmentSetting(cfg *proto.SandboxConfiguration, verbosity string) []string {
	if cfg == nil {
		return []string{}
	}

	return []string{
		fmt.Sprintf("ITERATIONS_MULTIPLIER=%d", cfg.IterationMultiplier),
		fmt.Sprintf("COLD_START_BUSY_LOOP_MS=%d", cfg.ColdStartBusyLoopMs),
		fmt.Sprintf("LOG_LEVEL=%s", verbosity),
	}
}

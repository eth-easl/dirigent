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

package firecracker

import (
	"cluster_manager/api/proto"
	"cluster_manager/internal/worker_node/managers"
	"cluster_manager/internal/worker_node/sandbox"
	"cluster_manager/internal/worker_node/sandbox/containerd"
	"context"
	"fmt"
	"github.com/coreos/go-iptables/iptables"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/emptypb"
	"os"
	"path/filepath"
	"time"
)

type Runtime struct {
	sandbox.RuntimeInterface

	cpApi       proto.CpiInterfaceClient
	idGenerator *managers.ThreadSafeRandomGenerator

	VMDebugMode  bool
	UseSnapshots bool

	KernelPath     string
	FileSystemPath string
	NetworkManager *NetworkPoolManager

	SandboxManager  *managers.SandboxManager
	ProcessMonitor  *managers.ProcessMonitor
	SnapshotManager *SnapshotManager
	IPT             *iptables.IPTables
}

type FirecrackerMetadata struct {
	managers.RuntimeMetadata

	VMCS *VMControlStructure
}

func NewFirecrackerRuntime(cpApi proto.CpiInterfaceClient, sandboxManager *managers.SandboxManager,
	kernelPath string, fileSystemPath string, internalIPPrefix string, externalIPPrefix string,
	vmDebugMode bool, useSnapshots bool, networkPoolSize int) *Runtime {

	DeleteAllSnapshots()
	err := DeleteUnusedNetworkDevices()
	if err != nil {
		logrus.Error("Failed to remove some or all network devices.")
	}

	ipt, err := managers.NewIptablesUtil()
	if err != nil {
		logrus.Fatal("Failed to start iptables utility. Terminating worker daemon.")
	}

	return &Runtime{
		cpApi:       cpApi,
		idGenerator: managers.NewThreadSafeRandomGenerator(),

		VMDebugMode:  vmDebugMode,
		UseSnapshots: useSnapshots,

		KernelPath:     kernelPath,
		FileSystemPath: fileSystemPath,
		NetworkManager: NewNetworkPoolManager(internalIPPrefix, externalIPPrefix, networkPoolSize),

		SandboxManager:  sandboxManager,
		ProcessMonitor:  managers.NewProcessMonitor(),
		SnapshotManager: NewFirecrackerSnapshotManager(),
		IPT:             ipt,
	}
}

func (fcr *Runtime) createVMCS() *VMControlStructure {
	return &VMControlStructure{
		Context: context.Background(),

		KernelPath:     fcr.KernelPath,
		FileSystemPath: fcr.FileSystemPath,

		SandboxID: fmt.Sprintf("firecracker-%d", fcr.idGenerator.Int()),
	}
}

func createMetadata(in *proto.ServiceInfo, vmcs *VMControlStructure) *managers.Metadata {
	return &managers.Metadata{
		ServiceName: in.Name,

		RuntimeMetadata: FirecrackerMetadata{
			VMCS: vmcs,
		},

		IP:        vmcs.NetworkConfiguration.TapInternalIP,
		GuestPort: int(in.PortForwarding.GuestPort),

		ExitStatusChannel: make(chan uint32),
	}
}

func (fcr *Runtime) CreateSandbox(ctx context.Context, in *proto.ServiceInfo) (*proto.SandboxCreationStatus, error) {
	start := time.Now()

	vmcs := fcr.createVMCS()

	startFindSnapshot := time.Now()
	snapshot, _ := fcr.SnapshotManager.FindSnapshot(in.Image)
	if snapshot != nil {
		logrus.Infof("Snapshot found for image %s", in.Image)
	}
	findSnapshotDuration := time.Since(startFindSnapshot)

	err, netCreateDuration, vmCreateDuration, vmStartDuration := StartFirecrackerVM(fcr.NetworkManager, vmcs, fcr.VMDebugMode, snapshot)
	if err != nil {
		// resource deallocation already done in StartFirecrackerVM
		return &proto.SandboxCreationStatus{Success: false}, err
	}

	startConfigureMonitoring := time.Now()
	metadata := createMetadata(in, vmcs)
	metadata.HostPort = containerd.AssignRandomPort()
	metadata.IP = vmcs.NetworkConfiguration.ExposedIP

	// VM process monitoring
	vmPID, err := vmcs.VM.PID()
	if err != nil {
		fcr.NetworkManager.GiveUpNetwork(vmcs.NetworkConfiguration)
		containerd.UnassignPort(metadata.HostPort)

		logrus.Debugf("Failed to get PID of the virtual machine - %v", err)
		return &proto.SandboxCreationStatus{Success: false}, err
	}

	fcr.SandboxManager.AddSandbox(vmcs.SandboxID, metadata)
	fcr.ProcessMonitor.AddChannel(uint32(vmPID), metadata.ExitStatusChannel)
	configureMonitoringDuration := time.Since(startConfigureMonitoring)

	// port forwarding
	iptablesStart := time.Now()
	managers.AddRules(fcr.IPT, metadata.HostPort, metadata.IP, metadata.GuestPort)
	iptablesDuration := time.Since(iptablesStart)

	go containerd.WatchExitChannel(fcr.cpApi, metadata, func(metadata *managers.Metadata) string {
		return metadata.RuntimeMetadata.(FirecrackerMetadata).VMCS.SandboxID
	})

	in.PortForwarding.HostPort = int32(metadata.HostPort)

	logrus.Debug("Worker node part: ", time.Since(start).Milliseconds(), " ms")

	// blocking call
	timeToPass, passed := managers.SendReadinessProbe(fmt.Sprintf("localhost:%d", metadata.HostPort))

	// create a snapshot for the service if it does not exist
	startSnapshotCreation := time.Now()
	if passed && fcr.UseSnapshots && !fcr.SnapshotManager.Exists(in.Image) {
		ok, paths := CreateVMSnapshot(ctx, vmcs)

		if !ok {
			logrus.Warn("Due to failure, bypassing snapshot creation for image ", in.Image)
		} else {
			fcr.SnapshotManager.AddSnapshot(in.Image, paths)

			logrus.Debug("Snapshot successfully created for image ", in.Image)
		}

		err = vmcs.VM.ResumeVM(ctx)
		if err != nil {
			logrus.Errorf("Failed to create a sandbox %s due to error resuming virtual machine.", vmcs.SandboxID)
			passed = false // let the following code below take care of deletion
		}
	}
	snapshotCreationDuration := time.Since(startSnapshotCreation)

	if passed {
		return &proto.SandboxCreationStatus{
			Success:      true,
			ID:           vmcs.SandboxID,
			PortMappings: in.PortForwarding,
			LatencyBreakdown: &proto.SandboxCreationBreakdown{
				Total:               durationpb.New(time.Since(start)),
				ImageFetch:          durationpb.New(0),
				SandboxCreate:       durationpb.New(vmCreateDuration),
				NetworkSetup:        durationpb.New(netCreateDuration),
				SandboxStart:        durationpb.New(vmStartDuration),
				ReadinessProbing:    durationpb.New(timeToPass),
				Iptables:            durationpb.New(iptablesDuration),
				SnapshotCreation:    durationpb.New(snapshotCreationDuration),
				ConfigureMonitoring: durationpb.New(configureMonitoringDuration),
				FindSnapshot:        durationpb.New(findSnapshotDuration),
			},
		}, nil
	} else {
		_, _ = fcr.DeleteSandbox(context.Background(), &proto.SandboxID{ID: vmcs.SandboxID})

		return &proto.SandboxCreationStatus{
			Success: false,
			ID:      vmcs.SandboxID,
		}, nil
	}
}

func CreateVMSnapshot(ctx context.Context, vmcs *VMControlStructure) (bool, *SnapshotMetadata) {
	err := vmcs.VM.PauseVM(ctx)
	if err != nil {
		logrus.Error("Error pausing virtual machine ", vmcs.SandboxID)
		return false, nil
	}

	snapshotDir := filepath.Join(os.TempDir(), "snapshots")

	err = os.MkdirAll(snapshotDir, os.ModePerm)
	if err != nil {
		logrus.Error("Error creating a directory for snapshot storage")
		return false, nil
	}

	snapshotPaths := &SnapshotMetadata{
		MemoryPath:   filepath.Join(snapshotDir, fmt.Sprintf("memory-firecracker-%s", vmcs.SandboxID)),
		SnapshotPath: filepath.Join(snapshotDir, fmt.Sprintf("snapshot-firecracker-%s", vmcs.SandboxID)),
	}

	err = vmcs.VM.CreateSnapshot(ctx, snapshotPaths.MemoryPath, snapshotPaths.SnapshotPath)
	if err != nil {
		logrus.Error("Error creating a snapshot out of virtual machine ", vmcs.SandboxID)
		return false, nil
	}

	return true, snapshotPaths
}

func (fcr *Runtime) DeleteSandbox(_ context.Context, in *proto.SandboxID) (*proto.ActionStatus, error) {
	metadata := fcr.SandboxManager.DeleteSandbox(in.ID)
	if metadata == nil {
		logrus.Errorf("Error while deleting sandbox from the manager. Invalid name %s.", in.ID)
		return &proto.ActionStatus{Success: false}, nil
	}

	start := time.Now()

	sandboxMetadata := metadata.RuntimeMetadata.(FirecrackerMetadata)

	// delete networking rules
	managers.DeleteRules(fcr.IPT, metadata.HostPort, metadata.IP, metadata.GuestPort)
	containerd.UnassignPort(metadata.HostPort)
	logrus.Debug("IP tables configuration (remove rule(s)) took ", time.Since(start).Microseconds(), " μs")

	// destroy the VM
	err := StopFirecrackerVM(sandboxMetadata.VMCS)
	if err != nil {
		logrus.Errorf("Error deleting a sandbox - %v", err)
	}

	// delete the network associated with the VM
	fcr.NetworkManager.GiveUpNetwork(sandboxMetadata.VMCS.NetworkConfiguration)
	// delete Firecracker logs
	deleteLogs(sandboxMetadata.VMCS)

	logrus.Debug("Sandbox deletion took ", time.Since(start).Microseconds(), " μs")
	return &proto.ActionStatus{Success: true}, nil
}

func (fcr *Runtime) ListEndpoints(_ context.Context, _ *emptypb.Empty) (*proto.EndpointsList, error) {
	return fcr.SandboxManager.ListEndpoints()
}

func (fcr *Runtime) ValidateHostConfig() bool {
	// Check for KVM access
	err := unix.Access("/dev/kvm", unix.W_OK)
	if err != nil {
		logrus.Errorf("KVM access denied - %v", err)
		return false
	}

	return true
}

func DeleteAllSnapshots() {
	// don't handle error because the directory may not exist
	_ = os.RemoveAll(filepath.Join(os.TempDir(), "snapshots"))
}

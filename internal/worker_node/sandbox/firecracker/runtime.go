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
	"math/rand"
	"os"
	"path/filepath"
	"time"
)

type Runtime struct {
	sandbox.RuntimeInterface

	cpApi proto.CpiInterfaceClient

	VMDebugMode  bool
	UseSnapshots bool

	KernelPath     string
	FileSystemPath string
	IpManager      *IPManager

	SandboxManager  *managers.SandboxManager
	ProcessMonitor  *managers.ProcessMonitor
	SnapshotManager *SnapshotManager
	IPT             *iptables.IPTables
}

type FirecrackerMetadata struct {
	managers.RuntimeMetadata

	VMCS *VMControlStructure
}

func NewFirecrackerRuntime(cpApi proto.CpiInterfaceClient, sandboxManager *managers.SandboxManager, kernelPath string, fileSystemPath string, ipPrefix string, vmDebugMode bool, useSnapshots bool) *Runtime {
	_ = DeleteFirecrackerTAPDevices()
	ipt, _ := managers.NewIptablesUtil()

	return &Runtime{
		cpApi: cpApi,

		VMDebugMode:  vmDebugMode,
		UseSnapshots: useSnapshots,

		KernelPath:     kernelPath,
		FileSystemPath: fileSystemPath,
		IpManager:      NewIPManager(ipPrefix),

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
		IpManager:      fcr.IpManager,

		SandboxID: fmt.Sprintf("firecracker-%d", rand.Int()),
	}
}

func createMetadata(in *proto.ServiceInfo, vmcs *VMControlStructure) *managers.Metadata {
	return &managers.Metadata{
		ServiceName: in.Name,

		RuntimeMetadata: FirecrackerMetadata{
			VMCS: vmcs,
		},

		HostPort:  containerd.AssignRandomPort(),
		IP:        vmcs.tapLink.VmIP,
		GuestPort: int(in.PortForwarding.GuestPort),

		ExitStatusChannel: make(chan uint32),
	}
}

func (fcr *Runtime) CreateSandbox(ctx context.Context, in *proto.ServiceInfo) (*proto.SandboxCreationStatus, error) {
	start := time.Now()

	vmcs := fcr.createVMCS()

	snapshot, _ := fcr.SnapshotManager.FindSnapshot(in.Name)

	err, tapCreation, vmCreate, vmStart := StartFirecrackerVM(vmcs, fcr.VMDebugMode, snapshot)
	if err != nil {
		// TODO: deallocate resources
		return &proto.SandboxCreationStatus{Success: false}, err
	}

	metadata := createMetadata(in, vmcs)

	// VM process monitoring
	vmPID, err := vmcs.vm.PID()
	if err != nil {
		// TODO: deallocate resources
		logrus.Debug(err)
		return &proto.SandboxCreationStatus{Success: false}, err
	}

	fcr.SandboxManager.AddSandbox(vmcs.SandboxID, metadata)
	fcr.ProcessMonitor.AddChannel(uint32(vmPID), metadata.ExitStatusChannel)

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
	if fcr.UseSnapshots && !fcr.SnapshotManager.Exists(in.Name) {
		ok, paths := createVMSnapshot(ctx, vmcs)
		if !ok {
			logrus.Warn("Due to failure, bypassing snapshot creation for service ", in.Name)
		} else {
			fcr.SnapshotManager.AddSnapshot(in.Name, paths)

			err = vmcs.vm.ResumeVM(ctx)
			if err != nil {
				logrus.Errorf("Error creating a sandbox %s due to error resuming virtual machine.", vmcs.SandboxID)

				return &proto.SandboxCreationStatus{
					Success: false,
					ID:      vmcs.SandboxID,
				}, nil
			}

			logrus.Debug("Snapshot successfully created for service ", in.Name)
		}
	}

	if passed {
		return &proto.SandboxCreationStatus{
			Success:      true,
			ID:           vmcs.SandboxID,
			PortMappings: in.PortForwarding,
			LatencyBreakdown: &proto.SandboxCreationBreakdown{
				Total:            durationpb.New(time.Since(start)),
				ImageFetch:       durationpb.New(0),
				SandboxCreate:    durationpb.New(vmCreate),
				NetworkSetup:     durationpb.New(tapCreation),
				SandboxStart:     durationpb.New(vmStart),
				ReadinessProbing: durationpb.New(timeToPass),
				Iptables:         durationpb.New(iptablesDuration),
			},
		}, nil
	} else {
		// TODO: deallocate resources
		return &proto.SandboxCreationStatus{
			Success: false,
			ID:      vmcs.SandboxID,
		}, nil
	}
}

func createVMSnapshot(ctx context.Context, vmcs *VMControlStructure) (bool, *SnapshotMetadata) {
	err := vmcs.vm.PauseVM(ctx)
	if err != nil {
		logrus.Error("Error pausing virtual machine ", vmcs.SandboxID)
		return false, nil
	}

	tmpDir, err := os.MkdirTemp(os.TempDir(), "firecracker")
	if err != nil {
		logrus.Error("Error creating a directory for snapshot storage")
		return false, nil
	}

	snapshotPaths := &SnapshotMetadata{
		MemoryPath:   filepath.Join(tmpDir, "memory"),
		SnapshotPath: filepath.Join(tmpDir, "snapshot"),
	}

	err = vmcs.vm.CreateSnapshot(ctx, snapshotPaths.MemoryPath, snapshotPaths.SnapshotPath)
	if err != nil {
		logrus.Error("Error creating a snapshot out of virtual machine ", vmcs.SandboxID)
		return false, nil
	}

	return true, snapshotPaths
}

func (fcr *Runtime) DeleteSandbox(_ context.Context, in *proto.SandboxID) (*proto.ActionStatus, error) {
	metadata := fcr.SandboxManager.DeleteSandbox(in.ID)
	if metadata == nil {
		logrus.Warn("Tried to delete non-existing sandbox ", in.ID)
		return &proto.ActionStatus{Success: false}, nil
	}

	start := time.Now()

	managers.DeleteRules(fcr.IPT, metadata.HostPort, metadata.IP, metadata.GuestPort)
	containerd.UnassignPort(metadata.HostPort)
	logrus.Debug("IP tables configuration (remove rule(s)) took ", time.Since(start).Microseconds(), " μs")

	start = time.Now()

	err := StopFirecrackerVM(metadata.RuntimeMetadata.(FirecrackerMetadata).VMCS)
	if err != nil {
		logrus.Error(err)
		return &proto.ActionStatus{Success: false}, err
	}

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
		logrus.Error("KVM access denied - ", err)
		return false
	}

	return true
}

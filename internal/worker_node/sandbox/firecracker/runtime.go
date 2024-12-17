package firecracker

import (
	"cluster_manager/internal/worker_node/managers"
	"cluster_manager/internal/worker_node/sandbox"
	"cluster_manager/internal/worker_node/sandbox/containerd"
	"cluster_manager/pkg/config"
	"cluster_manager/proto"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/coreos/go-iptables/iptables"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/emptypb"
)

type Runtime struct {
	sandbox.RuntimeInterface

	cpApi       proto.CpiInterfaceClient
	idGenerator *managers.ThreadSafeRandomGenerator
	config      *config.FirecrackerConfig

	NetworkManager *NetworkPoolManager

	SandboxManager  *managers.SandboxManager
	ProcessMonitor  *managers.ProcessMonitor
	SnapshotManager *SnapshotManager
	IPT             *iptables.IPTables
}

type Metadata struct {
	managers.RuntimeMetadata

	VMCS *VMControlStructure
}

func NewFirecrackerRuntime(cpApi proto.CpiInterfaceClient, sandboxManager *managers.SandboxManager, config *config.FirecrackerConfig) *Runtime {
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

		config: config,

		SandboxManager:  sandboxManager,
		ProcessMonitor:  managers.NewProcessMonitor(),
		SnapshotManager: NewFirecrackerSnapshotManager(),
		IPT:             ipt,
	}
}

func (fcr *Runtime) ConfigureNetwork(cidr string) {
	logrus.Infof("CIDR %s dynamically allocated by the control plane", cidr)

	fcr.NetworkManager = NewNetworkPoolManager(fcr.config.InternalIPPrefix, managers.CIDRToPrefix(cidr), fcr.config.NetworkPoolSize)
}

func (fcr *Runtime) createVMCS() *VMControlStructure {
	return &VMControlStructure{
		Context: context.Background(),

		KernelPath:     fcr.config.Kernel,
		FileSystemPath: fcr.config.FileSystem,

		SandboxID: fmt.Sprintf("firecracker-%d", fcr.idGenerator.Int()),
	}
}

func createMetadata(in *proto.ServiceInfo, vmcs *VMControlStructure) *managers.Metadata {
	return &managers.Metadata{
		ServiceName: in.Name,

		RuntimeMetadata: Metadata{
			VMCS: vmcs,
		},

		IP:        vmcs.NetworkConfiguration.ExposedIP,
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

	err, netCreateDuration, vmCreateDuration, vmStartDuration := StartFirecrackerVM(fcr.NetworkManager, vmcs, fcr.config.VMDebugMode, snapshot)
	if err != nil {
		// resource deallocation already done in StartFirecrackerVM
		return &proto.SandboxCreationStatus{Success: false}, err
	}

	startConfigureMonitoring := time.Now()
	metadata := createMetadata(in, vmcs)

	// VM process monitoring
	vmPID, err := vmcs.VM.PID()
	if err != nil {
		fcr.NetworkManager.GiveUpNetwork(vmcs.NetworkConfiguration)

		logrus.Debugf("Failed to get PID of the virtual machine - %v", err)
		return &proto.SandboxCreationStatus{Success: false}, err
	}

	fcr.SandboxManager.AddSandbox(vmcs.SandboxID, metadata)
	fcr.ProcessMonitor.AddChannel(uint32(vmPID), metadata.ExitStatusChannel)
	configureMonitoringDuration := time.Since(startConfigureMonitoring)

	go containerd.WatchExitChannel(fcr.cpApi, metadata, func(metadata *managers.Metadata) string {
		return metadata.RuntimeMetadata.(Metadata).VMCS.SandboxID
	})

	logrus.Debug("Worker node part: ", time.Since(start).Milliseconds(), " ms")

	url := fmt.Sprintf("%s:%d", vmcs.NetworkConfiguration.ExposedIP, metadata.GuestPort)

	// blocking call
	timeToPass, passed := managers.SendReadinessProbe(url)

	// create a snapshot for the service if it does not exist
	startSnapshotCreation := time.Now()
	if passed && fcr.config.UseSnapshots && !fcr.SnapshotManager.Exists(in.Image) {
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
			Success: true,
			ID:      vmcs.SandboxID,
			URL:     url,
			LatencyBreakdown: &proto.SandboxCreationBreakdown{
				Total:               durationpb.New(time.Since(start)),
				SandboxCreate:       durationpb.New(vmCreateDuration),
				NetworkSetup:        durationpb.New(netCreateDuration),
				SandboxStart:        durationpb.New(vmStartDuration),
				ReadinessProbing:    durationpb.New(timeToPass),
				Iptables:            durationpb.New(0),
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
			LatencyBreakdown: &proto.SandboxCreationBreakdown{
				Total: durationpb.New(time.Since(start)),
			},
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

	sandboxMetadata := metadata.RuntimeMetadata.(Metadata)

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

func (fcr *Runtime) CreateTaskSandbox(_ context.Context, _ *proto.WorkflowTaskInfo) (*proto.SandboxCreationStatus, error) {
	// not supported by the firecracker runtime
	return &proto.SandboxCreationStatus{
		Success:          false,
		ID:               "-1",
		LatencyBreakdown: &proto.SandboxCreationBreakdown{},
	}, nil
}

func (fcr *Runtime) ListEndpoints(_ context.Context, _ *emptypb.Empty) (*proto.EndpointsList, error) {
	return fcr.SandboxManager.ListEndpoints()
}

func (fcr *Runtime) PrepullImage(grpcCtx context.Context, imageInfo *proto.ImageInfo) (*proto.ActionStatus, error) {
	// TODO: Implement Firecracker image fetching.
	return &proto.ActionStatus{
		Success: false,
		Message: "Firecracker runtime does not currently support prepulling images.",
	}, nil
}

func (fcr *Runtime) GetImages(grpcCtx context.Context) ([]*proto.ImageInfo, error) {
	// TODO: Implement Firecracker image fetching.
	return []*proto.ImageInfo{}, errors.New("image pulling in Firecracker not implemented yet")
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

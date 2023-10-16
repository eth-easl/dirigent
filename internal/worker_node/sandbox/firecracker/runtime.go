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
	"time"
)

type Runtime struct {
	sandbox.RuntimeInterface

	cpApi proto.CpiInterfaceClient

	VMDebugMode bool

	KernelPath     string
	FileSystemPath string
	IpManager      *IPManager

	SandboxManager *managers.SandboxManager
	ProcessMonitor *managers.ProcessMonitor
	IPT            *iptables.IPTables
}

type FirecrackerMetadata struct {
	managers.RuntimeMetadata

	VMCS *VMControlStructure
}

func NewFirecrackerRuntime(hostname string, cpApi proto.CpiInterfaceClient, kernelPath string, fileSystemPath string, ipPrefix string, vmDebugMode bool) *Runtime {
	_ = DeleteFirecrackerTAPDevices()
	ipt, _ := managers.NewIptablesUtil()

	return &Runtime{
		cpApi: cpApi,

		VMDebugMode: vmDebugMode,

		KernelPath:     kernelPath,
		FileSystemPath: fileSystemPath,
		IpManager:      NewIPManager(ipPrefix),

		SandboxManager: managers.NewSandboxManager(hostname),
		ProcessMonitor: managers.NewProcessMonitor(),
		IPT:            ipt,
	}
}

func (fcr *Runtime) CreateSandbox(_ context.Context, in *proto.ServiceInfo) (*proto.SandboxCreationStatus, error) {
	start := time.Now()

	vmcs := &VMControlStructure{
		Context: context.Background(),

		KernelPath:     fcr.KernelPath,
		FileSystemPath: fcr.FileSystemPath,
		IpManager:      fcr.IpManager,

		SandboxID: fmt.Sprintf("firecracker-%d", rand.Int()),
	}

	err, tapCreation, vmCreate, vmStart := StartFirecrackerVM(vmcs, fcr.VMDebugMode)
	if err != nil {
		// TODO: deallocate resources
		return &proto.SandboxCreationStatus{Success: false}, err
	}

	metadata := &managers.Metadata{
		ServiceName: in.Name,

		RuntimeMetadata: FirecrackerMetadata{
			VMCS: vmcs,
		},

		HostPort:  containerd.AssignRandomPort(),
		IP:        vmcs.tapLink.VmIP,
		GuestPort: int(in.PortForwarding.GuestPort),

		ExitStatusChannel: make(chan uint32),
	}

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

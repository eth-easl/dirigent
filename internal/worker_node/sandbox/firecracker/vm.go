package firecracker

import (
	"context"
	"errors"
	"fmt"
	"github.com/firecracker-microvm/firecracker-go-sdk"
	"github.com/sirupsen/logrus"
	"os"
	"os/exec"
	"syscall"
	"time"
)

type VMControlStructure struct {
	Context context.Context

	VM                   *firecracker.Machine
	VMConfig             *firecracker.Config
	NetworkConfiguration *NetworkConfig

	SandboxID string

	KernelPath     string
	FileSystemPath string

	HostPort  int
	GuestPort int
}

func getVMCommandBuild(vmcs *VMControlStructure) *exec.Cmd {
	vmCommandBuild := firecracker.VMCommandBuilder{}.
		WithBin("firecracker").
		WithSocketPath(vmcs.VMConfig.SocketPath)

	if logrus.GetLevel() != logrus.InfoLevel {
		vmCommandBuild = vmCommandBuild.
			WithStdin(os.Stdin).
			WithStdout(os.Stdout).
			WithStderr(os.Stderr)
	}

	return vmCommandBuild.Build(vmcs.Context)
}

func StartFirecrackerVM(networkManager *NetworkPoolManager, vmcs *VMControlStructure, vmDebugMode bool, snapshotMetadata *SnapshotMetadata) (error, time.Duration, time.Duration, time.Duration) {
	networkCreation := time.Now()
	vmcs.NetworkConfiguration = networkManager.GetOneConfig()
	if vmcs.NetworkConfiguration == nil {
		return errors.New("error getting a network interface"), time.Duration(0), time.Duration(0), time.Duration(0)
	}
	tapEnd := time.Since(networkCreation)
	logrus.Debug("Time to create network: ", tapEnd.Milliseconds(), " ms")

	logrus.Debugf("VM %s TAP external = %s, TAP internal = %s, TAP MAC = %s, VETH external (%s) = %s, VETH internal = %s, and exposed through %s",
		vmcs.SandboxID,
		vmcs.NetworkConfiguration.TapExternalIP,
		vmcs.NetworkConfiguration.TapInternalIP,
		vmcs.NetworkConfiguration.TapMAC,
		vmcs.NetworkConfiguration.VETHHostName,
		vmcs.NetworkConfiguration.VETHExternalIP,
		vmcs.NetworkConfiguration.VETHInternalIP,
		vmcs.NetworkConfiguration.ExposedIP,
	)

	startVMCreation := time.Now()

	makeFirecrackerConfig(vmcs, vmDebugMode, snapshotMetadata)
	newMachineOpts := []firecracker.Opt{firecracker.WithProcessRunner(getVMCommandBuild(vmcs))}
	if vmDebugMode {
		logger := logrus.New()
		logger.SetLevel(logrus.GetLevel())
		newMachineOpts = append(newMachineOpts, firecracker.WithLogger(logrus.NewEntry(logger)))
	}

	if snapshotMetadata != nil {
		snapshotsOpts := firecracker.WithSnapshot(snapshotMetadata.MemoryPath, snapshotMetadata.SnapshotPath)
		newMachineOpts = append(newMachineOpts, snapshotsOpts)

		vmcs.VMConfig = &firecracker.Config{
			SocketPath:        vmcs.VMConfig.SocketPath,
			LogLevel:          vmcs.VMConfig.LogLevel,
			Drives:            vmcs.VMConfig.Drives,
			NetworkInterfaces: vmcs.VMConfig.NetworkInterfaces,
			NetNS:             vmcs.VMConfig.NetNS,
		}
	}

	machine, err := firecracker.NewMachine(vmcs.Context, *vmcs.VMConfig, newMachineOpts...)
	if err != nil {
		logrus.Errorf("Failed creating a new virtual machine - %v", err)
		networkManager.GiveUpNetwork(vmcs.NetworkConfiguration)

		return err, tapEnd, time.Since(startVMCreation), time.Duration(0)
	}
	vmcs.VM = machine

	vmCreateEnd := time.Since(startVMCreation)
	logrus.Debug("VM creation time: ", vmCreateEnd.Milliseconds(), " ms")

	timeVMStart := time.Now()

	err = machine.Start(vmcs.Context)
	if err != nil {
		logrus.Errorf("Error starting a virtual machine - %v", err)
		networkManager.GiveUpNetwork(vmcs.NetworkConfiguration)

		return err, tapEnd, vmCreateEnd, time.Since(timeVMStart)
	}

	if snapshotMetadata != nil {
		err = machine.ResumeVM(vmcs.Context)
		if err != nil {
			logrus.Errorf("Error creating virtual machine from snapshot - %v", err)
			networkManager.GiveUpNetwork(vmcs.NetworkConfiguration)

			return err, tapEnd, vmCreateEnd, time.Since(timeVMStart)
		}
	}

	vmStartEnd := time.Since(timeVMStart)
	logrus.Debug("VM starting time: ", vmStartEnd.Milliseconds(), " ms")

	return err, tapEnd, vmCreateEnd, vmStartEnd
}

func StopFirecrackerVM(vmcs *VMControlStructure) error {
	pid, err := vmcs.VM.PID()
	if err != nil {
		return err
	}

	err = syscall.Kill(pid, syscall.SIGKILL)
	return err
}

func deleteLogs(vmcs *VMControlStructure) {
	// don't handle error because the file may not exist
	_ = os.Remove(fmt.Sprintf(pathToLog, vmcs.SandboxID))
}

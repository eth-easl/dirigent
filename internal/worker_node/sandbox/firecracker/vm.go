package firecracker

import (
	"context"
	"github.com/firecracker-microvm/firecracker-go-sdk"
	"github.com/sirupsen/logrus"
	"os"
	"os/exec"
	"syscall"
	"time"
)

type VMControlStructure struct {
	Context context.Context

	VM      *firecracker.Machine
	Config  *firecracker.Config
	TapLink *TAPLink

	SandboxID string
	NetworkNS string

	KernelPath     string
	FileSystemPath string
	IpManager      *IPManager
}

func getVMCommandBuild(vmcs *VMControlStructure) *exec.Cmd {
	vmCommandBuild := firecracker.VMCommandBuilder{}.
		WithBin("firecracker").
		WithSocketPath(vmcs.Config.SocketPath)

	if logrus.GetLevel() != logrus.InfoLevel {
		vmCommandBuild = vmCommandBuild.
			WithStdin(os.Stdin).
			WithStdout(os.Stdout).
			WithStderr(os.Stderr)
	}

	return vmCommandBuild.Build(vmcs.Context)
}

func StartFirecrackerVM(vmcs *VMControlStructure, vmDebugMode bool, snapshotMetadata *SnapshotMetadata) (error, time.Duration, time.Duration, time.Duration) {
	startTAP := time.Now()
	err := createTAPDevice(vmcs, snapshotMetadata)
	if err != nil {
		logrus.Error("Error setting up network for a microVM - ", err)
		return err, time.Since(startTAP), time.Duration(0), time.Duration(0)
	}
	tapEnd := time.Since(startTAP)
	logrus.Debug("TAP creation time: ", tapEnd.Milliseconds(), " ms")

	logrus.Debugf("VM %s allocated host gateway IP = %s, VM IP = %s, MAC = %s",
		vmcs.SandboxID,
		vmcs.TapLink.GatewayIP,
		vmcs.TapLink.VmIP,
		vmcs.TapLink.MAC,
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

		vmcs.Config = &firecracker.Config{
			SocketPath:        vmcs.Config.SocketPath,
			LogLevel:          vmcs.Config.LogLevel,
			Drives:            vmcs.Config.Drives,
			NetworkInterfaces: vmcs.Config.NetworkInterfaces,
			NetNS:             vmcs.Config.NetNS,
		}
	}

	machine, err := firecracker.NewMachine(vmcs.Context, *vmcs.Config, newMachineOpts...)
	if err != nil {
		logrus.Fatal(err)
		return err, tapEnd, time.Since(startVMCreation), time.Duration(0)
	}
	vmcs.VM = machine

	vmCreateEnd := time.Since(startVMCreation)
	logrus.Debug("VM creation time: ", vmCreateEnd.Milliseconds(), " ms")

	logrus.Debug("Starting VM with IP = ", vmcs.TapLink.VmIP, " (MAC = ", vmcs.TapLink.MAC, ")")

	timeVMStart := time.Now()

	err = machine.Start(vmcs.Context)
	if err != nil {
		logrus.Error(err)
		return err, tapEnd, vmCreateEnd, time.Since(timeVMStart)
	}

	if snapshotMetadata != nil {
		err = machine.ResumeVM(vmcs.Context)
		if err != nil {
			logrus.Error("Error creating virtual machine from snapshot - ", err)
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

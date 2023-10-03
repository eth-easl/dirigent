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

	vm      *firecracker.Machine
	config  *firecracker.Config
	tapLink *TAPLink

	SandboxID string

	KernelPath     string
	FileSystemPath string
	IpManager      *IPManager
}

func getVMCommandBuild(vmcs *VMControlStructure) *exec.Cmd {
	vmCommandBuild := firecracker.VMCommandBuilder{}.
		WithBin("firecracker").
		WithSocketPath(vmcs.config.SocketPath)

	if logrus.GetLevel() != logrus.InfoLevel {
		vmCommandBuild = vmCommandBuild.
			WithStdin(os.Stdin).
			WithStdout(os.Stdout).
			WithStderr(os.Stderr)
	}

	return vmCommandBuild.Build(vmcs.Context)
}

func StartFirecrackerVM(vmcs *VMControlStructure, vmDebugMode bool) (error, time.Duration, time.Duration, time.Duration) {
	startTAP := time.Now()
	err := createTAPDevice(vmcs)
	if err != nil {
		logrus.Error("Error setting up network for a microVM - ", err)
		return err, time.Since(startTAP), time.Duration(0), time.Duration(0)
	}
	tapEnd := time.Since(startTAP)
	logrus.Debug("TAP creation time: ", tapEnd.Milliseconds(), " ms")

	logrus.Debugf("VM %s allocated host gateway IP = %s, VM IP = %s, MAC = %s",
		vmcs.SandboxID,
		vmcs.tapLink.GatewayIP,
		vmcs.tapLink.VmIP,
		vmcs.tapLink.MAC,
	)

	startVMCreation := time.Now()

	makeFirecrackerConfig(vmcs, vmDebugMode)
	newMachineOpts := []firecracker.Opt{firecracker.WithProcessRunner(getVMCommandBuild(vmcs))}
	if vmDebugMode {
		logger := logrus.New()
		logger.SetLevel(logrus.GetLevel())
		newMachineOpts = append(newMachineOpts, firecracker.WithLogger(logrus.NewEntry(logger)))
	}

	machine, err := firecracker.NewMachine(vmcs.Context, *vmcs.config, newMachineOpts...)
	if err != nil {
		logrus.Fatal(err)
		return err, tapEnd, time.Since(startVMCreation), time.Duration(0)
	}
	vmcs.vm = machine

	vmCreateEnd := time.Since(startVMCreation)
	logrus.Debug("VM creation time: ", vmCreateEnd.Milliseconds(), " ms")

	logrus.Debug("Starting VM with IP = ", vmcs.tapLink.VmIP, " (MAC = ", vmcs.tapLink.MAC, ")")

	timeVMStart := time.Now()

	err = machine.Start(vmcs.Context)
	if err != nil {
		logrus.Error(err)
		return err, tapEnd, vmCreateEnd, time.Since(timeVMStart)
	}

	vmStartEnd := time.Since(timeVMStart)
	logrus.Debug("VM starting time: ", vmStartEnd.Milliseconds(), " ms")

	return err, tapEnd, vmCreateEnd, vmStartEnd
}

func StopFirecrackerVM(vmcs *VMControlStructure) error {
	pid, err := vmcs.vm.PID()
	if err != nil {
		return err
	}

	err = syscall.Kill(pid, syscall.SIGKILL)
	return err
}

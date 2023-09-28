package firecracker

import (
	"context"
	"github.com/firecracker-microvm/firecracker-go-sdk"
	"github.com/sirupsen/logrus"
	"os"
	"os/exec"
	"syscall"
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
		WithSocketPath(vmcs.config.SocketPath).
		WithStderr(os.Stderr)

	if logrus.GetLevel() != logrus.InfoLevel {
		vmCommandBuild = vmCommandBuild.
			WithStdin(os.Stdin).
			WithStdout(os.Stdout)
	}

	return vmCommandBuild.Build(vmcs.Context)
}

func StartFirecrackerVM(vmcs *VMControlStructure) error {
	err := createTAPDevice(vmcs)
	if err != nil {
		logrus.Error("Error setting up network for a microVM - ", err)
		return err
	}

	logrus.Debugf("VM %s allocated host IP = %s, VM IP = %s, MAC = %s",
		vmcs.SandboxID,
		vmcs.tapLink.IP,
		vmcs.tapLink.VMIP,
		vmcs.tapLink.MAC,
	)

	makeFirecrackerConfig(vmcs)

	logger := logrus.New()
	logger.SetLevel(logrus.GetLevel())

	newMachineOpts := []firecracker.Opt{firecracker.WithProcessRunner(getVMCommandBuild(vmcs))}
	if logrus.GetLevel() != logrus.InfoLevel {
		newMachineOpts = append(newMachineOpts, firecracker.WithLogger(logrus.NewEntry(logger)))
	}

	machine, err := firecracker.NewMachine(vmcs.Context, *vmcs.config, newMachineOpts...)
	if err != nil {
		logrus.Fatal(err)
		return err
	}

	vmcs.vm = machine

	logrus.Debug("Starting VM with IP = ", vmcs.tapLink.IP, " (MAC = ", vmcs.tapLink.MAC, ")")

	err = machine.Start(vmcs.Context)
	if err != nil {
		logrus.Fatal(err)
		return err
	}

	return err
}

func StopFirecrackerVM(vmcs *VMControlStructure) error {
	pid, err := vmcs.vm.PID()
	if err != nil {
		return err
	}

	err = syscall.Kill(pid, syscall.SIGKILL)
	return err
}

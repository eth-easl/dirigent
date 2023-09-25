package firecracker

import (
	"context"
	"github.com/firecracker-microvm/firecracker-go-sdk"
	"github.com/sirupsen/logrus"
	"os"
	"os/exec"
)

type VMControlStructure struct {
	Context context.Context
	cancel  context.CancelFunc

	vm      *firecracker.Machine
	config  *firecracker.Config
	tapLink *TAPLink

	KernelPath     string
	FileSystemPath string
	IpManager      *IPManager
}

func getVMCommandBuild(vmcs *VMControlStructure) *exec.Cmd {
	return firecracker.VMCommandBuilder{}.
		WithBin("firecracker").
		WithSocketPath(vmcs.config.SocketPath).
		WithStdin(os.Stdin).
		WithStdout(os.Stdout).
		WithStderr(os.Stderr).
		Build(vmcs.Context)
}

func StartFirecrackerVM(vmcs *VMControlStructure) bool {
	err := createTAPDevice(vmcs)
	if err != nil {
		logrus.Error("Error setting up network for a microVM - ", err)
		return false
	}

	makeFirecrackerConfig(vmcs)

	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	machine, err := firecracker.NewMachine(vmcs.Context,
		*vmcs.config,
		firecracker.WithLogger(logrus.NewEntry(logger)),
		firecracker.WithProcessRunner(getVMCommandBuild(vmcs)),
	)
	if err != nil {
		logrus.Fatal(err)
		return false
	}

	vmcs.vm = machine

	logrus.Debug("Starting VM with IP = ", vmcs.tapLink.IP, " (MAC = ", vmcs.tapLink.MAC, ")")

	err = machine.Start(vmcs.Context)
	if err != nil {
		logrus.Fatal(err)
		return false
	}

	err = machine.Wait(vmcs.Context)
	if err != nil {
		logrus.Fatal(err)
		return false
	}

	return true
}

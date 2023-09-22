package firecracker

import (
	"context"
	"github.com/firecracker-microvm/firecracker-go-sdk"
	"github.com/sirupsen/logrus"
	"os"
	"os/exec"
)

type VMControlStructure struct {
	context context.Context
	cancel  context.CancelFunc

	vm      *firecracker.Machine
	config  *firecracker.Config
	tapLink *TAPLink

	kernelPath     string
	fileSystemPath string
	ipManager      *IPManager
}

func getVMCommandBuild(vmcs *VMControlStructure) *exec.Cmd {
	return firecracker.VMCommandBuilder{}.
		WithBin("firecracker").
		WithSocketPath(vmcs.config.SocketPath).
		WithStdin(os.Stdin).
		WithStdout(os.Stdout).
		WithStderr(os.Stderr).
		Build(vmcs.context)
}

func StartFirecrackerVM(vmcs *VMControlStructure) bool {
	err := createTAPDevice(vmcs)
	if err != nil {
		logrus.Error("Error setting up network for a microVM - ", err)
		return false
	}

	makeFirecrackerConfig(vmcs)

	machine, err := firecracker.NewMachine(vmcs.context, *vmcs.config, firecracker.WithProcessRunner(getVMCommandBuild(vmcs)))
	if err != nil {
		logrus.Fatal(err)
		return false
	}

	vmcs.vm = machine

	err = machine.Start(vmcs.context)
	if err != nil {
		logrus.Fatal(err)
		return false
	}

	err = machine.Wait(vmcs.context)
	if err != nil {
		logrus.Fatal(err)
		return false
	}

	return true
}

package firecracker

import (
	"fmt"
	"github.com/firecracker-microvm/firecracker-go-sdk"
	"github.com/firecracker-microvm/firecracker-go-sdk/client/models"
	"github.com/sirupsen/logrus"
	"os"
	"path/filepath"
)

const (
	noDebugKernelArgs = "panic=1 pci=off nomodule reboot=k tsc=reliable quiet i8042.nokbd i8042.noaux 8250.nr_uarts=0 ipv6.disable=1"
	debugKernelArgs   = "panic=1 pci=off nomodule reboot=k tsc=reliable quiet i8042.noaux ipv6.disable=1 console=ttyS0 random.trust_cpu=on"
	ipKernelArg       = " ip=%s::%s:255.255.255.252::eth0:off"
)

func makeSocketPath(vmmID string) string {
	return filepath.Join(os.TempDir(), vmmID)
}

func makeFirecrackerConfig(vmcs *VMControlStructure, vmDebugMode bool) {
	if vmcs.tapLink == nil {
		logrus.Error("Network must be created before creating a Firecracker config.")
		return
	}

	kernelArgs := noDebugKernelArgs
	if vmDebugMode {
		kernelArgs = debugKernelArgs
	}
	kernelArgs += fmt.Sprintf(ipKernelArg, vmcs.tapLink.VmIP, vmcs.tapLink.GatewayIP)

	vmcs.config = &firecracker.Config{
		SocketPath:      makeSocketPath(vmcs.SandboxID),
		KernelImagePath: vmcs.KernelPath,
		KernelArgs:      kernelArgs,
		LogPath:         fmt.Sprintf("/tmp/%s.log", vmcs.SandboxID),
		LogLevel:        "Debug",
		Drives: []models.Drive{{
			DriveID:      firecracker.String("1"),
			PathOnHost:   firecracker.String(vmcs.FileSystemPath),
			IsReadOnly:   firecracker.Bool(false),
			IsRootDevice: firecracker.Bool(true),
		}},
		NetworkInterfaces: []firecracker.NetworkInterface{{
			StaticConfiguration: &firecracker.StaticNetworkConfiguration{
				HostDevName: vmcs.tapLink.Device,
				MacAddress:  vmcs.tapLink.MAC,
			},
		}},
		// TODO: add resource requests/limits
		MachineCfg: models.MachineConfiguration{
			MemSizeMib: firecracker.Int64(128),
			VcpuCount:  firecracker.Int64(1),
			Smt:        firecracker.Bool(false),
		},
		// TODO: integrate jailer with Firecracker
		JailerCfg: nil,
		// TODO: integrate separate network ns with Firecracker
		NetNS: "",
		// TODO: integrate seccomp filters with Firecracker
		Seccomp: firecracker.SeccompConfig{},
	}
}

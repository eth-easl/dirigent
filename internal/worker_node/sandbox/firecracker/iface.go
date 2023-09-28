package firecracker

import (
	"fmt"
	"github.com/firecracker-microvm/firecracker-go-sdk"
	"github.com/firecracker-microvm/firecracker-go-sdk/client/models"
	"github.com/sirupsen/logrus"
	"os"
	"path/filepath"
)

func makeSocketPath(vmmID string) string {
	return filepath.Join(os.TempDir(), vmmID)
}

func createCNIConfig() *firecracker.CNIConfiguration {
	return &firecracker.CNIConfiguration{
		NetworkName: "fcnet",
		IfName:      "veth0",
		ConfDir:     "/home/lcvetkovic/projects/cluster_manager/configs/firecracker",
	}
}

func makeFirecrackerConfig(vmcs *VMControlStructure) {
	if vmcs.tapLink == nil {
		logrus.Fatal("Network must be created before creating a Firecracker config.")
	}

	kernelArgs := "panic=1 pci=off nomodules reboot=k tsc=reliable quiet i8042.nokbd i8042.noaux 8250.nr_uarts=0 ipv6.disable=1"
	if logrus.GetLevel() != logrus.InfoLevel {
		kernelArgs = "panic=1 pci=off nomodules reboot=k tsc=reliable quiet i8042.noaux ipv6.disable=1 console=ttyS0 random.trust_cpu=on"
	}
	kernelArgs += fmt.Sprintf(" ip=%s::%s:255.255.255.252::eth0:off", vmcs.tapLink.VMIP, vmcs.tapLink.IP)

	vmcs.config = &firecracker.Config{
		SocketPath:      makeSocketPath(vmcs.SandboxID),
		KernelImagePath: vmcs.KernelPath,
		KernelArgs:      kernelArgs,
		LogPath:         fmt.Sprintf("/tmp/%s.log", vmcs.SandboxID),
		LogLevel:        logrus.DebugLevel.String(),
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
			MemSizeMib: firecracker.Int64(256),
			VcpuCount:  firecracker.Int64(2),
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

package firecracker

import (
	"fmt"
	"github.com/firecracker-microvm/firecracker-go-sdk"
	"github.com/firecracker-microvm/firecracker-go-sdk/client/models"
	"github.com/sirupsen/logrus"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func makeSocketPath(vmmID string) string {
	filename := strings.Join([]string{"firecracker.socket", vmmID}, "-")
	return filepath.Join(os.TempDir(), filename)
}

func createCNIConfig() *firecracker.CNIConfiguration {
	return &firecracker.CNIConfiguration{
		NetworkName: "fcnet",
		IfName:      "veth0",
		ConfDir:     "/home/lcvetkovic/projects/cluster_manager/configs/firecracker",
	}
}

func makeFirecrackerConfig(vmcs *VMControlStructure) {
	socket := makeSocketPath(strconv.Itoa(rand.Int()))

	if vmcs.tapLink == nil {
		logrus.Fatal("Network must be created before creating a Firecracker config.")
	}

	vmcs.config = &firecracker.Config{
		SocketPath:      makeSocketPath(strconv.Itoa(rand.Int())),
		KernelImagePath: vmcs.kernelPath,
		KernelArgs:      "panic=1 pci=off nomodules reboot=k tsc=reliable quiet i8042.noaux ipv6.disable=1 console=ttyS0 random.trust_cpu=on",
		LogPath:         fmt.Sprintf("%s.log", socket),
		Drives: []models.Drive{{
			DriveID:      firecracker.String("1"),
			PathOnHost:   firecracker.String(vmcs.fileSystemPath),
			IsReadOnly:   firecracker.Bool(false),
			IsRootDevice: firecracker.Bool(true),
		}},
		NetworkInterfaces: []firecracker.NetworkInterface{{
			StaticConfiguration: &firecracker.StaticNetworkConfiguration{
				HostDevName: vmcs.tapLink.Device,
				MacAddress:  vmcs.tapLink.MAC,
			},
		}},
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

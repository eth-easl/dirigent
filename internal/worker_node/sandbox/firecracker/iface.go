/*
 * MIT License
 *
 * Copyright (c) 2024 EASL
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */

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
	pathToLog         = "/tmp/%s.log"
)

func makeSocketPath(vmmID string) string {
	return filepath.Join(os.TempDir(), vmmID)
}

func makeFirecrackerConfig(vmcs *VMControlStructure, vmDebugMode bool, metadata *SnapshotMetadata) {
	if vmcs.NetworkConfiguration == nil {
		logrus.Error("Network must be created before creating a Firecracker config.")
		return
	}

	kernelArgs := noDebugKernelArgs
	if vmDebugMode {
		kernelArgs = debugKernelArgs
	}
	kernelArgs += fmt.Sprintf(ipKernelArg, vmcs.NetworkConfiguration.TapInternalIP, vmcs.NetworkConfiguration.TapExternalIP)

	vmcs.VMConfig = &firecracker.Config{
		SocketPath:      makeSocketPath(vmcs.SandboxID),
		KernelImagePath: vmcs.KernelPath,
		KernelArgs:      kernelArgs,
		LogPath:         fmt.Sprintf(pathToLog, vmcs.SandboxID),
		LogLevel:        "Info",
		Drives: []models.Drive{{
			DriveID:      firecracker.String("1"),
			PathOnHost:   firecracker.String(vmcs.FileSystemPath),
			IsReadOnly:   firecracker.Bool(false),
			IsRootDevice: firecracker.Bool(true),
		}},
		// TODO: add resource requests/limits
		MachineCfg: models.MachineConfiguration{
			MemSizeMib: firecracker.Int64(2048),
			VcpuCount:  firecracker.Int64(2),
			Smt:        firecracker.Bool(false),
		},
		NetworkInterfaces: []firecracker.NetworkInterface{{
			StaticConfiguration: &firecracker.StaticNetworkConfiguration{
				HostDevName: vmcs.NetworkConfiguration.TapDeviceName,
				MacAddress:  vmcs.NetworkConfiguration.TapMAC,
			},
		}},
		NetNS: fmt.Sprintf("/var/run/netns/%s", vmcs.NetworkConfiguration.NetNS),
	}
}

package main

import (
	"cluster_manager/internal/worker_node/sandbox/firecracker"
	"context"
	"github.com/sirupsen/logrus"
	"os"
)

func main() {
	os.Setenv("PATH", os.Getenv("PATH")+":/usr/local/bin/firecracker")

	logrus.SetLevel(logrus.DebugLevel)

	_ = firecracker.DeleteFirecrackerTAPDevices()

	vmcs := &firecracker.VMControlStructure{
		Context:        context.Background(),
		KernelPath:     "/home/lcvetkovic/projects/cluster_manager/configs/vmlinux.bin",
		FileSystemPath: "/home/lcvetkovic/projects/dandelion_experiments/servers/firecracker/image/rootfs.ext4",
		IpManager:      firecracker.NewIPManager("169.254"),
	}

	firecracker.StartFirecrackerVM(vmcs)
}

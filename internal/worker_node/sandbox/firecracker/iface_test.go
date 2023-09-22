package firecracker

import (
	"context"
	"os"
	"testing"
)

func TestVMStart(t *testing.T) {
	os.Setenv("PATH", os.Getenv("PATH")+":/usr/local/bin/firecracker")

	vmcs := &VMControlStructure{
		context:        context.Background(),
		kernelPath:     "/home/lcvetkovic/projects/cluster_manager/configs/vmlinux.bin",
		fileSystemPath: "/home/lcvetkovic/projects/dandelion_experiments/servers/firecracker/image/rootfs.ext4",
		ipManager:      NewIPManager("169.254"),
	}

	StartFirecrackerVM(vmcs)
}

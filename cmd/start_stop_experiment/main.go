package main

import (
	"cluster_manager/internal/worker_node/managers"
	"cluster_manager/internal/worker_node/sandbox"
	"cluster_manager/internal/worker_node/sandbox/containerd"
	"cluster_manager/internal/worker_node/sandbox/fcctr"
	"cluster_manager/internal/worker_node/sandbox/firecracker"
	"cluster_manager/pkg/config"
	"cluster_manager/proto"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
)

var (
	runtime   = flag.String("runtime", "fcctr", "Runtime to use")
	snapshots = flag.Bool("snapshots", true, "Use snapshots")
	repeat    = flag.Int("repeat", 1, "Number of times to repeat the experiment")
)

func startStopSandbox(ctx context.Context, r sandbox.RuntimeInterface, serviceInfo *proto.ServiceInfo) (*proto.SandboxCreationBreakdown, error) {
	createStatus, err := r.CreateSandbox(ctx, serviceInfo)
	if err != nil {
		return nil, err
	}
	_, err = r.DeleteSandbox(ctx, &proto.SandboxID{
		ID: createStatus.ID,
	})
	return createStatus.LatencyBreakdown, err
}

func createRuntime(runtime string, useSnapshots bool) (sandbox.RuntimeInterface, sandbox.PostRegistrationCallback) {
	sandboxManager := managers.NewSandboxManager("test")
	switch runtime {
	case "containerd":
		return containerd.NewContainerdRuntime(nil, config.ContainerdConfig{
			CRIPath:       "/run/containerd/containerd.sock",
			CNIConfigPath: "configs/cni.conf",
			PrefetchImage: false,
		}, sandboxManager, false), sandbox.ContainerdPostRegistrationCallback
	case "firecracker":
		return firecracker.NewFirecrackerRuntime(nil, sandboxManager, &config.FirecrackerConfig{
			Kernel:           "configs/firecracker/vmlinux-4.14.bin",
			FileSystem:       "configs/firecracker/rootfs.ext4",
			VMDebugMode:      false,
			InternalIPPrefix: "100",
			NetworkPoolSize:  8,
			UseSnapshots:     useSnapshots,
		}), sandbox.FirecrackerPostRegistrationCallback
	case "fcctr":
		return fcctr.NewRuntime(nil, sandboxManager, &config.FirecrackerConfig{
			InternalIPPrefix: "100",
			NetworkPoolSize:  8,
			UseSnapshots:     useSnapshots,
		}, "info"), sandbox.FirecrackerContainerdPostRegistrationCallback
	default:
		logrus.Fatalf("Unknown runtime: %s", runtime)
		return nil, sandbox.EmptyPostRegistrationCallback
	}
}

func writeResults(runtime string, snapshots bool, latencies []*proto.SandboxCreationBreakdown) error {
	// Write a list of sandbox creation breaksdowns to a JSON file
	file, err := os.Create(fmt.Sprintf("results_%s_%v.json", runtime, snapshots))
	if err != nil {
		return err
	}
	defer file.Close()
	serialized, err := json.Marshal(latencies)
	if err != nil {
		return err
	}
	_, err = file.Write(serialized)
	if err != nil {
		return err
	}
	return nil
}

func main() {
	logrus.SetLevel(logrus.TraceLevel)
	flag.Parse()

	ctx := context.Background()
	serviceInfo := &proto.ServiceInfo{
		Name:  "test",
		Image: "docker.io/cvetkovic/dirigent_empty_function:latest",
		SandboxConfiguration: &proto.SandboxConfiguration{
			IterationMultiplier: 1,
			ColdStartBusyLoopMs: 10,
		},
		PortForwarding: &proto.PortMapping{
			GuestPort: 80,
		},
	}

	cidr := "192.254.0.0/16"
	r, callback := createRuntime(*runtime, *snapshots)
	if *snapshots {
		// Prepare the snapshot
		_, err := startStopSandbox(ctx, r, serviceInfo)
		if err != nil {
			logrus.WithError(err).Fatal("Failed to start/stop sandbox")
		}
	}
	callback(r, cidr)

	results := make([]*proto.SandboxCreationBreakdown, *repeat)
	for i := 0; i < *repeat; i++ {
		latency, err := startStopSandbox(ctx, r, serviceInfo)
		if err != nil {
			logrus.WithError(err).Fatal("Failed to start/stop sandbox")
		}
		results[i] = latency
	}

	err := writeResults(*runtime, *snapshots, results)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to write results")
	}
}

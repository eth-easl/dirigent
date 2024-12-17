package fcctr

import (
	"cluster_manager/internal/worker_node/managers"
	"cluster_manager/internal/worker_node/sandbox"
	ctrmanagers "cluster_manager/internal/worker_node/sandbox/containerd"
	"cluster_manager/internal/worker_node/sandbox/firecracker"
	"cluster_manager/pkg/config"
	"cluster_manager/proto"
	"context"
	"errors"
	"fmt"
	"syscall"
	"time"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/namespaces"
	"github.com/coreos/go-iptables/iptables"
	fcctr "github.com/firecracker-microvm/firecracker-containerd/firecracker-control/client"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/emptypb"
)

const (
	containerdSocket = "/run/firecracker-containerd/containerd.sock"
	ttrpcSocket      = containerdSocket + ".ttrpc"
	namespaceName    = "fcctr"
)

type Runtime struct {
	sandbox.RuntimeInterface

	cpApi       proto.CpiInterfaceClient
	idGenerator *managers.ThreadSafeRandomGenerator

	containerdClient *containerd.Client
	fcctrClient      *fcctr.Client

	imageManager    *ctrmanagers.ImageManager
	networkManager  *firecracker.NetworkPoolManager
	sandboxManager  *managers.SandboxManager
	snapshotManager *firecracker.SnapshotManager
	ipt             *iptables.IPTables

	config    *config.FirecrackerConfig
	verbosity string
}

func NewRuntime(cpApi proto.CpiInterfaceClient, sandboxManager *managers.SandboxManager, config *config.FirecrackerConfig, verbosity string) *Runtime {
	err := firecracker.DeleteUnusedNetworkDevices()
	if err != nil {
		logrus.WithError(err).Error("Failed to remove some or all network devices")
	}

	containerdClient, err := containerd.New(containerdSocket)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to create containerd client")
	}

	fcctrClient, err := fcctr.New(ttrpcSocket)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to create containerd client")
	}

	ipt, err := managers.NewIptablesUtil()
	if err != nil {
		logrus.WithError(err).Fatal("Failed to start iptables utility")
	}

	return &Runtime{
		cpApi:       cpApi,
		idGenerator: managers.NewThreadSafeRandomGenerator(),

		containerdClient: containerdClient,
		fcctrClient:      fcctrClient,

		imageManager:    ctrmanagers.NewContainerdImageManager(),
		sandboxManager:  sandboxManager,
		snapshotManager: firecracker.NewFirecrackerSnapshotManager(),
		ipt:             ipt,

		config:    config,
		verbosity: verbosity,
	}
}

func (r *Runtime) ConfigureNetwork(cidr string) {
	logrus.Infof("CIDR %s dynamically allocated by the control plane", cidr)

	r.networkManager = firecracker.NewNetworkPoolManager(r.config.InternalIPPrefix, managers.CIDRToPrefix(cidr), r.config.NetworkPoolSize)
}

func (r *Runtime) CreateSandbox(grpcCtx context.Context, serviceInfo *proto.ServiceInfo) (_ *proto.SandboxCreationStatus, retErr error) {
	logrus.Debugf("Creating sandbox for service = '%s'", serviceInfo.Name)

	start := time.Now()

	scs := &SandboxControlStructure{
		SandboxID:            fmt.Sprintf("fcctr-%d", r.idGenerator.Int()),
		SandboxConfiguration: serviceInfo.SandboxConfiguration,
	}
	ctx := namespaces.WithNamespace(grpcCtx, namespaceName)
	latencyBreakdown := &proto.SandboxCreationBreakdown{}

	// Check if snapshot has been created
	var snapshot *firecracker.SnapshotMetadata
	if r.config.UseSnapshots {
		var durationFindSnapshot time.Duration
		snapshot, durationFindSnapshot = FindSnapshot(r.snapshotManager, serviceInfo.Image)
		latencyBreakdown.FindSnapshot = durationpb.New(durationFindSnapshot)
	}

	if snapshot == nil {
		// Image fetching
		image, err, durationFetch := r.imageManager.GetImage(ctx,
			r.containerdClient,
			serviceInfo.Image,
			containerd.WithPullSnapshotter("devmapper"))
		latencyBreakdown.ImageFetch = durationpb.New(durationFetch)
		if err != nil {
			logrus.WithError(err).Warn("Failed to fetch an image")
			latencyBreakdown.Total = durationpb.New(time.Since(start))
			return &proto.SandboxCreationStatus{
				Success:          false,
				ID:               scs.SandboxID,
				LatencyBreakdown: latencyBreakdown,
			}, err
		}
		scs.Image = image
	}

	// Network configuration
	networkConfig, durationNetworkSetup, err := CreateNetworkConfig(r.networkManager)
	latencyBreakdown.NetworkSetup = durationpb.New(durationNetworkSetup)
	if err != nil {
		logrus.WithError(err).Warn("Failed to create network configuration")
		latencyBreakdown.Total = durationpb.New(time.Since(start))
		return &proto.SandboxCreationStatus{
			Success:          false,
			ID:               scs.SandboxID,
			LatencyBreakdown: latencyBreakdown,
		}, err
	}
	scs.NetworkConfiguration = networkConfig

	// VM creation
	durationVMCreate, err := CreateVM(ctx, r.fcctrClient, scs, snapshot, r.config.VMDebugMode)
	latencyBreakdown.SandboxCreate = durationpb.New(durationVMCreate)
	if err != nil {
		logrus.WithError(err).Warn("Failed to create a Firecracker VM")
		latencyBreakdown.Total = durationpb.New(time.Since(start))
		return &proto.SandboxCreationStatus{
			Success:          false,
			ID:               scs.SandboxID,
			LatencyBreakdown: latencyBreakdown,
		}, err
	}
	defer func() {
		if retErr != nil {
			if err := StopVM(ctx, r.fcctrClient, scs); err != nil {
				logrus.WithError(err).Warn("Failed to stop VM after failure")
			}
		}
	}()

	if snapshot == nil {
		// Container creation
		container, durationContainerCreate, err := CreateContainer(ctx, r.containerdClient, scs, r.verbosity)
		latencyBreakdown.SandboxCreate = durationpb.New(durationVMCreate + durationContainerCreate)
		if err != nil {
			logrus.WithError(err).Warn("Failed to create a container")
			latencyBreakdown.Total = durationpb.New(time.Since(start))
			return &proto.SandboxCreationStatus{
				Success:          false,
				ID:               scs.SandboxID,
				LatencyBreakdown: latencyBreakdown,
			}, err
		}
		defer func() {
			if retErr != nil {
				if err := container.Delete(ctx, containerd.WithSnapshotCleanup); err != nil {
					logrus.WithError(err).Warn("Failed to delete container after failure")
				}
			}
		}()

		// Container start
		task, durationStart, err := StartContainer(ctx, container)
		latencyBreakdown.SandboxStart = durationpb.New(durationStart)
		if err != nil {
			logrus.WithError(err).Warn("Failed to start container")
			latencyBreakdown.Total = durationpb.New(time.Since(start))
			return &proto.SandboxCreationStatus{
				Success:          false,
				ID:               scs.SandboxID,
				LatencyBreakdown: latencyBreakdown,
			}, err
		}
		defer func() {
			if retErr != nil {
				if err := task.Kill(ctx, syscall.SIGKILL); err != nil {
					logrus.WithError(err).Warn("Failed to kill task after failure")
				} else {
					status, err := task.Status(ctx)
					if err != nil {
						logrus.WithError(err).Warn("Failed to get task status after failure")
					} else {
						logrus.Debugf("Task status after killing: %s", status.Status)
					}
					if _, err := task.Delete(ctx); err != nil {
						logrus.WithError(err).Warn("Failed to delete task after failure")
					}
				}
			}
		}()
	}

	// Monitoring configuration
	metadata, durationMonitoring := ConfigureMonitoring(r.sandboxManager, r.cpApi, serviceInfo, scs)
	latencyBreakdown.ConfigureMonitoring = durationpb.New(durationMonitoring)
	logrus.Debugf("Sandbox creation took %d μs (%s)", time.Since(start).Microseconds(), scs.SandboxID)
	logrus.Debugf("Network namespace of %s is %s", scs.SandboxID, metadata.NetNs)

	url := fmt.Sprintf("%s:%d", scs.NetworkConfiguration.ExposedIP, metadata.GuestPort)

	// Readiness probing
	durationProbing, passed := managers.SendReadinessProbe(url)
	latencyBreakdown.ReadinessProbing = durationpb.New(durationProbing)
	if !passed {
		latencyBreakdown.Total = durationpb.New(time.Since(start))
		return &proto.SandboxCreationStatus{
			Success:          false,
			ID:               scs.SandboxID,
			LatencyBreakdown: latencyBreakdown,
		}, errors.New("readiness probe failed")
	}

	if snapshot == nil && r.config.UseSnapshots {
		// Snapshot creation
		durationSnapshotCreation, err := CreateSnapshot(ctx, r.containerdClient, r.fcctrClient, r.snapshotManager, scs)
		latencyBreakdown.SnapshotCreation = durationpb.New(durationSnapshotCreation)
		if err != nil {
			logrus.WithError(err).Warnf("Failed to create a snapshot for service %s", serviceInfo.Name)
		}
	}

	latencyBreakdown.Total = durationpb.New(time.Since(start))
	return &proto.SandboxCreationStatus{
		Success:          true,
		ID:               scs.SandboxID,
		URL:              url,
		LatencyBreakdown: latencyBreakdown,
	}, nil
}

func (r *Runtime) DeleteSandbox(grpcCtx context.Context, in *proto.SandboxID) (*proto.ActionStatus, error) {
	metadata := r.sandboxManager.DeleteSandbox(in.ID)
	if metadata == nil {
		logrus.Errorf("Error while deleting sandbox from the manager. Invalid name %s.", in.ID)
		return &proto.ActionStatus{Success: false}, nil
	}

	start := time.Now()

	sandboxMetadata := metadata.RuntimeMetadata.(SandboxControlStructure)
	ctx := namespaces.WithNamespace(grpcCtx, namespaceName)

	// destroy the VM
	err := StopVM(ctx, r.fcctrClient, &sandboxMetadata)
	if err != nil {
		logrus.Errorf("Error deleting a sandbox - %v", err)
	}

	r.networkManager.GiveUpNetwork(sandboxMetadata.NetworkConfiguration)

	logrus.Debugf("Sandbox deletion took %d μs", time.Since(start).Microseconds())
	return &proto.ActionStatus{Success: true}, nil
}

func (r *Runtime) CreateTaskSandbox(_ context.Context, _ *proto.WorkflowTaskInfo) (*proto.SandboxCreationStatus, error) {
	// not supported
	return &proto.SandboxCreationStatus{
		Success:          false,
		ID:               "-1",
		LatencyBreakdown: &proto.SandboxCreationBreakdown{},
	}, nil
}

func (r *Runtime) PrepullImage(grpcCtx context.Context, imageInfo *proto.ImageInfo) (*proto.ActionStatus, error) {
	logrus.Debugf("PrepullImage with image = '%s'", imageInfo.URL)

	ctx := namespaces.WithNamespace(grpcCtx, namespaceName)

	_, err, _ := r.imageManager.GetImage(ctx, r.containerdClient, imageInfo.URL)
	if err != nil {
		return &proto.ActionStatus{Success: false}, err
	}

	return &proto.ActionStatus{Success: true}, nil
}

func (r *Runtime) ListEndpoints(grpcCtx context.Context, _ *emptypb.Empty) (*proto.EndpointsList, error) {
	return r.sandboxManager.ListEndpoints()
}

func (r *Runtime) GetImages(grpcCtx context.Context) ([]*proto.ImageInfo, error) {
	ctx := namespaces.WithNamespace(grpcCtx, namespaceName)

	images, err := r.containerdClient.ListImages(ctx)
	if err != nil {
		return []*proto.ImageInfo{}, err
	}

	imageList := make([]*proto.ImageInfo, 0)
	for _, image := range images {
		size, err := image.Size(ctx)
		if err != nil {
			return imageList, err
		}

		imageList = append(imageList, &proto.ImageInfo{
			URL:  image.Name(),
			Size: uint64(size),
		})
	}

	return imageList, nil
}

func (r *Runtime) ValidateHostConfig() bool {
	return true
}

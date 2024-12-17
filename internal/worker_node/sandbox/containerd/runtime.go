package containerd

import (
	"cluster_manager/internal/worker_node/managers"
	"cluster_manager/internal/worker_node/sandbox"
	"cluster_manager/pkg/config"
	"cluster_manager/proto"
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/go-cni"
	"github.com/coreos/go-iptables/iptables"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/emptypb"
)

const (
	bridgeName          = "cni0"
	containerdNamespace = "cm"
)

type Runtime struct {
	sandbox.RuntimeInterface

	cpApi proto.CpiInterfaceClient

	ContainerdClient *containerd.Client
	cniTemplatePath  string
	CNIClient        cni.CNI
	IPT              *iptables.IPTables

	ImageManager   *ImageManager
	SandboxManager *managers.SandboxManager
	ProcessMonitor *managers.ProcessMonitor
	CPUConstraints bool
}

type Metadata struct {
	managers.RuntimeMetadata

	Task      containerd.Task
	Container containerd.Container
}

func NewContainerdRuntime(cpApi proto.CpiInterfaceClient, config config.ContainerdConfig, sandboxManager *managers.SandboxManager, CPUConstraints bool) *Runtime {
	containerdClient := GetContainerdClient(config.CRIPath)

	imageManager := NewContainerdImageManager()
	ipt, err := managers.NewIptablesUtil()

	if err != nil {
		logrus.Fatal("Error while accessing iptables - ", err)
	}

	if config.PrefetchImage {
		ctx := namespaces.WithNamespace(context.Background(), "default")

		_, err, _ = imageManager.GetImage(ctx, containerdClient, "docker.io/cvetkovic/dirigent_empty_function:latest")
		if err != nil {
			logrus.Errorf("Failed to prefetch the image")
		}

		_, err, _ = imageManager.GetImage(ctx, containerdClient, "docker.io/cvetkovic/dirigent_trace_function:latest")
		if err != nil {
			logrus.Errorf("Failed to prefetch the image")
		}
	}

	return &Runtime{
		cpApi: cpApi,

		ContainerdClient: containerdClient,
		cniTemplatePath:  config.CNIConfigPath,
		IPT:              ipt,

		ImageManager:   imageManager,
		SandboxManager: sandboxManager,
		ProcessMonitor: managers.NewProcessMonitor(),
		CPUConstraints: CPUConstraints,
	}
}

func tearDownDefaultBridge() {
	interfaces, err := net.Interfaces()
	if err != nil {
		logrus.Fatal("Failed to list network interfaces on this system.")
	}

	for _, iface := range interfaces {
		if iface.Name == bridgeName {
			err = exec.Command("sudo", "ip", "link", "del", iface.Name).Run()
			if err != nil {
				logrus.Fatalf("Failed to delete network interface %s - %v", iface.Name, err)
			} else {
				logrus.Infof("Successfully deleted network interface %s", iface.Name)
			}

			break
		}
	}
}

func (cr *Runtime) ConfigureNetwork(cidr string) {
	tearDownDefaultBridge()

	rawData, err := os.ReadFile(cr.cniTemplatePath)
	if err != nil {
		logrus.Fatal("Could not read CNI configuration template.")
	}

	finalCNIConfig := strings.Replace(string(rawData), "$SUBNET", cidr, -1)
	finalCNIConfig = strings.Replace(finalCNIConfig, "$BRIDGE_NAME", bridgeName, -1)

	network, err := cni.New(cni.WithConf([]byte(finalCNIConfig)))
	if err != nil {
		logrus.Fatal("Failed to create a CNI client - ", err)
	}

	cr.CNIClient = network
	logrus.Infof("Configured CNI network to use CIDR %s", cidr)
}

func (cr *Runtime) CreateSandbox(grpcCtx context.Context, in *proto.ServiceInfo) (*proto.SandboxCreationStatus, error) {
	logrus.Debug("Create sandbox for service = '", in.Name, "'")

	start := time.Now()

	ctx := namespaces.WithNamespace(grpcCtx, containerdNamespace)
	image, err, durationFetch := cr.ImageManager.GetImage(ctx, cr.ContainerdClient, in.Image)

	if err != nil {
		logrus.Warn("Failed fetching image - ", err)
		return &proto.SandboxCreationStatus{
			Success: false,
			LatencyBreakdown: &proto.SandboxCreationBreakdown{
				Total:      durationpb.New(time.Since(start)),
				ImageFetch: durationpb.New(durationFetch),
			},
		}, err
	}

	container, err, durationContainerCreation := CreateContainer(ctx, cr.ContainerdClient, image, in, cr.CPUConstraints)
	if err != nil {
		logrus.Warn("Failed creating a container - ", err)
		return &proto.SandboxCreationStatus{
			Success: false,
			LatencyBreakdown: &proto.SandboxCreationBreakdown{
				Total:         durationpb.New(time.Since(start)),
				ImageFetch:    durationpb.New(durationFetch),
				SandboxCreate: durationpb.New(durationContainerCreation),
			},
		}, err
	}

	task, _, ip, netNs, err, durationContainerStart, durationCNI := StartContainer(ctx, container, cr.CNIClient)
	if err != nil {
		logrus.Warn("Failed starting a container - ", err)
		return &proto.SandboxCreationStatus{
			Success: false,
			LatencyBreakdown: &proto.SandboxCreationBreakdown{
				Total:         durationpb.New(time.Since(start)),
				ImageFetch:    durationpb.New(durationFetch),
				SandboxCreate: durationpb.New(durationContainerCreation),
				NetworkSetup:  durationpb.New(durationCNI),
				SandboxStart:  durationpb.New(durationContainerStart),
			},
		}, err
	}

	startConfigureMonitoring := time.Now()
	metadata := &managers.Metadata{
		ServiceName: in.Name,

		RuntimeMetadata: Metadata{
			Task:      task,
			Container: container,
		},

		IP:        ip,
		GuestPort: int(in.PortForwarding.GuestPort),
		NetNs:     netNs,

		ExitStatusChannel: make(chan uint32),
	}

	cr.ProcessMonitor.AddChannel(task.Pid(), metadata.ExitStatusChannel)
	cr.SandboxManager.AddSandbox(container.ID(), metadata)
	configureMonitoringDuration := time.Since(startConfigureMonitoring)

	logrus.Debug("Sandbox creation took ", time.Since(start).Microseconds(), " μs (", container.ID(), ")")

	go WatchExitChannel(cr.cpApi, metadata, func(metadata *managers.Metadata) string {
		return metadata.RuntimeMetadata.(Metadata).Container.ID()
	})

	url := fmt.Sprintf("%s:%d", metadata.IP, metadata.GuestPort)

	// blocking call
	timeToPass, passed := managers.SendReadinessProbe(url)

	if passed {
		return &proto.SandboxCreationStatus{
			Success: true,
			ID:      container.ID(),
			URL:     url,
			LatencyBreakdown: &proto.SandboxCreationBreakdown{
				Total:               durationpb.New(time.Since(start)),
				ImageFetch:          durationpb.New(durationFetch),
				SandboxCreate:       durationpb.New(durationContainerCreation),
				NetworkSetup:        durationpb.New(durationCNI),
				SandboxStart:        durationpb.New(durationContainerStart),
				Iptables:            durationpb.New(0),
				ReadinessProbing:    durationpb.New(timeToPass),
				ConfigureMonitoring: durationpb.New(configureMonitoringDuration),
			},
		}, nil
	} else {
		return &proto.SandboxCreationStatus{
			Success: false,
		}, errors.New(fmt.Sprintf("readiness probe failed for %s", container.ID()))
	}
}

func (cr *Runtime) DeleteSandbox(grpcCtx context.Context, in *proto.SandboxID) (*proto.ActionStatus, error) {
	logrus.Debug("RemoveKey sandbox with ID = '", in.ID, "'")
	start := time.Now()

	ctx := namespaces.WithNamespace(grpcCtx, containerdNamespace)
	metadata := cr.SandboxManager.DeleteSandbox(in.ID)

	if metadata == nil {
		logrus.Warn("Tried to delete non-existing sandbox ", in.ID)
		return &proto.ActionStatus{Success: false}, nil
	}

	err := DeleteContainer(ctx, cr.CNIClient, metadata)

	if err != nil {
		logrus.Warn(err)
		return &proto.ActionStatus{Success: false}, err
	}

	logrus.Debug("Sandbox deletion took ", time.Since(start).Microseconds(), " μs")

	return &proto.ActionStatus{Success: true}, nil
}

func (cr *Runtime) CreateTaskSandbox(_ context.Context, _ *proto.WorkflowTaskInfo) (*proto.SandboxCreationStatus, error) {
	// not supported by the dandelion runtime
	return &proto.SandboxCreationStatus{
		Success:          false,
		ID:               "-1",
		LatencyBreakdown: &proto.SandboxCreationBreakdown{},
	}, nil
}

func (cr *Runtime) ListEndpoints(_ context.Context, _ *emptypb.Empty) (*proto.EndpointsList, error) {
	return cr.SandboxManager.ListEndpoints()
}

func (cr *Runtime) PrepullImage(grpcCtx context.Context, imageInfo *proto.ImageInfo) (*proto.ActionStatus, error) {
	logrus.Debugf("PrepullImage with image = '%s'", imageInfo.URL)

	ctx := namespaces.WithNamespace(grpcCtx, containerdNamespace)
	_, err, _ := cr.ImageManager.GetImage(ctx, cr.ContainerdClient, imageInfo.URL)
	if err != nil {
		return &proto.ActionStatus{Success: false}, err
	}
	return &proto.ActionStatus{Success: true}, nil
}

func (cr *Runtime) GetImages(grpcCtx context.Context) ([]*proto.ImageInfo, error) {
	ctx := namespaces.WithNamespace(grpcCtx, containerdNamespace)
	images, err := cr.ContainerdClient.ListImages(ctx)
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

func (cr *Runtime) ValidateHostConfig() bool {
	return true
}

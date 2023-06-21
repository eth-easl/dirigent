package sandbox

import (
	"context"
	"github.com/containerd/containerd"
	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/go-cni"
	"github.com/sirupsen/logrus"
	"log"
	"math/rand"
	"sync"
	"testing"
	"time"
)

const cniConfigPath = "/home/lcvetkovic/projects/cluster_manager/configs/cni.conf"

func TestCreateAContainer(t *testing.T) {
	// fails to expose networking to the container
	rand.Seed(time.Now().UnixNano())

	client, err := containerd.New("/run/containerd/containerd.sock")
	if err != nil {
		log.Fatal("Failed to create a containerd client")
	}

	network, err := cni.New(cni.WithConfFile(cniConfigPath))
	if err != nil {
		logrus.Fatal(err)
	}

	ctx := namespaces.WithNamespace(context.Background(), "default")

	start := time.Now()
	image, _ := FetchImage(ctx, client, "docker.io/cvetkovic/empty_function:latest")
	logrus.Info("Image fetching - ", time.Since(start).Microseconds(), "μs")

	start = time.Now()
	container, err := CreateContainer(ctx, client, image)
	logrus.Info("Create container - ", time.Since(start).Microseconds(), "μs")

	start = time.Now()
	task, exitCh, ip, netns, err := StartContainer(ctx, container, network)
	logrus.Info("Start container - ", time.Since(start).Microseconds(), "μs")

	sm := &Metadata{
		Task:        task,
		Container:   container,
		ExitChannel: exitCh,
		IP:          ip,
		NetNs:       netns,
	}

	_ = DeleteContainer(ctx, network, sm)
}

func TestParallelCreation(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.TextFormatter{TimestampFormat: time.StampMilli, FullTimestamp: true})

	wg := sync.WaitGroup{}

	client, err := containerd.New("/run/containerd/containerd.sock")
	if err != nil {
		log.Fatal("Failed to create a containerd client")
	}

	network, err := cni.New(cni.WithConfFile(cniConfigPath))
	if err != nil {
		logrus.Fatal(err)
	}

	ctx := namespaces.WithNamespace(context.Background(), "default")
	image, _ := FetchImage(ctx, client, "docker.io/cvetkovic/empty_function:latest")

	for i := 0; i < 10; i++ {
		wg.Add(1)

		go func() {
			start := time.Now()

			start = time.Now()
			container, _ := CreateContainer(ctx, client, image)

			start = time.Now()
			task, exitCh, ip, netns, _ := StartContainer(ctx, container, network)

			sm := &Metadata{
				Task:        task,
				Container:   container,
				ExitChannel: exitCh,
				IP:          ip,
				NetNs:       netns,
			}

			logrus.Debug("Sandbox creation took: ", time.Since(start).Milliseconds(), " ms")
			time.Sleep(10 * time.Second)

			_ = DeleteContainer(ctx, network, sm)

			wg.Done()
		}()
	}

	wg.Wait()
}

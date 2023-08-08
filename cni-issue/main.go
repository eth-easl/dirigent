package main

import (
	"cluster_manager/internal/worker_node/sandbox"
	"context"
	"flag"
	"fmt"
	"github.com/containerd/containerd"
	"github.com/containerd/containerd/cio"
	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/containerd/oci"
	"github.com/sirupsen/logrus"
	"github.com/xyproto/randomstring"
	"log"
	"sync"
	"syscall"
	"time"
)

var (
	invocations = flag.Int("invocations", 1, "number of invocations")
)

func GetContainerdClient(containerdSocket string) *containerd.Client {
	client, err := containerd.New(containerdSocket)
	if err != nil {
		log.Fatal("Failed to create a containerd client - ", err)
	}

	return client
}

func main() {
	flag.Parse()
	logrus.Info("Setup")

	imageManager := sandbox.NewImageManager()
	ctx := namespaces.WithNamespace(context.Background(), "default")
	containerdClient := GetContainerdClient("/run/containerd/containerd.sock")

	image, err, _ := imageManager.GetImage(ctx, containerdClient, "docker.io/cvetkovic/empty_function:latest")
	if err != nil {
		logrus.Error(err.Error())
	}

	nbInvocations := *invocations

	fmt.Println(nbInvocations)

	tasks := make([]containerd.Task, 0)
	containers := make([]containerd.Container, 0)

	start := time.Now()

	wg := sync.WaitGroup{}
	wg.Add(nbInvocations)

	lock := sync.Mutex{}

	for i := 0; i < nbInvocations; i++ {
		go func(i int) {
			containerName := randomstring.HumanFriendlyEnglishString(50)
			container, err := containerdClient.NewContainer(ctx, containerName,
				containerd.WithImage(image),
				containerd.WithNewSnapshot(containerName, image),
				containerd.WithNewSpec(oci.WithImageConfig(image)),
			)
			if err != nil {
				logrus.Error(err.Error())
			}

			task, err := container.NewTask(ctx, cio.NewCreator())
			if err != nil {
				logrus.Error(err.Error())
			}

			lock.Lock()
			containers = append(containers, container)
			tasks = append(tasks, task)
			lock.Unlock()

			wg.Done()
		}(i)
	}

	wg.Wait()

	logrus.Info(len(containers))
	logrus.Info(len(tasks))

	elapsed := time.Since(start)
	log.Printf("Creation of the containers took %s", elapsed)

	logrus.Info("Starts")

	start = time.Now()

	{
		wg := sync.WaitGroup{}
		wg.Add(nbInvocations)
		for i := 0; i < nbInvocations; i++ {
			go func(i int) {
				cniClient := sandbox.GetCNIClient("../configs/cni.conf")
				netns := fmt.Sprintf("/proc/%v/ns/net", tasks[i].Pid())

				_, err = cniClient.Setup(ctx, containers[i].ID(), netns)
				if err != nil {
					logrus.Error(err.Error())
				}

				wg.Done()
			}(i)
		}
		wg.Wait()
	}

	elapsed = time.Since(start)
	log.Printf("Creation of the network took %s", elapsed)

	fmt.Println(len(containers))

	{
		wg := sync.WaitGroup{}
		wg.Add(nbInvocations)
		for i := 0; i < nbInvocations; i++ {
			go func(i int) {
				cniClient := sandbox.GetCNIClient("../configs/cni.conf")
				netns := fmt.Sprintf("/proc/%v/ns/net", tasks[i].Pid())
				if err := cniClient.Remove(ctx, containers[i].ID(), netns); err != nil {
					logrus.Error(err.Error())
				}

				if err := tasks[i].Kill(ctx, syscall.SIGKILL, containerd.WithKillAll); err != nil {
					logrus.Error(err.Error())
				}

				if _, err := tasks[i].Delete(ctx, containerd.WithProcessKill); err != nil {
					logrus.Error(err.Error())
				}

				if err := containers[i].Delete(ctx, containerd.WithSnapshotCleanup); err != nil {
					logrus.Error(err.Error())
				}

				wg.Done()
			}(i)
		}
		wg.Wait()
	}

	log.Println("We are done")
}

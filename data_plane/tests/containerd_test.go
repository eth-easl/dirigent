package tests

import (
	"cluster_manager/common"
	"context"
	"github.com/containerd/containerd"
	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/containerd/third_party/k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
	"github.com/sirupsen/logrus"
	"log"
	"math/rand"
	"testing"
	"time"
)

func TestCreateAContainer(t *testing.T) {
	// fails to expose networking to the container
	t.Skip()

	rand.Seed(time.Now().UnixNano())

	client, err := containerd.New("/run/containerd/containerd.sock")
	if err != nil {
		log.Fatal("Failed to create a containerd client")
	}

	ctx := namespaces.WithNamespace(context.Background(), "cm")
	image := common.FetchImage(ctx, client, "docker.io/cvetkovic/empty_function:latest")
	container := common.CreateContainerOld(ctx, client, "test-cnt", image)
	task, exitCh := common.StartContainer(ctx, container)
	common.DeleteContainer(ctx, task, exitCh)
}

func TestContainerCreate(t *testing.T) {
	// fails to create a pod because CNI does not exist
	t.Skip()

	runtimeClient := common.CreateRuntimeClient()
	imageClient := common.CreateImageClient()

	//podSandboxID := common.CreateSandbox(runtimeClient)
	common.CreateContainer(runtimeClient, imageClient, "")

	_, err := runtimeClient.RemovePodSandbox(context.Background(), &v1alpha2.RemovePodSandboxRequest{PodSandboxId: "934d01da1f149ef3aa416cb76a1dda68ad38325448ee1c9960b7a86d3ce3aca1"})
	logrus.Warn(err)
}

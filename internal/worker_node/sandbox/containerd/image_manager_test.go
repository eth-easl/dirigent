package containerd

import (
	"cluster_manager/pkg/utils"
	"context"
	"github.com/containerd/containerd/namespaces"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestImageManager(t *testing.T) {
	imageManager := NewImageManager()

	containerdClient := GetContainerdClient("/run/containerd/containerd.sock")
	defer containerdClient.Close()

	_, err, _ := imageManager.GetImage(namespaces.WithNamespace(context.Background(), "default"), containerdClient, utils.TestDockerImageName)
	assert.NoError(t, err, "Image fetching should return no error")
}

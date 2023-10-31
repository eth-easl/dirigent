package containerd

import (
	"cluster_manager/pkg/utils"
	"context"
	"github.com/containerd/containerd/namespaces"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestImageManager(t *testing.T) {
	t.Skip()

	imageManager := NewContainerdImageManager()

	containerdClient := GetContainerdClient("/run/containerd/containerd.sock")
	defer containerdClient.Close()

	_, err, _ := imageManager.GetImage(namespaces.WithNamespace(context.Background(), "default"), containerdClient, utils.TestDockerImageName)
	assert.NoError(t, err, "Image fetching should return no error")
}

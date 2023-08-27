package worker_node

import (
	"cluster_manager/internal/worker_node/sandbox"
	"cluster_manager/pkg/config"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"testing"
)

func TestCreationWorkerNode(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConfig := config.WorkerNodeConfig{
		ControlPlaneIp:   "",
		ControlPlanePort: "",
		Port:             0,
		Verbosity:        "",
		CRIPath:          "/run/containerd/containerd.sock",
		CNIConfigPath:    "../../configs/cni.conf",
		PrefetchImage:    false,
	}

	containerdClient := sandbox.GetContainerdClient(mockConfig.CRIPath)
	defer containerdClient.Close()

	assert.NotNil(t, NewWorkerNode(mockConfig, containerdClient), "Created worker not should not be nil")
}

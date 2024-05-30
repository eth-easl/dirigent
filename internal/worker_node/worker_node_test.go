package worker_node

import (
	"cluster_manager/internal/worker_node/sandbox/containerd"
	"cluster_manager/pkg/config"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"testing"
)

func TestCreationWorkerNode(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConfig := config.WorkerNodeConfig{
		ControlPlaneAddress: []string{},
		Port:                0,
		Verbosity:           "",
		CRIType:             "containerd",
		Containerd: config.ContainerdConfig{
			CRIPath:       "/run/containerd/containerd.sock",
			CNIConfigPath: "../../configs/cni.conf",
			PrefetchImage: false,
		},
	}

	containerdClient := containerd.GetContainerdClient(mockConfig.Containerd.CRIPath)
	defer containerdClient.Close()

	assert.NotNil(t, NewWorkerNode(nil, mockConfig), "Created worker not should not be nil")
}

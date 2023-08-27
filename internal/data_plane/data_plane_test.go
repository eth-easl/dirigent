package data_plane

import (
	"cluster_manager/internal/data_plane/function_metadata"
	"cluster_manager/pkg/config"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"testing"
)

func TestCreationWorkerNode(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConfig := config.DataPlaneConfig{
		ControlPlaneIp:      "",
		ControlPlanePort:    "",
		PortProxy:           "",
		PortGRPC:            "",
		Verbosity:           "",
		TraceOutputFolder:   "",
		LoadBalancingPolicy: "",
	}

	cache := function_metadata.NewDeploymentList()
	assert.NotNil(t, NewDataplane(mockConfig, cache), "Generated dataplane should not be nil")
}

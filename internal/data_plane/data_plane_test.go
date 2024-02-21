package data_plane

import (
	"cluster_manager/pkg/config"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"testing"
)

func TestCreationWorkerNode(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConfig := config.DataPlaneConfig{
		ControlPlaneAddress: []string{""},
		PortProxy:           "",
		PortGRPC:            "",
		Verbosity:           "",
		TraceOutputFolder:   "",
		LoadBalancingPolicy: "",
	}

	assert.NotNil(t, NewDataplane(mockConfig), "Generated dataplane should not be nil")
}

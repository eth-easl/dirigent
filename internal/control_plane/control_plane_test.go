package control_plane

import (
	"cluster_manager/api/proto"
	mock_persistence "cluster_manager/mock"
	"cluster_manager/pkg/config"
	"context"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"testing"
	"time"
)

func TestCreationControlPlane(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConfig := config.ControlPlaneConfig{
		Port:              "",
		PortRegistration:  "",
		Verbosity:         "",
		TraceOutputFolder: "",
		PlacementPolicy:   "",
		Persistence:       false,
		Profiler:          config.ProfilerConfig{},
		RedisConf:         config.RedisConf{},
		Reconstruct:       true,
	}

	persistenceLayer := mock_persistence.NewMockPersistenceLayer(ctrl)

	persistenceLayer.EXPECT().GetWorkerNodeInformation(gomock.Any()).DoAndReturn(func(_ context.Context) ([]*proto.WorkerNodeInformation, error) {
		return make([]*proto.WorkerNodeInformation, 0), nil
	}).Times(1)

	persistenceLayer.EXPECT().GetServiceInformation(gomock.Any()).DoAndReturn(func(_ context.Context) ([]*proto.ServiceInfo, error) {
		return make([]*proto.ServiceInfo, 0), nil
	}).Times(1)

	persistenceLayer.EXPECT().GetDataPlaneInformation(gomock.Any()).DoAndReturn(func(_ context.Context) ([]*proto.DataplaneInformation, error) {
		return make([]*proto.DataplaneInformation, 0), nil
	}).Times(1)

	persistenceLayer.EXPECT().GetEndpoints(gomock.Any()).DoAndReturn(func(_ context.Context) ([]*proto.Endpoint, []string, error) {
		return make([]*proto.Endpoint, 0), make([]string, 0), nil
	}).Times(1)

	controlPlane := NewControlPlane(persistenceLayer, "", NewRandomPolicy())

	start := time.Now()
	controlPlane.ReconstructState(context.Background(), mockConfig)
	elapsed := time.Since(start)

	assert.Zero(t, controlPlane.GetNumberConnectedWorkers(), "Number of connected workers should be 0")
	assert.Zero(t, controlPlane.GetNumberDataplanes(), "Number of connected dataplanes should be 0")
	assert.Zero(t, controlPlane.GetNumberServices(), "Number of registered services should be 0")

	logrus.Infof("Took %s seconds to reconstruct", elapsed)
}

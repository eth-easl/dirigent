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

var mockConfig config.ControlPlaneConfig = config.ControlPlaneConfig{
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

func TestCreationControlPlaneEmpty(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

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

	controlPlane := NewControlPlane(persistenceLayer, "", NewRandomPolicy())

	start := time.Now()
	err := controlPlane.ReconstructState(context.Background(), mockConfig)
	elapsed := time.Since(start)

	assert.NoError(t, err, "reconstructing control plane state failed")

	assert.Zero(t, controlPlane.GetNumberConnectedWorkers(), "Number of connected workers should be 0")
	assert.Zero(t, controlPlane.GetNumberDataplanes(), "Number of connected data planes should be 0")
	assert.Zero(t, controlPlane.GetNumberServices(), "Number of registered services should be 0")

	logrus.Infof("Took %s seconds to reconstruct", elapsed)
}

func TestCreationControlPlaneWith5Services(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	persistenceLayer := mock_persistence.NewMockPersistenceLayer(ctrl)

	persistenceLayer.EXPECT().GetWorkerNodeInformation(gomock.Any()).DoAndReturn(func(_ context.Context) ([]*proto.WorkerNodeInformation, error) {
		return make([]*proto.WorkerNodeInformation, 0), nil
	}).Times(1)

	persistenceLayer.EXPECT().GetServiceInformation(gomock.Any()).DoAndReturn(func(_ context.Context) ([]*proto.ServiceInfo, error) {
		return []*proto.ServiceInfo{
			&proto.ServiceInfo{
				Name:              "1",
				Image:             "",
				PortForwarding:    nil,
				AutoscalingConfig: nil,
			},
			&proto.ServiceInfo{
				Name:              "2",
				Image:             "",
				PortForwarding:    nil,
				AutoscalingConfig: nil,
			},
			&proto.ServiceInfo{
				Name:              "3",
				Image:             "",
				PortForwarding:    nil,
				AutoscalingConfig: nil,
			},
			&proto.ServiceInfo{
				Name:              "4",
				Image:             "",
				PortForwarding:    nil,
				AutoscalingConfig: nil,
			},
			&proto.ServiceInfo{
				Name:              "5",
				Image:             "",
				PortForwarding:    nil,
				AutoscalingConfig: nil,
			},
		}, nil
	}).Times(1)

	persistenceLayer.EXPECT().GetDataPlaneInformation(gomock.Any()).DoAndReturn(func(_ context.Context) ([]*proto.DataplaneInformation, error) {
		return make([]*proto.DataplaneInformation, 0), nil
	}).Times(1)

	controlPlane := NewControlPlane(persistenceLayer, "", NewRandomPolicy())

	start := time.Now()
	err := controlPlane.ReconstructState(context.Background(), mockConfig)
	elapsed := time.Since(start)

	assert.NoError(t, err, "reconstructing control plane state failed")

	assert.Zero(t, controlPlane.GetNumberConnectedWorkers(), "Number of connected workers should be 0")
	assert.Zero(t, controlPlane.GetNumberDataplanes(), "Number of connected data planes should be 0")
	assert.Equal(t, 5, controlPlane.GetNumberServices(), "Number of registered services should be 0")

	logrus.Infof("Took %s seconds to reconstruct", elapsed)
}

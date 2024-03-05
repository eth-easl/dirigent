package control_plane

import (
	"cluster_manager/api/proto"
	"cluster_manager/internal/control_plane/autoscaling"
	"cluster_manager/internal/control_plane/core"
	"cluster_manager/internal/control_plane/data_plane"
	"cluster_manager/internal/control_plane/data_plane/empty_dataplane"
	"cluster_manager/internal/control_plane/placement_policy"
	"cluster_manager/internal/control_plane/workers"
	"cluster_manager/internal/control_plane/workers/empty_worker"
	"cluster_manager/mock/mock_core"
	"cluster_manager/mock/mock_persistence"
	"cluster_manager/pkg/config"
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"sync"
	"testing"
	"time"
)

var mockConfig = config.ControlPlaneConfig{
	Port:              "",
	Verbosity:         "",
	TraceOutputFolder: "",
	PlacementPolicy:   "",
	Persistence:       false,
	Profiler:          config.ProfilerConfig{},
	RedisConf:         config.RedisConf{},
	Reconstruct:       true,
}

// Smoke test

func simplePersistenceLayer(t *testing.T) (*mock_persistence.MockPersistenceLayer, *gomock.Controller) {
	ctrl := gomock.NewController(t)

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

	persistenceLayer.EXPECT().SetLeader(gomock.Any()).DoAndReturn(func(_ context.Context) error {
		return nil
	}).Times(1)

	return persistenceLayer, ctrl
}

func TestCreationControlPlaneEmpty(t *testing.T) {
	logrus.SetLevel(logrus.FatalLevel)

	persistenceLayer, ctrl := simplePersistenceLayer(t)
	defer ctrl.Finish()

	controlPlane := NewControlPlane(persistenceLayer, "", placement_policy.NewRandomPlacement(), data_plane.NewDataplaneConnection, workers.NewWorkerNode, &mockConfig)

	err := controlPlane.ReconstructState(context.Background(), mockConfig)

	assert.NoError(t, err, "reconstructing control plane state failed")

	assert.Zero(t, controlPlane.GetNumberConnectedWorkers(), "Number of connected workers should be 0")
	assert.Zero(t, controlPlane.GetNumberDataplanes(), "Number of connected data planes should be 0")
	assert.Zero(t, controlPlane.GetNumberServices(), "Number of registered services should be 0")
}

// Registration tests

func TestRegisterWorker(t *testing.T) {
	logrus.SetLevel(logrus.FatalLevel)

	persistenceLayer, ctrl := simplePersistenceLayer(t)
	defer ctrl.Finish()

	nbRegistrations := 100

	persistenceLayer.EXPECT().StoreWorkerNodeInformation(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, _ *proto.WorkerNodeInformation, _ time.Time) error {
		return nil
	}).Times(nbRegistrations)

	controlPlane := NewControlPlane(persistenceLayer, "", placement_policy.NewRandomPlacement(), data_plane.NewDataplaneConnection, workers.NewWorkerNode, &mockConfig)

	err := controlPlane.ReconstructState(context.Background(), mockConfig)

	assert.NoError(t, err, "reconstructing control plane state failed")

	assert.Zero(t, controlPlane.GetNumberConnectedWorkers(), "Number of connected workers should be 0")
	assert.Zero(t, controlPlane.GetNumberDataplanes(), "Number of connected data planes should be 0")
	assert.Zero(t, controlPlane.GetNumberServices(), "Number of registered services should be 0")

	for i := 1; i <= nbRegistrations; i++ {
		name := "mock" + fmt.Sprint(i)

		status, err := controlPlane.RegisterNode(context.Background(), &proto.NodeInfo{
			NodeID: name,
		})

		assert.True(t, status.Success, "status should be successful")
		assert.NoError(t, err, "error should not be nil")

		assert.Equal(t, i, controlPlane.GetNumberConnectedWorkers())

		status, err = controlPlane.RegisterNode(context.Background(), &proto.NodeInfo{
			NodeID: name,
		})

		assert.False(t, status.Success, "status should be unsuccessful")
		assert.NoError(t, err, "error should not be nil")

		assert.Equal(t, i, controlPlane.GetNumberConnectedWorkers())
	}
}

func TestRegisterDataplanes(t *testing.T) {
	logrus.SetLevel(logrus.FatalLevel)

	persistenceLayer, ctrl := simplePersistenceLayer(t)
	defer ctrl.Finish()

	nbRegistrations := 100

	persistenceLayer.EXPECT().StoreDataPlaneInformation(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, _ *proto.DataplaneInformation, _ time.Time) error {
		return nil
	}).Times(nbRegistrations)

	controlPlane := NewControlPlane(persistenceLayer, "", placement_policy.NewRandomPlacement(), empty_dataplane.NewDataplaneConnectionEmpty, empty_worker.NewEmptyWorkerNode, &mockConfig)

	err := controlPlane.ReconstructState(context.Background(), mockConfig)

	assert.NoError(t, err, "reconstructing control plane state failed")

	assert.Zero(t, controlPlane.GetNumberConnectedWorkers(), "Number of connected workers should be 0")
	assert.Zero(t, controlPlane.GetNumberDataplanes(), "Number of connected data planes should be 0")
	assert.Zero(t, controlPlane.GetNumberServices(), "Number of registered services should be 0")

	for i := 1; i <= nbRegistrations; i++ {
		name := "mock" + fmt.Sprint(i)

		status, err, _ := controlPlane.RegisterDataplane(context.Background(), &proto.DataplaneInfo{
			IP:        name,
			APIPort:   0,
			ProxyPort: 0,
		})

		assert.True(t, status.Success, "status should be successful")
		assert.NoError(t, err, "error should not be nil")

		assert.Equal(t, i, controlPlane.GetNumberDataplanes())

		status, err, _ = controlPlane.RegisterDataplane(context.Background(), &proto.DataplaneInfo{
			IP:        name,
			APIPort:   0,
			ProxyPort: 0,
		})

		assert.True(t, status.Success, "status should be unsuccessful")
		assert.NoError(t, err, "error should not be nil")

		assert.Equal(t, i, controlPlane.GetNumberDataplanes())
	}
}

func TestRegisterServices(t *testing.T) {
	logrus.SetLevel(logrus.FatalLevel)

	persistenceLayer, ctrl := simplePersistenceLayer(t)
	defer ctrl.Finish()

	nbRegistrations := 100

	persistenceLayer.EXPECT().StoreServiceInformation(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, _ *proto.ServiceInfo, _ time.Time) error {
		return nil
	}).Times(nbRegistrations)

	controlPlane := NewControlPlane(persistenceLayer, "", placement_policy.NewRandomPlacement(), empty_dataplane.NewDataplaneConnectionEmpty, empty_worker.NewEmptyWorkerNode, &mockConfig)

	err := controlPlane.ReconstructState(context.Background(), mockConfig)

	assert.NoError(t, err, "reconstructing control plane state failed")

	assert.Zero(t, controlPlane.GetNumberConnectedWorkers(), "Number of connected workers should be 0")
	assert.Zero(t, controlPlane.GetNumberDataplanes(), "Number of connected data planes should be 0")
	assert.Zero(t, controlPlane.GetNumberServices(), "Number of registered services should be 0")

	for i := 1; i <= nbRegistrations; i++ {
		name := "mock" + fmt.Sprint(i)

		status, err := controlPlane.RegisterService(context.Background(), &proto.ServiceInfo{
			Name:              name,
			Image:             "",
			PortForwarding:    nil,
			AutoscalingConfig: nil,
		})

		assert.True(t, status.Success, "status should be successful")
		assert.NoError(t, err, "error should not be nil")

		assert.Equal(t, i, controlPlane.GetNumberServices())

		status, err = controlPlane.RegisterService(context.Background(), &proto.ServiceInfo{
			Name:              name,
			Image:             "",
			PortForwarding:    nil,
			AutoscalingConfig: nil,
		})

		assert.False(t, status.Success, "status should be unsuccessful")
		assert.Error(t, err, "error should not be nil")

		assert.Equal(t, i, controlPlane.GetNumberServices())
	}
}

// Deregistration tests

func TestDeregisterWorker(t *testing.T) {
	logrus.SetLevel(logrus.FatalLevel)

	persistenceLayer, ctrl := simplePersistenceLayer(t)
	defer ctrl.Finish()

	nbRegistrations := 100

	persistenceLayer.EXPECT().StoreWorkerNodeInformation(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, _ *proto.WorkerNodeInformation, _ time.Time) error {
		return nil
	}).Times(nbRegistrations)

	persistenceLayer.EXPECT().DeleteWorkerNodeInformation(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, _ string, _ time.Time) error {
		return nil
	}).Times(nbRegistrations)

	controlPlane := NewControlPlane(persistenceLayer, "", placement_policy.NewRandomPlacement(), data_plane.NewDataplaneConnection, workers.NewWorkerNode, &mockConfig)

	err := controlPlane.ReconstructState(context.Background(), mockConfig)

	assert.NoError(t, err, "reconstructing control plane state failed")

	assert.Zero(t, controlPlane.GetNumberConnectedWorkers(), "Number of connected workers should be 0")
	assert.Zero(t, controlPlane.GetNumberDataplanes(), "Number of connected data planes should be 0")
	assert.Zero(t, controlPlane.GetNumberServices(), "Number of registered services should be 0")

	for i := 1; i <= nbRegistrations; i++ {
		name := "mock" + fmt.Sprint(i)

		status, err := controlPlane.RegisterNode(context.Background(), &proto.NodeInfo{
			NodeID: name,
		})

		assert.True(t, status.Success, "status should be successful")
		assert.NoError(t, err, "error should not be nil")

		assert.Equal(t, i, controlPlane.GetNumberConnectedWorkers())

		status, err = controlPlane.RegisterNode(context.Background(), &proto.NodeInfo{
			NodeID: name,
		})

		assert.False(t, status.Success, "status should be unsuccessful")
		assert.NoError(t, err, "error should not be nil")

		assert.Equal(t, i, controlPlane.GetNumberConnectedWorkers())
	}

	for i := nbRegistrations; i >= 1; i-- {
		name := "mock" + fmt.Sprint(i)

		status, err := controlPlane.DeregisterNode(context.Background(), &proto.NodeInfo{
			NodeID: name,
		})

		assert.True(t, status.Success, "status should be successful")
		assert.NoError(t, err, "error should not be nil")

		assert.Equal(t, i-1, controlPlane.GetNumberConnectedWorkers())

		status, err = controlPlane.DeregisterNode(context.Background(), &proto.NodeInfo{
			NodeID: name,
		})

		assert.False(t, status.Success, "status should be unsuccessful")
		assert.NoError(t, err, "error should not be nil")

		assert.Equal(t, i-1, controlPlane.GetNumberConnectedWorkers())
	}
}

func TestDeregisterDataplanes(t *testing.T) {
	logrus.SetLevel(logrus.FatalLevel)

	persistenceLayer, ctrl := simplePersistenceLayer(t)
	defer ctrl.Finish()

	nbRegistrations := 1

	persistenceLayer.EXPECT().StoreDataPlaneInformation(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, _ *proto.DataplaneInformation, _ time.Time) error {
		return nil
	}).Times(nbRegistrations)

	persistenceLayer.EXPECT().DeleteDataPlaneInformation(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, _ *proto.DataplaneInformation, _ time.Time) error {
		return nil
	}).Times(nbRegistrations)

	controlPlane := NewControlPlane(persistenceLayer, "", placement_policy.NewRandomPlacement(), empty_dataplane.NewDataplaneConnectionEmpty, empty_worker.NewEmptyWorkerNode, &mockConfig)

	err := controlPlane.ReconstructState(context.Background(), mockConfig)

	assert.NoError(t, err, "reconstructing control plane state failed")

	assert.Zero(t, controlPlane.GetNumberConnectedWorkers(), "Number of connected workers should be 0")
	assert.Zero(t, controlPlane.GetNumberDataplanes(), "Number of connected data planes should be 0")
	assert.Zero(t, controlPlane.GetNumberServices(), "Number of registered services should be 0")

	for i := 1; i <= nbRegistrations; i++ {
		name := "mock" + fmt.Sprint(i)

		status, err, _ := controlPlane.RegisterDataplane(context.Background(), &proto.DataplaneInfo{
			IP:        name,
			APIPort:   0,
			ProxyPort: 0,
		})

		assert.True(t, status.Success, "status should be successful")
		assert.NoError(t, err, "error should not be nil")

		assert.Equal(t, i, controlPlane.GetNumberDataplanes())

		status, err, _ = controlPlane.RegisterDataplane(context.Background(), &proto.DataplaneInfo{
			IP:        name,
			APIPort:   0,
			ProxyPort: 0,
		})

		assert.True(t, status.Success, "status should be unsuccessful")
		assert.NoError(t, err, "error should not be nil")

		assert.Equal(t, i, controlPlane.GetNumberDataplanes())
	}

	for i := nbRegistrations; i >= 1; i-- {
		name := "mock" + fmt.Sprint(i)

		status, err := controlPlane.DeregisterDataplane(context.Background(), &proto.DataplaneInfo{
			IP:        name,
			APIPort:   0,
			ProxyPort: 0,
		})

		assert.True(t, status.Success, "status should be successful")
		assert.NoError(t, err, "error should not be nil")

		assert.Equal(t, i-1, controlPlane.GetNumberDataplanes())

		status, err = controlPlane.DeregisterDataplane(context.Background(), &proto.DataplaneInfo{
			IP:        name,
			APIPort:   0,
			ProxyPort: 0,
		})

		assert.True(t, status.Success, "status should be unsuccessful")
		assert.NoError(t, err, "error should not be nil")

		assert.Equal(t, i-1, controlPlane.GetNumberDataplanes())
	}
}

func TestDeregisterServices(t *testing.T) {
	logrus.SetLevel(logrus.FatalLevel)

	persistenceLayer, ctrl := simplePersistenceLayer(t)
	defer ctrl.Finish()

	nbRegistrations := 100

	persistenceLayer.EXPECT().StoreServiceInformation(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, _ *proto.ServiceInfo, _ time.Time) error {
		return nil
	}).Times(nbRegistrations)

	persistenceLayer.EXPECT().DeleteServiceInformation(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, _ *proto.ServiceInfo, _ time.Time) error {
		return nil
	}).Times(nbRegistrations)

	controlPlane := NewControlPlane(persistenceLayer, "", placement_policy.NewRandomPlacement(), empty_dataplane.NewDataplaneConnectionEmpty, empty_worker.NewEmptyWorkerNode, &mockConfig)

	err := controlPlane.ReconstructState(context.Background(), mockConfig)

	assert.NoError(t, err, "reconstructing control plane state failed")

	assert.Zero(t, controlPlane.GetNumberConnectedWorkers(), "Number of connected workers should be 0")
	assert.Zero(t, controlPlane.GetNumberDataplanes(), "Number of connected data planes should be 0")
	assert.Zero(t, controlPlane.GetNumberServices(), "Number of registered services should be 0")

	for i := 1; i <= nbRegistrations; i++ {
		name := "mock" + fmt.Sprint(i)

		status, err := controlPlane.RegisterService(context.Background(), &proto.ServiceInfo{
			Name:              name,
			Image:             "",
			PortForwarding:    nil,
			AutoscalingConfig: nil,
		})

		assert.True(t, status.Success, "status should be successful")
		assert.NoError(t, err, "error should not be nil")

		assert.Equal(t, i, controlPlane.GetNumberServices())

		status, err = controlPlane.RegisterService(context.Background(), &proto.ServiceInfo{
			Name:              name,
			Image:             "",
			PortForwarding:    nil,
			AutoscalingConfig: nil,
		})

		assert.False(t, status.Success, "status should be unsuccessful")
		assert.Error(t, err, "error should not be nil")

		assert.Equal(t, i, controlPlane.GetNumberServices())
	}

	for i := nbRegistrations; i >= 1; i-- {
		name := "mock" + fmt.Sprint(i)

		status, err := controlPlane.DeregisterService(context.Background(), &proto.ServiceInfo{
			Name:              name,
			Image:             "",
			PortForwarding:    nil,
			AutoscalingConfig: nil,
		})

		assert.True(t, status.Success, "status should be successful")
		assert.NoError(t, err, "error should not be nil")

		assert.Equal(t, i-1, controlPlane.GetNumberServices())

		status, err = controlPlane.DeregisterService(context.Background(), &proto.ServiceInfo{
			Name:              name,
			Image:             "",
			PortForwarding:    nil,
			AutoscalingConfig: nil,
		})

		assert.False(t, status.Success, "status should be unsuccessful")
		assert.Error(t, err, "error should not be nil")

		assert.Equal(t, i-1, controlPlane.GetNumberServices())
	}

	time.Sleep(500 * time.Millisecond)
}

// Reconstruction tests

func TestReconstructionService(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	logrus.SetLevel(logrus.FatalLevel)

	persistenceLayer := mock_persistence.NewMockPersistenceLayer(ctrl)

	persistenceLayer.EXPECT().GetWorkerNodeInformation(gomock.Any()).DoAndReturn(func(_ context.Context) ([]*proto.WorkerNodeInformation, error) {
		return make([]*proto.WorkerNodeInformation, 0), nil
	}).Times(1)

	persistenceLayer.EXPECT().GetServiceInformation(gomock.Any()).DoAndReturn(func(_ context.Context) ([]*proto.ServiceInfo, error) {
		return []*proto.ServiceInfo{
			{
				Name:              "1",
				Image:             "",
				PortForwarding:    nil,
				AutoscalingConfig: nil,
			},
			{
				Name:              "2",
				Image:             "",
				PortForwarding:    nil,
				AutoscalingConfig: nil,
			},
			{
				Name:              "3",
				Image:             "",
				PortForwarding:    nil,
				AutoscalingConfig: nil,
			},
			{
				Name:              "4",
				Image:             "",
				PortForwarding:    nil,
				AutoscalingConfig: nil,
			},
			{
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

	persistenceLayer.EXPECT().SetLeader(gomock.Any()).DoAndReturn(func(_ context.Context) error {
		return nil
	}).Times(1)

	controlPlane := NewControlPlane(persistenceLayer, "", placement_policy.NewRandomPlacement(), data_plane.NewDataplaneConnection, workers.NewWorkerNode, &mockConfig)

	start := time.Now()
	err := controlPlane.ReconstructState(context.Background(), mockConfig)
	elapsed := time.Since(start)

	assert.NoError(t, err, "reconstructing control plane state failed")

	assert.Zero(t, controlPlane.GetNumberConnectedWorkers(), "Number of connected workers should be 0")
	assert.Zero(t, controlPlane.GetNumberDataplanes(), "Number of connected data planes should be 0")
	assert.Equal(t, 5, controlPlane.GetNumberServices(), "Number of registered services should be 0")

	logrus.Infof("Took %s seconds to reconstruct", elapsed)
}

func TestReconstructionWorkers(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	logrus.SetLevel(logrus.FatalLevel)

	te := testStructure{ctrl: ctrl}

	persistenceLayer := mock_persistence.NewMockPersistenceLayer(ctrl)

	persistenceLayer.EXPECT().GetWorkerNodeInformation(gomock.Any()).DoAndReturn(func(_ context.Context) ([]*proto.WorkerNodeInformation, error) {
		wi := make([]*proto.WorkerNodeInformation, 0)

		wi = append(wi, &proto.WorkerNodeInformation{
			Name:     "w1",
			Ip:       "",
			Port:     "",
			CpuCores: 0,
			Memory:   0,
		})

		wi = append(wi, &proto.WorkerNodeInformation{
			Name:     "w2",
			Ip:       "",
			Port:     "",
			CpuCores: 0,
			Memory:   0,
		})

		wi = append(wi, &proto.WorkerNodeInformation{
			Name:     "w3",
			Ip:       "",
			Port:     "",
			CpuCores: 0,
			Memory:   0,
		})

		return wi, nil
	}).Times(1)

	persistenceLayer.EXPECT().GetServiceInformation(gomock.Any()).DoAndReturn(func(_ context.Context) ([]*proto.ServiceInfo, error) {
		return make([]*proto.ServiceInfo, 0), nil
	}).Times(1)

	persistenceLayer.EXPECT().GetDataPlaneInformation(gomock.Any()).DoAndReturn(func(_ context.Context) ([]*proto.DataplaneInformation, error) {
		return make([]*proto.DataplaneInformation, 0), nil
	}).Times(1)

	persistenceLayer.EXPECT().SetLeader(gomock.Any()).DoAndReturn(func(_ context.Context) error {
		return nil
	}).Times(1)

	controlPlane := NewControlPlane(persistenceLayer, "", placement_policy.NewRandomPlacement(), te.NewMockDataplaneConnection, te.NewMockWorkerConnection, &mockConfig)

	start := time.Now()
	err := controlPlane.ReconstructState(context.Background(), mockConfig)
	elapsed := time.Since(start)

	assert.NoError(t, err, "reconstructing control plane state failed")

	assert.Equal(t, 0, controlPlane.GetNumberConnectedWorkers(), "Number of connected workers should be 0")
	assert.Zero(t, controlPlane.GetNumberDataplanes(), "Number of connected data planes should be 0")
	assert.Zero(t, controlPlane.GetNumberServices(), "Number of registered services should be 0")

	logrus.Infof("Took %s seconds to reconstruct", elapsed)
}

func TestReconstructionDataplanes(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	logrus.SetLevel(logrus.FatalLevel)

	te := testStructure{ctrl: ctrl}

	persistenceLayer := mock_persistence.NewMockPersistenceLayer(ctrl)

	persistenceLayer.EXPECT().GetWorkerNodeInformation(gomock.Any()).DoAndReturn(func(_ context.Context) ([]*proto.WorkerNodeInformation, error) {
		return make([]*proto.WorkerNodeInformation, 0), nil
	}).Times(1)

	persistenceLayer.EXPECT().GetServiceInformation(gomock.Any()).DoAndReturn(func(_ context.Context) ([]*proto.ServiceInfo, error) {
		return make([]*proto.ServiceInfo, 0), nil
	}).Times(1)

	persistenceLayer.EXPECT().GetDataPlaneInformation(gomock.Any()).DoAndReturn(func(_ context.Context) ([]*proto.DataplaneInformation, error) {
		arr := make([]*proto.DataplaneInformation, 0)

		arr = append(arr, &proto.DataplaneInformation{
			Address:   "A1",
			ApiPort:   "",
			ProxyPort: "",
		})

		arr = append(arr, &proto.DataplaneInformation{
			Address:   "A2",
			ApiPort:   "",
			ProxyPort: "",
		})

		arr = append(arr, &proto.DataplaneInformation{
			Address:   "A3",
			ApiPort:   "",
			ProxyPort: "",
		})

		return arr, nil
	}).Times(1)

	persistenceLayer.EXPECT().SetLeader(gomock.Any()).DoAndReturn(func(_ context.Context) error {
		return nil
	}).Times(1)

	controlPlane := NewControlPlane(persistenceLayer, "", placement_policy.NewRandomPlacement(), te.NewMockDataplaneConnection, workers.NewWorkerNode, &mockConfig)

	start := time.Now()
	err := controlPlane.ReconstructState(context.Background(), mockConfig)
	elapsed := time.Since(start)

	assert.NoError(t, err, "reconstructing control plane state failed")

	assert.Zero(t, controlPlane.GetNumberConnectedWorkers(), "Number of connected workers should be 0")
	assert.Equal(t, 3, controlPlane.GetNumberDataplanes(), "Number of connected data planes should be 3")
	assert.Zero(t, controlPlane.GetNumberServices(), "Number of registered services should be 0")

	logrus.Infof("Took %s seconds to reconstruct", elapsed)
}

// Concurrency tests

func TestStressRegisterServices(t *testing.T) {
	logrus.SetLevel(logrus.FatalLevel)

	persistenceLayer, ctrl := simplePersistenceLayer(t)
	defer ctrl.Finish()

	size := 100

	persistenceLayer.EXPECT().StoreServiceInformation(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, _ *proto.ServiceInfo, _ time.Time) error {
		return nil
	}).Times(size)

	controlPlane := NewControlPlane(persistenceLayer, "", placement_policy.NewRandomPlacement(), data_plane.NewDataplaneConnection, workers.NewWorkerNode, &mockConfig)

	start := time.Now()
	assert.NoError(t, controlPlane.ReconstructState(context.Background(), mockConfig))

	elapsed := time.Since(start)

	cnt := 0

	wg := sync.WaitGroup{}
	wg.Add(size)

	for cnt < size {
		go func() {
			status, err := controlPlane.RegisterService(context.Background(), &proto.ServiceInfo{
				Name:              uuid.New().String(),
				Image:             "",
				PortForwarding:    nil,
				AutoscalingConfig: nil,
			})
			assert.NotNil(t, status)
			assert.NoError(t, err)
			wg.Done()
		}()
		cnt++
	}

	wg.Wait()

	assert.Zero(t, controlPlane.GetNumberConnectedWorkers(), "Number of connected workers should be 0")
	assert.Zero(t, controlPlane.GetNumberDataplanes(), "Number of connected data planes should be 0")
	assert.Equal(t, size, controlPlane.GetNumberServices(), "Number of registered services should be equal")

	logrus.Infof("Took %s seconds to reconstruct", elapsed)
}

func TestStressRegisterNodes(t *testing.T) {
	logrus.SetLevel(logrus.FatalLevel)

	persistenceLayer, ctrl := simplePersistenceLayer(t)
	defer ctrl.Finish()

	size := 100

	persistenceLayer.EXPECT().StoreWorkerNodeInformation(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, workerNodeInfo *proto.WorkerNodeInformation, timestamp time.Time) error {
		return nil
	}).Times(size)

	controlPlane := NewControlPlane(persistenceLayer, "", placement_policy.NewRandomPlacement(), empty_dataplane.NewDataplaneConnectionEmpty, empty_worker.NewEmptyWorkerNode, &mockConfig)

	start := time.Now()
	assert.NoError(t, controlPlane.ReconstructState(context.Background(), mockConfig))

	elapsed := time.Since(start)

	cnt := 0

	wg := sync.WaitGroup{}
	wg.Add(size)

	for cnt < size {
		go func() {
			status, err := controlPlane.RegisterNode(context.Background(), &proto.NodeInfo{
				NodeID:     uuid.New().String(),
				IP:         uuid.New().String(),
				Port:       0,
				CpuCores:   0,
				MemorySize: 0,
			})
			assert.NotNil(t, status)
			assert.NoError(t, err)
			wg.Done()
		}()
		cnt++
	}

	wg.Wait()

	assert.Equal(t, size, controlPlane.GetNumberConnectedWorkers(), "Number of connected workers should be equal")
	assert.Zero(t, controlPlane.GetNumberDataplanes(), "Number of connected data planes should be 0")
	assert.Zero(t, controlPlane.GetNumberServices(), "Number of registered services should be 0")

	logrus.Infof("Took %s seconds to reconstruct", elapsed)
}

func TestStressRegisterDataplanes(t *testing.T) {
	logrus.SetLevel(logrus.FatalLevel)

	persistenceLayer, ctrl := simplePersistenceLayer(t)
	defer ctrl.Finish()

	size := 100

	persistenceLayer.EXPECT().StoreDataPlaneInformation(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, dataplaneInfo *proto.DataplaneInformation, timestamp time.Time) error {
		return nil
	}).Times(size)

	controlPlane := NewControlPlane(persistenceLayer, "", placement_policy.NewRandomPlacement(), empty_dataplane.NewDataplaneConnectionEmpty, empty_worker.NewEmptyWorkerNode, &mockConfig)

	start := time.Now()
	assert.NoError(t, controlPlane.ReconstructState(context.Background(), mockConfig))

	elapsed := time.Since(start)

	cnt := 0

	wg := sync.WaitGroup{}
	wg.Add(size)

	for cnt < size {
		go func() {
			status, err, _ := controlPlane.RegisterDataplane(context.Background(), &proto.DataplaneInfo{
				IP:        uuid.New().String(),
				APIPort:   0,
				ProxyPort: 0,
			})
			assert.NotNil(t, status)
			assert.NoError(t, err)
			wg.Done()
		}()
		cnt++
	}

	wg.Wait()

	assert.Zero(t, controlPlane.GetNumberConnectedWorkers(), "Number of connected workers should be 0")
	assert.Equal(t, size, controlPlane.GetNumberDataplanes(), "Number of connected data planes should be equal")
	assert.Zero(t, controlPlane.GetNumberServices(), "Number of registered services should be 0")

	logrus.Infof("Took %s seconds to reconstruct", elapsed)
}

func TestStressEverything(t *testing.T) {
	logrus.SetLevel(logrus.FatalLevel)

	persistenceLayer, ctrl := simplePersistenceLayer(t)
	defer ctrl.Finish()

	size := 10

	persistenceLayer.EXPECT().StoreWorkerNodeInformation(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, workerNodeInfo *proto.WorkerNodeInformation, timestamp time.Time) error {
		return nil
	}).Times(size)

	persistenceLayer.EXPECT().StoreServiceInformation(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, _ *proto.ServiceInfo, _ time.Time) error {
		return nil
	}).Times(size)

	persistenceLayer.EXPECT().StoreDataPlaneInformation(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, dataplaneInfo *proto.DataplaneInformation, timestamp time.Time) error {
		return nil
	}).Times(size)

	controlPlane := NewControlPlane(persistenceLayer, "", placement_policy.NewRandomPlacement(), empty_dataplane.NewDataplaneConnectionEmpty, empty_worker.NewEmptyWorkerNode, &mockConfig)

	start := time.Now()
	assert.NoError(t, controlPlane.ReconstructState(context.Background(), mockConfig))

	elapsed := time.Since(start)

	cnt := 0

	wg := sync.WaitGroup{}
	wg.Add(3 * size)

	for cnt < size {
		go func() {
			status, err := controlPlane.RegisterNode(context.Background(), &proto.NodeInfo{
				NodeID:     uuid.New().String(),
				IP:         uuid.New().String(),
				Port:       0,
				CpuCores:   0,
				MemorySize: 0,
			})
			assert.NotNil(t, status)
			assert.NoError(t, err)
			wg.Done()
		}()

		go func() {
			status, err, _ := controlPlane.RegisterDataplane(context.Background(), &proto.DataplaneInfo{
				IP:        uuid.New().String(),
				APIPort:   0,
				ProxyPort: 0,
			})
			assert.NotNil(t, status)
			assert.NoError(t, err)
			wg.Done()
		}()

		go func() {
			status, err := controlPlane.RegisterService(context.Background(), &proto.ServiceInfo{
				Name:              uuid.New().String(),
				Image:             "",
				PortForwarding:    nil,
				AutoscalingConfig: nil,
			})

			assert.NotNil(t, status)
			assert.NoError(t, err)
			wg.Done()
		}()
		cnt++
	}

	wg.Wait()

	assert.Equal(t, size, controlPlane.GetNumberConnectedWorkers(), "Number of connected workers should be equal")
	assert.Equal(t, size, controlPlane.GetNumberDataplanes(), "Number of connected data planes should be equal")
	assert.Equal(t, size, controlPlane.GetNumberServices(), "Number of registered services should be equal")

	logrus.Infof("Took %s seconds to reconstruct", elapsed)
}

// Advanced concurrency tests

func TestStressRegisterDeregisterServices(t *testing.T) {
	logrus.SetLevel(logrus.FatalLevel)

	persistenceLayer, ctrl := simplePersistenceLayer(t)
	defer ctrl.Finish()
	size := 100

	persistenceLayer.EXPECT().StoreServiceInformation(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, _ *proto.ServiceInfo, _ time.Time) error {
		return nil
	}).AnyTimes()

	persistenceLayer.EXPECT().DeleteServiceInformation(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, _ *proto.ServiceInfo, _ time.Time) error {
		return nil
	}).AnyTimes()

	controlPlane := NewControlPlane(persistenceLayer, "", placement_policy.NewRandomPlacement(), data_plane.NewDataplaneConnection, workers.NewWorkerNode, &mockConfig)

	start := time.Now()
	assert.NoError(t, controlPlane.ReconstructState(context.Background(), mockConfig))

	elapsed := time.Since(start)

	cnt := 0

	wg := sync.WaitGroup{}
	wg.Add(size)

	for cnt < size {
		go func(idx int) {
			defer wg.Done()

			_, err := controlPlane.RegisterService(context.Background(), &proto.ServiceInfo{
				Name:              "mock" + fmt.Sprint(idx),
				Image:             "",
				PortForwarding:    nil,
				AutoscalingConfig: nil,
			})

			assert.NoError(t, err)

			_, err = controlPlane.DeregisterService(context.Background(), &proto.ServiceInfo{
				Name:              "mock" + fmt.Sprint(idx),
				Image:             "",
				PortForwarding:    nil,
				AutoscalingConfig: nil,
			})

			assert.NoError(t, err)
		}(cnt)
		cnt++
	}

	wg.Wait()

	wg = sync.WaitGroup{}
	wg.Add(size)
	cnt = 0
	for cnt < size {
		go func(idx int) {
			defer wg.Done()
			controlPlane.RegisterService(context.Background(), &proto.ServiceInfo{
				Name:              "mock" + fmt.Sprint(idx),
				Image:             "",
				PortForwarding:    nil,
				AutoscalingConfig: nil,
			})
		}(cnt)

		cnt++
	}

	wg.Wait()

	assert.Zero(t, controlPlane.GetNumberConnectedWorkers(), "Number of connected workers should be 0")
	assert.Zero(t, controlPlane.GetNumberDataplanes(), "Number of connected data planes should be 0")
	assert.Equal(t, size, controlPlane.GetNumberServices(), "Number of registered services should be equal")

	logrus.Infof("Took %s seconds to reconstruct", elapsed)
}

func TestStressRegisterDeregisterNodes(t *testing.T) {
	logrus.SetLevel(logrus.FatalLevel)

	persistenceLayer, ctrl := simplePersistenceLayer(t)
	defer ctrl.Finish()

	size := 100

	persistenceLayer.EXPECT().StoreWorkerNodeInformation(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, _ *proto.WorkerNodeInformation, _ time.Time) error {
		return nil
	}).AnyTimes()

	persistenceLayer.EXPECT().DeleteWorkerNodeInformation(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, _ string, _ time.Time) error {
		return nil
	}).AnyTimes()

	controlPlane := NewControlPlane(persistenceLayer, "", placement_policy.NewRandomPlacement(), empty_dataplane.NewDataplaneConnectionEmpty, empty_worker.NewEmptyWorkerNode, &mockConfig)

	start := time.Now()
	assert.NoError(t, controlPlane.ReconstructState(context.Background(), mockConfig))

	elapsed := time.Since(start)

	cnt := 0

	wg := sync.WaitGroup{}
	wg.Add(2 * size)

	for cnt < size {
		go func(idx int) {
			controlPlane.RegisterNode(context.Background(), &proto.NodeInfo{
				NodeID:     "mock" + fmt.Sprint(idx),
				IP:         "",
				Port:       0,
				CpuCores:   0,
				MemorySize: 0,
			})
			wg.Done()
		}(cnt)

		go func(idx int) {
			controlPlane.DeregisterNode(context.Background(), &proto.NodeInfo{
				NodeID:     "mock" + fmt.Sprint(idx),
				IP:         "",
				Port:       0,
				CpuCores:   0,
				MemorySize: 0,
			})
			wg.Done()
		}(cnt)
		cnt++
	}

	wg.Wait()

	wg = sync.WaitGroup{}
	wg.Add(size)

	cnt = 0

	for cnt < size {
		go func(idx int) {
			controlPlane.RegisterNode(context.Background(), &proto.NodeInfo{
				NodeID:     "mock" + fmt.Sprint(idx),
				IP:         "",
				Port:       0,
				CpuCores:   0,
				MemorySize: 0,
			})
			wg.Done()
		}(cnt)
		cnt++
	}

	wg.Wait()

	assert.Equal(t, size, controlPlane.GetNumberConnectedWorkers(), "Number of connected workers should be equal")
	assert.Zero(t, controlPlane.GetNumberDataplanes(), "Number of connected data planes should be 0")
	assert.Zero(t, controlPlane.GetNumberServices(), "Number of registered services should be 0")

	logrus.Infof("Took %s seconds to reconstruct", elapsed)
}

func TestStressRegisterDeregisterDataplanes(t *testing.T) {
	logrus.SetLevel(logrus.FatalLevel)

	persistenceLayer, ctrl := simplePersistenceLayer(t)
	defer ctrl.Finish()

	size := 100

	persistenceLayer.EXPECT().StoreDataPlaneInformation(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, dataplaneInfo *proto.DataplaneInformation, timestamp time.Time) error {
		return nil
	}).AnyTimes()

	persistenceLayer.EXPECT().DeleteDataPlaneInformation(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, dataplaneInfo *proto.DataplaneInformation, timestamp time.Time) error {
		return nil
	}).AnyTimes()

	controlPlane := NewControlPlane(persistenceLayer, "", placement_policy.NewRandomPlacement(), empty_dataplane.NewDataplaneConnectionEmpty, empty_worker.NewEmptyWorkerNode, &mockConfig)

	start := time.Now()
	assert.NoError(t, controlPlane.ReconstructState(context.Background(), mockConfig))

	elapsed := time.Since(start)

	cnt := 0

	wg := sync.WaitGroup{}
	wg.Add(2 * size)

	for cnt < size {
		go func(idx int) {
			controlPlane.RegisterDataplane(context.Background(), &proto.DataplaneInfo{
				IP:        "mock" + fmt.Sprint(idx),
				APIPort:   0,
				ProxyPort: 0,
			})
			wg.Done()
		}(cnt)

		go func(idx int) {
			controlPlane.DeregisterDataplane(context.Background(), &proto.DataplaneInfo{
				IP:        "mock" + fmt.Sprint(idx),
				APIPort:   0,
				ProxyPort: 0,
			})
			wg.Done()
		}(cnt)
		cnt++
	}

	wg.Wait()

	wg = sync.WaitGroup{}
	wg.Add(size)

	cnt = 0

	for cnt < size {
		go func(idx int) {
			controlPlane.RegisterDataplane(context.Background(), &proto.DataplaneInfo{
				IP:        "mock" + fmt.Sprint(idx),
				APIPort:   0,
				ProxyPort: 0,
			})
			wg.Done()
		}(cnt)
		cnt++
	}

	wg.Wait()

	assert.Zero(t, controlPlane.GetNumberConnectedWorkers(), "Number of connected workers should be 0")
	assert.Equal(t, size, controlPlane.GetNumberDataplanes(), "Number of connected data planes should be equal")
	assert.Zero(t, controlPlane.GetNumberServices(), "Number of registered services should be 0")

	logrus.Infof("Took %s seconds to reconstruct", elapsed)
}

// Endpoints tests

func TestEndpointsWithDeregistration(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	logrus.SetLevel(logrus.FatalLevel)

	size := 100

	persistenceLayer := mock_persistence.NewMockPersistenceLayer(ctrl)

	persistenceLayer.EXPECT().StoreServiceInformation(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, _ *proto.ServiceInfo, _ time.Time) error {
		return nil
	}).Times(size)

	persistenceLayer.EXPECT().StoreWorkerNodeInformation(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, workerNodeInfo *proto.WorkerNodeInformation, timestamp time.Time) error {
		return nil
	}).Times(1)

	persistenceLayer.EXPECT().DeleteWorkerNodeInformation(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, _ string, _ time.Time) error {
		return nil
	}).Times(1)

	persistenceLayer.EXPECT().SetLeader(gomock.Any()).DoAndReturn(func(_ context.Context) error {
		return nil
	}).AnyTimes()

	controlPlane := NewControlPlane(persistenceLayer, "", placement_policy.NewRandomPlacement(), empty_dataplane.NewDataplaneConnectionEmpty, empty_worker.NewEmptyWorkerNode, &mockConfig)

	status, err := controlPlane.RegisterNode(context.Background(), &proto.NodeInfo{
		NodeID:     "mockNode",
		IP:         uuid.New().String(),
		Port:       0,
		CpuCores:   0,
		MemorySize: 0,
	})

	assert.NotNil(t, status)
	assert.NoError(t, err)

	for i := 0; i < size; i++ {
		autoscalingConfig := autoscaling.NewDefaultAutoscalingMetadata()
		autoscalingConfig.ScalingUpperBound = 1
		//autoscalingConfig.ScalingLowerBound = 1

		status, err = controlPlane.RegisterService(context.Background(), &proto.ServiceInfo{
			Name:              "mock" + fmt.Sprint(i),
			Image:             "",
			PortForwarding:    nil,
			AutoscalingConfig: autoscalingConfig,
		})

		assert.True(t, status.Success, "status should be successful")
		assert.NoError(t, err, "error should not be nil")
	}

	for i := 0; i < size; i++ {
		status, err = controlPlane.OnMetricsReceive(context.Background(), &proto.AutoscalingMetric{
			ServiceName:      "mock" + fmt.Sprint(i),
			DataplaneName:    "",
			InflightRequests: 1,
		})

		assert.True(t, status.Success)
		assert.NoError(t, err)
	}

	time.Sleep(500 * time.Millisecond)

	status, err = controlPlane.DeregisterNode(context.Background(), &proto.NodeInfo{
		NodeID:     "mockNode",
		IP:         uuid.New().String(),
		Port:       0,
		CpuCores:   0,
		MemorySize: 0,
	})

	time.Sleep(1 * time.Second)

	sum := 0
	for _, value := range controlPlane.SIStorage.GetMap() {
		sum += len(value.Controller.Endpoints)
	}

	assert.Zero(t, sum)

	sum = 0
	for _, value := range controlPlane.NIStorage.GetMap() {
		sum += value.GetEndpointMap().Len()
	}

	assert.Zero(t, sum)
}

func TestEndpointsWithDeregistrationMultipleNodes(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	logrus.SetLevel(logrus.FatalLevel)

	size := 2

	persistenceLayer := mock_persistence.NewMockPersistenceLayer(ctrl)

	persistenceLayer.EXPECT().StoreServiceInformation(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, _ *proto.ServiceInfo, _ time.Time) error {
		return nil
	}).Times(size)

	persistenceLayer.EXPECT().StoreWorkerNodeInformation(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, workerNodeInfo *proto.WorkerNodeInformation, timestamp time.Time) error {
		return nil
	}).Times(2)

	persistenceLayer.EXPECT().DeleteWorkerNodeInformation(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, _ string, _ time.Time) error {
		return nil
	}).Times(2)

	persistenceLayer.EXPECT().SetLeader(gomock.Any()).DoAndReturn(func(_ context.Context) error {
		return nil
	}).AnyTimes()

	controlPlane := NewControlPlane(persistenceLayer, "", placement_policy.NewRandomPlacement(), empty_dataplane.NewDataplaneConnectionEmpty, empty_worker.NewEmptyWorkerNode, &mockConfig)

	status, err := controlPlane.RegisterNode(context.Background(), &proto.NodeInfo{
		NodeID:     "mockNode",
		IP:         uuid.New().String(),
		Port:       0,
		CpuCores:   0,
		MemorySize: 0,
	})

	assert.NotNil(t, status)
	assert.NoError(t, err)

	status, err = controlPlane.RegisterNode(context.Background(), &proto.NodeInfo{
		NodeID:     "mockNode2",
		IP:         uuid.New().String(),
		Port:       0,
		CpuCores:   0,
		MemorySize: 0,
	})

	assert.True(t, status.Success)
	assert.NoError(t, err)

	for i := 0; i < size; i++ {
		autoscalingConfig := autoscaling.NewDefaultAutoscalingMetadata()
		autoscalingConfig.ScalingUpperBound = 1
		//autoscalingConfig.ScalingLowerBound = 1

		status, err = controlPlane.RegisterService(context.Background(), &proto.ServiceInfo{
			Name:              "mock" + fmt.Sprint(i),
			Image:             "",
			PortForwarding:    nil,
			AutoscalingConfig: autoscalingConfig,
		})

		assert.True(t, status.Success, "status should be successful")
		assert.NoError(t, err, "error should not be nil")
	}

	for i := 0; i < size; i++ {
		status, err = controlPlane.OnMetricsReceive(context.Background(), &proto.AutoscalingMetric{
			ServiceName:      "mock" + fmt.Sprint(i),
			DataplaneName:    "",
			InflightRequests: 1,
		})

		assert.True(t, status.Success)
		assert.NoError(t, err)
	}

	time.Sleep(time.Second)

	status, err = controlPlane.DeregisterNode(context.Background(), &proto.NodeInfo{
		NodeID:     "mockNode",
		IP:         uuid.New().String(),
		Port:       0,
		CpuCores:   0,
		MemorySize: 0,
	})

	assert.True(t, status.Success)
	assert.NoError(t, err)

	time.Sleep(time.Second)

	controlPlane.NIStorage.Lock()
	controlPlane.SIStorage.Lock()

	// Assert structures are consistent
	sum := 0
	for _, value := range controlPlane.SIStorage.GetMap() {
		sum += len(value.Controller.Endpoints)
	}
	for _, value := range controlPlane.NIStorage.GetMap() {
		sum -= value.GetEndpointMap().Len()
	}

	controlPlane.NIStorage.Unlock()
	controlPlane.SIStorage.Unlock()

	assert.Zero(t, sum)

	time.Sleep(time.Second)

	// Make sure scaling loop reconstructs as expected
	sum = 0
	for _, value := range controlPlane.SIStorage.GetMap() {
		sum += len(value.Controller.Endpoints)
	}

	assert.Equal(t, size, sum)

	sum = 0
	for _, value := range controlPlane.NIStorage.GetMap() {
		sum += value.GetEndpointMap().Len()
	}

	assert.Equal(t, size, sum)

	status, err = controlPlane.DeregisterNode(context.Background(), &proto.NodeInfo{
		NodeID:     "mockNode2",
		IP:         uuid.New().String(),
		Port:       0,
		CpuCores:   0,
		MemorySize: 0,
	})

	assert.True(t, status.Success)
	assert.NoError(t, err)

	time.Sleep(time.Second)

	controlPlane.NIStorage.Lock()
	controlPlane.SIStorage.Lock()

	// Assert structures are consistent
	sum = 0
	for _, value := range controlPlane.SIStorage.GetMap() {
		sum += len(value.Controller.Endpoints)
	}
	assert.Zero(t, sum)

	sum = 0
	for _, value := range controlPlane.NIStorage.GetMap() {
		sum -= value.GetEndpointMap().Len()
	}
	assert.Zero(t, sum)

	controlPlane.NIStorage.Unlock()
	controlPlane.SIStorage.Unlock()
}

// Other tests

func TestEndpointSearchByContainerName(t *testing.T) {
	endpoints := []*core.Endpoint{
		{SandboxID: "a"},
		{SandboxID: "b"},
		{SandboxID: "c"},
	}

	if searchEndpointByContainerName(endpoints, "b") == nil ||
		searchEndpointByContainerName(endpoints, "d") != nil {
		t.Error("Search failed.")
	}
}

func TestHandleNodeFailure(t *testing.T) {
	t.Skipf("We need to rewrite the test")
	/*wn1 := &workers.WorkerNode{Name: "node1"}
	wn2 := &workers.WorkerNode{Name: "node2"}

	wep1 := synchronization.NewControlPlaneSyncStructure[string, synchronization.SyncStructure[*core.Endpoint, string]]()
	ss1 := &ServiceInfoStorage{ServiceInfo: &proto.ServiceInfo{Name: "service1"}, WorkerEndpoints: wep1}

	wep2 := synchronization.NewControlPlaneSyncStructure[string, synchronization.SyncStructure[*core.Endpoint, string]]()
	ss2 := &ServiceInfoStorage{ServiceInfo: &proto.ServiceInfo{Name: "service2"}, WorkerEndpoints: wep2}

	wep1.Set(wn1.Name, synchronization.NewControlPlaneSyncStructure[*core.Endpoint, string]())
	d, _ := wep1.Get(wn1.Name)
	d.Set(&core.Endpoint{SandboxID: "sandbox1", Node: wn1}, ss1.ServiceInfo.Name)
	d.Set(&core.Endpoint{SandboxID: "sandbox2", Node: wn1}, ss1.ServiceInfo.Name)

	wep1.Set(wn2.Name, synchronization.NewControlPlaneSyncStructure[*core.Endpoint, string]())
	d, _ = wep1.Get(wn2.Name)
	d.Set(&core.Endpoint{SandboxID: "sandbox3", Node: wn2}, ss2.ServiceInfo.Name)

	wep2.Set(wn1.Name, synchronization.NewControlPlaneSyncStructure[*core.Endpoint, string]())
	d, _ = wep2.Get(wn1.Name)
	d.Set(&core.Endpoint{SandboxID: "sandbox4", Node: wn1}, ss1.ServiceInfo.Name)

	cp := NewControlPlane(nil, "", nil, data_plane.NewDataplaneConnection, workers.NewWorkerNode)
	cp.SIStorage.Set(ss1.ServiceInfo.Name, ss1)
	cp.SIStorage.Set(ss2.ServiceInfo.Name, ss2)

	// node1: sandbox1, sandbox2, sandbox4
	// node2: sandbox3

	failuresWn1 := cp.createWorkerNodeFailureEvents(wn1)
	failuresWn2 := cp.createWorkerNodeFailureEvents(wn2)

	if len(failuresWn1) != 2 ||
		(len(failuresWn1[0].SandboxIDs)+len(failuresWn1[1].SandboxIDs) != 3) {
		t.Error("Invalid number of failures on worker node 1")
	}

	if len(failuresWn2) != 1 || len(failuresWn2[0].SandboxIDs) != 1 {
		t.Error("Invalid number of failures on worker node 2")
	}

	wn1S := make(map[string]struct{})
	if len(failuresWn1[0].SandboxIDs) == 1 {
		wn1S[failuresWn1[0].SandboxIDs[0]] = struct{}{}
		wn1S[failuresWn1[1].SandboxIDs[0]] = struct{}{}
		wn1S[failuresWn1[1].SandboxIDs[1]] = struct{}{}
	} else {
		wn1S[failuresWn1[0].SandboxIDs[0]] = struct{}{}
		wn1S[failuresWn1[0].SandboxIDs[1]] = struct{}{}
		wn1S[failuresWn1[1].SandboxIDs[0]] = struct{}{}
	}
	wn2S1 := failuresWn2[0].SandboxIDs[0]

	if _, ok := wn1S["sandbox1"]; !ok {
		t.Error("Unexpected failure on worker node 1")
	}

	if _, ok := wn1S["sandbox2"]; !ok {
		t.Error("Unexpected failure on worker node 1")
	}

	if _, ok := wn1S["sandbox4"]; !ok {
		t.Error("Unexpected failure on worker node 1")
	}

	if wn2S1 != "sandbox3" {
		t.Error("Unexpected failure on worker node 2")
	}*/
}

// Helpers

type testStructure struct {
	ctrl *gomock.Controller
}

func (t *testStructure) NewMockDataplaneConnection(IP, APIPort, ProxyPort string) core.DataPlaneInterface {
	mockInterface := mock_core.NewMockDataPlaneInterface(t.ctrl)

	mockInterface.EXPECT().InitializeDataPlaneConnection(gomock.Any(), gomock.Any()).DoAndReturn(func(string, string) error {
		return nil
	}).Times(1)

	mockInterface.EXPECT().UpdateEndpointList(gomock.Any(), gomock.Any()).DoAndReturn(func(context.Context, *proto.DeploymentEndpointPatch) (*proto.DeploymentUpdateSuccess, error) {
		return &proto.DeploymentUpdateSuccess{}, nil
	}).AnyTimes()

	return mockInterface
}

func (t *testStructure) NewMockWorkerConnection(input core.WorkerNodeConfiguration) core.WorkerNodeInterface {
	mockInterface := mock_core.NewMockWorkerNodeInterface(t.ctrl)

	mockInterface.EXPECT().GetName().DoAndReturn(func() string {
		return input.Name
	}).AnyTimes()
	mockInterface.EXPECT().GetEndpointMap().AnyTimes()
	mockInterface.EXPECT().ConnectToWorker().AnyTimes()

	mockInterface.EXPECT().ListEndpoints(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(context.Context, *emptypb.Empty, ...grpc.CallOption) (*proto.EndpointsList, error) {
		return &proto.EndpointsList{}, nil
	}).AnyTimes()

	return mockInterface
}

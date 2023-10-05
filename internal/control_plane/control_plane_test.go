package control_plane

import (
	"cluster_manager/api/proto"
	"cluster_manager/internal/control_plane/core"
	"cluster_manager/internal/control_plane/data_plane"
	"cluster_manager/internal/control_plane/placement_policy"
	"cluster_manager/internal/control_plane/workers"
	"cluster_manager/mock/mock_core"
	"cluster_manager/mock/mock_persistence"
	"cluster_manager/pkg/config"
	"cluster_manager/pkg/synchronization"
	"context"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"testing"
	"time"
)

var mockConfig = config.ControlPlaneConfig{
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

	controlPlane := NewControlPlane(persistenceLayer, "", placement_policy.NewRandomPolicy(), data_plane.NewDataplaneConnection, workers.NewWorkerNode)

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

	controlPlane := NewControlPlane(persistenceLayer, "", placement_policy.NewRandomPolicy(), data_plane.NewDataplaneConnection, workers.NewWorkerNode)

	start := time.Now()
	err := controlPlane.ReconstructState(context.Background(), mockConfig)
	elapsed := time.Since(start)

	assert.NoError(t, err, "reconstructing control plane state failed")

	assert.Zero(t, controlPlane.GetNumberConnectedWorkers(), "Number of connected workers should be 0")
	assert.Zero(t, controlPlane.GetNumberDataplanes(), "Number of connected data planes should be 0")
	assert.Equal(t, 5, controlPlane.GetNumberServices(), "Number of registered services should be 0")

	logrus.Infof("Took %s seconds to reconstruct", elapsed)
}

type testStructure struct {
	ctrl *gomock.Controller
}

func (t *testStructure) NewMockDataplaneConnection(IP, APIPort, ProxyPort string) core.DataPlaneInterface {
	mockInterface := mock_core.NewMockDataPlaneInterface(t.ctrl)

	mockInterface.EXPECT().InitializeDataPlaneConnection(gomock.Any(), gomock.Any()).DoAndReturn(func(string, string) error {
		return nil
	}).Times(1)

	return mockInterface
}

func (t *testStructure) NewMockWorkerConnection(input core.WorkerNodeConfiguration) core.WorkerNodeInterface {
	mockInterface := mock_core.NewMockWorkerNodeInterface(t.ctrl)

	mockInterface.EXPECT().GetName().DoAndReturn(func() string {
		return input.Name
	}).AnyTimes()
	mockInterface.EXPECT().GetAPI().AnyTimes()

	mockInterface.EXPECT().ListEndpoints(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(context.Context, *emptypb.Empty, ...grpc.CallOption) (*proto.EndpointsList, error) {
		return &proto.EndpointsList{}, nil
	}).AnyTimes()

	return mockInterface
}

func TestRegisterWorkers(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

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

	controlPlane := NewControlPlane(persistenceLayer, "", placement_policy.NewRandomPolicy(), te.NewMockDataplaneConnection, te.NewMockWorkerConnection)

	start := time.Now()
	err := controlPlane.ReconstructState(context.Background(), mockConfig)
	elapsed := time.Since(start)

	assert.NoError(t, err, "reconstructing control plane state failed")

	assert.Equal(t, 3, controlPlane.GetNumberConnectedWorkers(), "Number of connected workers should be 3")
	assert.Zero(t, controlPlane.GetNumberDataplanes(), "Number of connected data planes should be 0")
	assert.Zero(t, controlPlane.GetNumberServices(), "Number of registered services should be 0")

	logrus.Infof("Took %s seconds to reconstruct", elapsed)
}

func TestRegisterDataplanes(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

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

	controlPlane := NewControlPlane(persistenceLayer, "", placement_policy.NewRandomPolicy(), te.NewMockDataplaneConnection, workers.NewWorkerNode)

	start := time.Now()
	err := controlPlane.ReconstructState(context.Background(), mockConfig)
	elapsed := time.Since(start)

	assert.NoError(t, err, "reconstructing control plane state failed")

	assert.Zero(t, controlPlane.GetNumberConnectedWorkers(), "Number of connected workers should be 0")
	assert.Equal(t, 3, controlPlane.GetNumberDataplanes(), "Number of connected data planes should be 3")
	assert.Zero(t, controlPlane.GetNumberServices(), "Number of registered services should be 0")

	logrus.Infof("Took %s seconds to reconstruct", elapsed)
}

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
	wn1 := &workers.WorkerNode{Name: "node1"}
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
	}
}

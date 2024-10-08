// Code generated by MockGen. DO NOT EDIT.
// Source: internal/control_plane/persistence/interface.go
//
// Generated by this command:
//
//	mockgen -source internal/control_plane/persistence/interface.go
//
// Package mock_persistence is a generated GoMock package.
package mock_persistence

import (
	proto "cluster_manager/api/proto"
	core "cluster_manager/internal/control_plane/core"
	context "context"
	reflect "reflect"

	gomock "go.uber.org/mock/gomock"
)

// MockPersistenceLayer is a mock of PersistenceLayer interface.
type MockPersistenceLayer struct {
	ctrl     *gomock.Controller
	recorder *MockPersistenceLayerMockRecorder
}

// MockPersistenceLayerMockRecorder is the mock recorder for MockPersistenceLayer.
type MockPersistenceLayerMockRecorder struct {
	mock *MockPersistenceLayer
}

// NewMockPersistenceLayer creates a new mock instance.
func NewMockPersistenceLayer(ctrl *gomock.Controller) *MockPersistenceLayer {
	mock := &MockPersistenceLayer{ctrl: ctrl}
	mock.recorder = &MockPersistenceLayerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockPersistenceLayer) EXPECT() *MockPersistenceLayerMockRecorder {
	return m.recorder
}

// DeleteDataPlaneInformation mocks base method.
func (m *MockPersistenceLayer) DeleteDataPlaneInformation(ctx context.Context, dataplaneInfo *proto.DataplaneInformation) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteDataPlaneInformation", ctx, dataplaneInfo)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteDataPlaneInformation indicates an expected call of DeleteDataPlaneInformation.
func (mr *MockPersistenceLayerMockRecorder) DeleteDataPlaneInformation(ctx, dataplaneInfo any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteDataPlaneInformation", reflect.TypeOf((*MockPersistenceLayer)(nil).DeleteDataPlaneInformation), ctx, dataplaneInfo)
}

// DeleteServiceInformation mocks base method.
func (m *MockPersistenceLayer) DeleteServiceInformation(ctx context.Context, serviceInfo *proto.ServiceInfo) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteServiceInformation", ctx, serviceInfo)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteServiceInformation indicates an expected call of DeleteServiceInformation.
func (mr *MockPersistenceLayerMockRecorder) DeleteServiceInformation(ctx, serviceInfo any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteServiceInformation", reflect.TypeOf((*MockPersistenceLayer)(nil).DeleteServiceInformation), ctx, serviceInfo)
}

// DeleteWorkerNodeInformation mocks base method.
func (m *MockPersistenceLayer) DeleteWorkerNodeInformation(ctx context.Context, name string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteWorkerNodeInformation", ctx, name)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteWorkerNodeInformation indicates an expected call of DeleteWorkerNodeInformation.
func (mr *MockPersistenceLayerMockRecorder) DeleteWorkerNodeInformation(ctx, name any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteWorkerNodeInformation", reflect.TypeOf((*MockPersistenceLayer)(nil).DeleteWorkerNodeInformation), ctx, name)
}

// GetDataPlaneInformation mocks base method.
func (m *MockPersistenceLayer) GetDataPlaneInformation(ctx context.Context) ([]*proto.DataplaneInformation, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetDataPlaneInformation", ctx)
	ret0, _ := ret[0].([]*proto.DataplaneInformation)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetDataPlaneInformation indicates an expected call of GetDataPlaneInformation.
func (mr *MockPersistenceLayerMockRecorder) GetDataPlaneInformation(ctx any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetDataPlaneInformation", reflect.TypeOf((*MockPersistenceLayer)(nil).GetDataPlaneInformation), ctx)
}

// GetServiceInformation mocks base method.
func (m *MockPersistenceLayer) GetServiceInformation(ctx context.Context) ([]*proto.ServiceInfo, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetServiceInformation", ctx)
	ret0, _ := ret[0].([]*proto.ServiceInfo)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetServiceInformation indicates an expected call of GetServiceInformation.
func (mr *MockPersistenceLayerMockRecorder) GetServiceInformation(ctx any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetServiceInformation", reflect.TypeOf((*MockPersistenceLayer)(nil).GetServiceInformation), ctx)
}

// GetWorkerNodeInformation mocks base method.
func (m *MockPersistenceLayer) GetWorkerNodeInformation(ctx context.Context) ([]*proto.WorkerNodeInformation, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetWorkerNodeInformation", ctx)
	ret0, _ := ret[0].([]*proto.WorkerNodeInformation)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetWorkerNodeInformation indicates an expected call of GetWorkerNodeInformation.
func (mr *MockPersistenceLayerMockRecorder) GetWorkerNodeInformation(ctx any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetWorkerNodeInformation", reflect.TypeOf((*MockPersistenceLayer)(nil).GetWorkerNodeInformation), ctx)
}

// SetLeader mocks base method.
func (m *MockPersistenceLayer) SetLeader(ctx context.Context) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SetLeader", ctx)
	ret0, _ := ret[0].(error)
	return ret0
}

// SetLeader indicates an expected call of SetLeader.
func (mr *MockPersistenceLayerMockRecorder) SetLeader(ctx any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetLeader", reflect.TypeOf((*MockPersistenceLayer)(nil).SetLeader), ctx)
}

// StoreDataPlaneInformation mocks base method.
func (m *MockPersistenceLayer) StoreDataPlaneInformation(ctx context.Context, dataplaneInfo *proto.DataplaneInformation) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "StoreDataPlaneInformation", ctx, dataplaneInfo)
	ret0, _ := ret[0].(error)
	return ret0
}

// StoreDataPlaneInformation indicates an expected call of StoreDataPlaneInformation.
func (mr *MockPersistenceLayerMockRecorder) StoreDataPlaneInformation(ctx, dataplaneInfo any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "StoreDataPlaneInformation", reflect.TypeOf((*MockPersistenceLayer)(nil).StoreDataPlaneInformation), ctx, dataplaneInfo)
}

// StoreEndpoint mocks base method.
func (m *MockPersistenceLayer) StoreEndpoint(ctx context.Context, endpoint core.Endpoint) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "StoreEndpoint", ctx, endpoint)
	ret0, _ := ret[0].(error)
	return ret0
}

// StoreEndpoint indicates an expected call of StoreEndpoint.
func (mr *MockPersistenceLayerMockRecorder) StoreEndpoint(ctx, endpoint any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "StoreEndpoint", reflect.TypeOf((*MockPersistenceLayer)(nil).StoreEndpoint), ctx, endpoint)
}

// StoreServiceInformation mocks base method.
func (m *MockPersistenceLayer) StoreServiceInformation(ctx context.Context, serviceInfo *proto.ServiceInfo) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "StoreServiceInformation", ctx, serviceInfo)
	ret0, _ := ret[0].(error)
	return ret0
}

// StoreServiceInformation indicates an expected call of StoreServiceInformation.
func (mr *MockPersistenceLayerMockRecorder) StoreServiceInformation(ctx, serviceInfo any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "StoreServiceInformation", reflect.TypeOf((*MockPersistenceLayer)(nil).StoreServiceInformation), ctx, serviceInfo)
}

// StoreWorkerNodeInformation mocks base method.
func (m *MockPersistenceLayer) StoreWorkerNodeInformation(ctx context.Context, workerNodeInfo *proto.WorkerNodeInformation) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "StoreWorkerNodeInformation", ctx, workerNodeInfo)
	ret0, _ := ret[0].(error)
	return ret0
}

// StoreWorkerNodeInformation indicates an expected call of StoreWorkerNodeInformation.
func (mr *MockPersistenceLayerMockRecorder) StoreWorkerNodeInformation(ctx, workerNodeInfo any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "StoreWorkerNodeInformation", reflect.TypeOf((*MockPersistenceLayer)(nil).StoreWorkerNodeInformation), ctx, workerNodeInfo)
}

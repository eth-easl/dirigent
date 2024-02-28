// Code generated by MockGen. DO NOT EDIT.
// Source: internal/control_plane/persistence/interface.go

// Package mock_persistence is a generated GoMock package.
package mock_persistence

import (
	proto "cluster_manager/api/proto"
	context "context"
	reflect "reflect"
	time "time"

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
func (m *MockPersistenceLayer) DeleteDataPlaneInformation(ctx context.Context, dataplaneInfo *proto.DataplaneInformation, timestamp time.Time) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteDataPlaneInformation", ctx, dataplaneInfo, timestamp)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteDataPlaneInformation indicates an expected call of DeleteDataPlaneInformation.
func (mr *MockPersistenceLayerMockRecorder) DeleteDataPlaneInformation(ctx, dataplaneInfo, timestamp interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteDataPlaneInformation", reflect.TypeOf((*MockPersistenceLayer)(nil).DeleteDataPlaneInformation), ctx, dataplaneInfo, timestamp)
}

// DeleteServiceInformation mocks base method.
func (m *MockPersistenceLayer) DeleteServiceInformation(ctx context.Context, serviceInfo *proto.ServiceInfo, timestamp time.Time) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteServiceInformation", ctx, serviceInfo, timestamp)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteServiceInformation indicates an expected call of DeleteServiceInformation.
func (mr *MockPersistenceLayerMockRecorder) DeleteServiceInformation(ctx, serviceInfo, timestamp interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteServiceInformation", reflect.TypeOf((*MockPersistenceLayer)(nil).DeleteServiceInformation), ctx, serviceInfo, timestamp)
}

// DeleteWorkerNodeInformation mocks base method.
func (m *MockPersistenceLayer) DeleteWorkerNodeInformation(ctx context.Context, name string, timestamp time.Time) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteWorkerNodeInformation", ctx, name, timestamp)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteWorkerNodeInformation indicates an expected call of DeleteWorkerNodeInformation.
func (mr *MockPersistenceLayerMockRecorder) DeleteWorkerNodeInformation(ctx, name, timestamp interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteWorkerNodeInformation", reflect.TypeOf((*MockPersistenceLayer)(nil).DeleteWorkerNodeInformation), ctx, name, timestamp)
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
func (mr *MockPersistenceLayerMockRecorder) GetDataPlaneInformation(ctx interface{}) *gomock.Call {
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
func (mr *MockPersistenceLayerMockRecorder) GetServiceInformation(ctx interface{}) *gomock.Call {
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
func (mr *MockPersistenceLayerMockRecorder) GetWorkerNodeInformation(ctx interface{}) *gomock.Call {
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
func (mr *MockPersistenceLayerMockRecorder) SetLeader(ctx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetLeader", reflect.TypeOf((*MockPersistenceLayer)(nil).SetLeader), ctx)
}

// StoreDataPlaneInformation mocks base method.
func (m *MockPersistenceLayer) StoreDataPlaneInformation(ctx context.Context, dataplaneInfo *proto.DataplaneInformation, timestamp time.Time) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "StoreDataPlaneInformation", ctx, dataplaneInfo, timestamp)
	ret0, _ := ret[0].(error)
	return ret0
}

// StoreDataPlaneInformation indicates an expected call of StoreDataPlaneInformation.
func (mr *MockPersistenceLayerMockRecorder) StoreDataPlaneInformation(ctx, dataplaneInfo, timestamp interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "StoreDataPlaneInformation", reflect.TypeOf((*MockPersistenceLayer)(nil).StoreDataPlaneInformation), ctx, dataplaneInfo, timestamp)
}

// StoreServiceInformation mocks base method.
func (m *MockPersistenceLayer) StoreServiceInformation(ctx context.Context, serviceInfo *proto.ServiceInfo, timestamp time.Time) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "StoreServiceInformation", ctx, serviceInfo, timestamp)
	ret0, _ := ret[0].(error)
	return ret0
}

// StoreServiceInformation indicates an expected call of StoreServiceInformation.
func (mr *MockPersistenceLayerMockRecorder) StoreServiceInformation(ctx, serviceInfo, timestamp interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "StoreServiceInformation", reflect.TypeOf((*MockPersistenceLayer)(nil).StoreServiceInformation), ctx, serviceInfo, timestamp)
}

// StoreWorkerNodeInformation mocks base method.
func (m *MockPersistenceLayer) StoreWorkerNodeInformation(ctx context.Context, workerNodeInfo *proto.WorkerNodeInformation, timestamp time.Time) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "StoreWorkerNodeInformation", ctx, workerNodeInfo, timestamp)
	ret0, _ := ret[0].(error)
	return ret0
}

// StoreWorkerNodeInformation indicates an expected call of StoreWorkerNodeInformation.
func (mr *MockPersistenceLayerMockRecorder) StoreWorkerNodeInformation(ctx, workerNodeInfo, timestamp interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "StoreWorkerNodeInformation", reflect.TypeOf((*MockPersistenceLayer)(nil).StoreWorkerNodeInformation), ctx, workerNodeInfo, timestamp)
}

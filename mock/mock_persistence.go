// Code generated by MockGen. DO NOT EDIT.
// Source: internal/control_plane/persistence/interface.go

// Package mock_persistence is a generated GoMock package.
package mock_persistence

import (
	proto "cluster_manager/api/proto"
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
func (mr *MockPersistenceLayerMockRecorder) DeleteDataPlaneInformation(ctx, dataplaneInfo interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteDataPlaneInformation", reflect.TypeOf((*MockPersistenceLayer)(nil).DeleteDataPlaneInformation), ctx, dataplaneInfo)
}

// DeleteEndpoint mocks base method.
func (m *MockPersistenceLayer) DeleteEndpoint(ctx context.Context, serviceName, workerNodeName string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteEndpoint", ctx, serviceName, workerNodeName)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteEndpoint indicates an expected call of DeleteEndpoint.
func (mr *MockPersistenceLayerMockRecorder) DeleteEndpoint(ctx, serviceName, workerNodeName interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteEndpoint", reflect.TypeOf((*MockPersistenceLayer)(nil).DeleteEndpoint), ctx, serviceName, workerNodeName)
}

// DeleteWorkerNodeInformation mocks base method.
func (m *MockPersistenceLayer) DeleteWorkerNodeInformation(ctx context.Context, workerNodeInfo *proto.WorkerNodeInformation) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteWorkerNodeInformation", ctx, workerNodeInfo)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteWorkerNodeInformation indicates an expected call of DeleteWorkerNodeInformation.
func (mr *MockPersistenceLayerMockRecorder) DeleteWorkerNodeInformation(ctx, workerNodeInfo interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteWorkerNodeInformation", reflect.TypeOf((*MockPersistenceLayer)(nil).DeleteWorkerNodeInformation), ctx, workerNodeInfo)
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

// GetEndpoints mocks base method.
func (m *MockPersistenceLayer) GetEndpoints(ctx context.Context) ([]*proto.Endpoint, []string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetEndpoints", ctx)
	ret0, _ := ret[0].([]*proto.Endpoint)
	ret1, _ := ret[1].([]string)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// GetEndpoints indicates an expected call of GetEndpoints.
func (mr *MockPersistenceLayerMockRecorder) GetEndpoints(ctx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetEndpoints", reflect.TypeOf((*MockPersistenceLayer)(nil).GetEndpoints), ctx)
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

// StoreDataPlaneInformation mocks base method.
func (m *MockPersistenceLayer) StoreDataPlaneInformation(ctx context.Context, dataplaneInfo *proto.DataplaneInformation) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "StoreDataPlaneInformation", ctx, dataplaneInfo)
	ret0, _ := ret[0].(error)
	return ret0
}

// StoreDataPlaneInformation indicates an expected call of StoreDataPlaneInformation.
func (mr *MockPersistenceLayerMockRecorder) StoreDataPlaneInformation(ctx, dataplaneInfo interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "StoreDataPlaneInformation", reflect.TypeOf((*MockPersistenceLayer)(nil).StoreDataPlaneInformation), ctx, dataplaneInfo)
}

// StoreSerialized mocks base method.
func (m *MockPersistenceLayer) StoreSerialized(ctx context.Context, controlPlane []byte) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "StoreSerialized", ctx, controlPlane)
	ret0, _ := ret[0].(error)
	return ret0
}

// StoreSerialized indicates an expected call of StoreSerialized.
func (mr *MockPersistenceLayerMockRecorder) StoreSerialized(ctx, controlPlane interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "StoreSerialized", reflect.TypeOf((*MockPersistenceLayer)(nil).StoreSerialized), ctx, controlPlane)
}

// StoreServiceInformation mocks base method.
func (m *MockPersistenceLayer) StoreServiceInformation(ctx context.Context, serviceInfo *proto.ServiceInfo) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "StoreServiceInformation", ctx, serviceInfo)
	ret0, _ := ret[0].(error)
	return ret0
}

// StoreServiceInformation indicates an expected call of StoreServiceInformation.
func (mr *MockPersistenceLayerMockRecorder) StoreServiceInformation(ctx, serviceInfo interface{}) *gomock.Call {
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
func (mr *MockPersistenceLayerMockRecorder) StoreWorkerNodeInformation(ctx, workerNodeInfo interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "StoreWorkerNodeInformation", reflect.TypeOf((*MockPersistenceLayer)(nil).StoreWorkerNodeInformation), ctx, workerNodeInfo)
}

// UpdateEndpoints mocks base method.
func (m *MockPersistenceLayer) UpdateEndpoints(ctx context.Context, serviceName string, endpoints []*proto.Endpoint) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateEndpoints", ctx, serviceName, endpoints)
	ret0, _ := ret[0].(error)
	return ret0
}

// UpdateEndpoints indicates an expected call of UpdateEndpoints.
func (mr *MockPersistenceLayerMockRecorder) UpdateEndpoints(ctx, serviceName, endpoints interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateEndpoints", reflect.TypeOf((*MockPersistenceLayer)(nil).UpdateEndpoints), ctx, serviceName, endpoints)
}
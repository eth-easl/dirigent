// Code generated by MockGen. DO NOT EDIT.
// Source: internal/control_plane/core/interface.go
//
// Generated by this command:
//
//	mockgen -source internal/control_plane/core/interface.go
//
// Package mock_core is a generated GoMock package.
package mock_core

import (
	proto "cluster_manager/api/proto"
	core "cluster_manager/internal/control_plane/core"
	synchronization "cluster_manager/pkg/synchronization"
	context "context"
	reflect "reflect"
	time "time"

	gomock "go.uber.org/mock/gomock"
	grpc "google.golang.org/grpc"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

// MockDataPlaneInterface is a mock of DataPlaneInterface interface.
type MockDataPlaneInterface struct {
	ctrl     *gomock.Controller
	recorder *MockDataPlaneInterfaceMockRecorder
}

// MockDataPlaneInterfaceMockRecorder is the mock recorder for MockDataPlaneInterface.
type MockDataPlaneInterfaceMockRecorder struct {
	mock *MockDataPlaneInterface
}

// NewMockDataPlaneInterface creates a new mock instance.
func NewMockDataPlaneInterface(ctrl *gomock.Controller) *MockDataPlaneInterface {
	mock := &MockDataPlaneInterface{ctrl: ctrl}
	mock.recorder = &MockDataPlaneInterfaceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockDataPlaneInterface) EXPECT() *MockDataPlaneInterfaceMockRecorder {
	return m.recorder
}

// AddDeployment mocks base method.
func (m *MockDataPlaneInterface) AddDeployment(arg0 context.Context, arg1 *proto.ServiceInfo) (*proto.DeploymentUpdateSuccess, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AddDeployment", arg0, arg1)
	ret0, _ := ret[0].(*proto.DeploymentUpdateSuccess)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// AddDeployment indicates an expected call of AddDeployment.
func (mr *MockDataPlaneInterfaceMockRecorder) AddDeployment(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AddDeployment", reflect.TypeOf((*MockDataPlaneInterface)(nil).AddDeployment), arg0, arg1)
}

// DeleteDeployment mocks base method.
func (m *MockDataPlaneInterface) DeleteDeployment(arg0 context.Context, arg1 *proto.ServiceInfo) (*proto.DeploymentUpdateSuccess, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteDeployment", arg0, arg1)
	ret0, _ := ret[0].(*proto.DeploymentUpdateSuccess)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// DeleteDeployment indicates an expected call of DeleteDeployment.
func (mr *MockDataPlaneInterfaceMockRecorder) DeleteDeployment(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteDeployment", reflect.TypeOf((*MockDataPlaneInterface)(nil).DeleteDeployment), arg0, arg1)
}

// DrainSandbox mocks base method.
func (m *MockDataPlaneInterface) DrainSandbox(arg0 context.Context, arg1 *proto.DeploymentEndpointPatch) (*proto.DeploymentUpdateSuccess, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DrainSandbox", arg0, arg1)
	ret0, _ := ret[0].(*proto.DeploymentUpdateSuccess)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// DrainSandbox indicates an expected call of DrainSandbox.
func (mr *MockDataPlaneInterfaceMockRecorder) DrainSandbox(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DrainSandbox", reflect.TypeOf((*MockDataPlaneInterface)(nil).DrainSandbox), arg0, arg1)
}

// GetApiPort mocks base method.
func (m *MockDataPlaneInterface) GetApiPort() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetApiPort")
	ret0, _ := ret[0].(string)
	return ret0
}

// GetApiPort indicates an expected call of GetApiPort.
func (mr *MockDataPlaneInterfaceMockRecorder) GetApiPort() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetApiPort", reflect.TypeOf((*MockDataPlaneInterface)(nil).GetApiPort))
}

// GetIP mocks base method.
func (m *MockDataPlaneInterface) GetIP() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetIP")
	ret0, _ := ret[0].(string)
	return ret0
}

// GetIP indicates an expected call of GetIP.
func (mr *MockDataPlaneInterfaceMockRecorder) GetIP() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetIP", reflect.TypeOf((*MockDataPlaneInterface)(nil).GetIP))
}

// GetLastHeartBeat mocks base method.
func (m *MockDataPlaneInterface) GetLastHeartBeat() time.Time {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetLastHeartBeat")
	ret0, _ := ret[0].(time.Time)
	return ret0
}

// GetLastHeartBeat indicates an expected call of GetLastHeartBeat.
func (mr *MockDataPlaneInterfaceMockRecorder) GetLastHeartBeat() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetLastHeartBeat", reflect.TypeOf((*MockDataPlaneInterface)(nil).GetLastHeartBeat))
}

// GetProxyPort mocks base method.
func (m *MockDataPlaneInterface) GetProxyPort() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetProxyPort")
	ret0, _ := ret[0].(string)
	return ret0
}

// GetProxyPort indicates an expected call of GetProxyPort.
func (mr *MockDataPlaneInterfaceMockRecorder) GetProxyPort() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetProxyPort", reflect.TypeOf((*MockDataPlaneInterface)(nil).GetProxyPort))
}

// InitializeDataPlaneConnection mocks base method.
func (m *MockDataPlaneInterface) InitializeDataPlaneConnection(host, port string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "InitializeDataPlaneConnection", host, port)
	ret0, _ := ret[0].(error)
	return ret0
}

// InitializeDataPlaneConnection indicates an expected call of InitializeDataPlaneConnection.
func (mr *MockDataPlaneInterfaceMockRecorder) InitializeDataPlaneConnection(host, port any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "InitializeDataPlaneConnection", reflect.TypeOf((*MockDataPlaneInterface)(nil).InitializeDataPlaneConnection), host, port)
}

// ResetMeasurements mocks base method.
func (m *MockDataPlaneInterface) ResetMeasurements(arg0 context.Context, arg1 *emptypb.Empty) (*proto.ActionStatus, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ResetMeasurements", arg0, arg1)
	ret0, _ := ret[0].(*proto.ActionStatus)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ResetMeasurements indicates an expected call of ResetMeasurements.
func (mr *MockDataPlaneInterfaceMockRecorder) ResetMeasurements(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ResetMeasurements", reflect.TypeOf((*MockDataPlaneInterface)(nil).ResetMeasurements), arg0, arg1)
}

// UpdateEndpointList mocks base method.
func (m *MockDataPlaneInterface) UpdateEndpointList(arg0 context.Context, arg1 *proto.DeploymentEndpointPatch) (*proto.DeploymentUpdateSuccess, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateEndpointList", arg0, arg1)
	ret0, _ := ret[0].(*proto.DeploymentUpdateSuccess)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateEndpointList indicates an expected call of UpdateEndpointList.
func (mr *MockDataPlaneInterfaceMockRecorder) UpdateEndpointList(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateEndpointList", reflect.TypeOf((*MockDataPlaneInterface)(nil).UpdateEndpointList), arg0, arg1)
}

// UpdateHeartBeat mocks base method.
func (m *MockDataPlaneInterface) UpdateHeartBeat() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "UpdateHeartBeat")
}

// UpdateHeartBeat indicates an expected call of UpdateHeartBeat.
func (mr *MockDataPlaneInterfaceMockRecorder) UpdateHeartBeat() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateHeartBeat", reflect.TypeOf((*MockDataPlaneInterface)(nil).UpdateHeartBeat))
}

// MockWorkerNodeInterface is a mock of WorkerNodeInterface interface.
type MockWorkerNodeInterface struct {
	ctrl     *gomock.Controller
	recorder *MockWorkerNodeInterfaceMockRecorder
}

// MockWorkerNodeInterfaceMockRecorder is the mock recorder for MockWorkerNodeInterface.
type MockWorkerNodeInterfaceMockRecorder struct {
	mock *MockWorkerNodeInterface
}

// NewMockWorkerNodeInterface creates a new mock instance.
func NewMockWorkerNodeInterface(ctrl *gomock.Controller) *MockWorkerNodeInterface {
	mock := &MockWorkerNodeInterface{ctrl: ctrl}
	mock.recorder = &MockWorkerNodeInterfaceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockWorkerNodeInterface) EXPECT() *MockWorkerNodeInterfaceMockRecorder {
	return m.recorder
}

// CreateSandbox mocks base method.
func (m *MockWorkerNodeInterface) CreateSandbox(arg0 context.Context, arg1 *proto.ServiceInfo, arg2 ...grpc.CallOption) (*proto.SandboxCreationStatus, error) {
	m.ctrl.T.Helper()
	varargs := []any{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "CreateSandbox", varargs...)
	ret0, _ := ret[0].(*proto.SandboxCreationStatus)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateSandbox indicates an expected call of CreateSandbox.
func (mr *MockWorkerNodeInterfaceMockRecorder) CreateSandbox(arg0, arg1 any, arg2 ...any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]any{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateSandbox", reflect.TypeOf((*MockWorkerNodeInterface)(nil).CreateSandbox), varargs...)
}

// DeleteSandbox mocks base method.
func (m *MockWorkerNodeInterface) DeleteSandbox(arg0 context.Context, arg1 *proto.SandboxID, arg2 ...grpc.CallOption) (*proto.ActionStatus, error) {
	m.ctrl.T.Helper()
	varargs := []any{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "DeleteSandbox", varargs...)
	ret0, _ := ret[0].(*proto.ActionStatus)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// DeleteSandbox indicates an expected call of DeleteSandbox.
func (mr *MockWorkerNodeInterfaceMockRecorder) DeleteSandbox(arg0, arg1 any, arg2 ...any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]any{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteSandbox", reflect.TypeOf((*MockWorkerNodeInterface)(nil).DeleteSandbox), varargs...)
}

// GetAPI mocks base method.
func (m *MockWorkerNodeInterface) ConnectToWorker() proto.WorkerNodeInterfaceClient {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ConnectToWorker")
	ret0, _ := ret[0].(proto.WorkerNodeInterfaceClient)
	return ret0
}

// GetAPI indicates an expected call of GetAPI.
func (mr *MockWorkerNodeInterfaceMockRecorder) GetAPI() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ConnectToWorker", reflect.TypeOf((*MockWorkerNodeInterface)(nil).ConnectToWorker))
}

// GetCpuCores mocks base method.
func (m *MockWorkerNodeInterface) GetCpuCores() uint64 {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetCpuCores")
	ret0, _ := ret[0].(uint64)
	return ret0
}

// GetCpuCores indicates an expected call of GetCpuCores.
func (mr *MockWorkerNodeInterfaceMockRecorder) GetCpuCores() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetCpuCores", reflect.TypeOf((*MockWorkerNodeInterface)(nil).GetCpuCores))
}

// GetCpuUsage mocks base method.
func (m *MockWorkerNodeInterface) GetCpuUsage() uint64 {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetCpuUsage")
	ret0, _ := ret[0].(uint64)
	return ret0
}

// GetCpuUsage indicates an expected call of GetCpuUsage.
func (mr *MockWorkerNodeInterfaceMockRecorder) GetCpuUsage() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetCpuUsage", reflect.TypeOf((*MockWorkerNodeInterface)(nil).GetCpuUsage))
}

// GetEndpointMap mocks base method.
func (m *MockWorkerNodeInterface) GetEndpointMap() synchronization.SyncStructure[*core.Endpoint, string] {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetEndpointMap")
	ret0, _ := ret[0].(synchronization.SyncStructure[*core.Endpoint, string])
	return ret0
}

// GetEndpointMap indicates an expected call of GetEndpointMap.
func (mr *MockWorkerNodeInterfaceMockRecorder) GetEndpointMap() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetEndpointMap", reflect.TypeOf((*MockWorkerNodeInterface)(nil).GetEndpointMap))
}

// GetIP mocks base method.
func (m *MockWorkerNodeInterface) GetIP() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetIP")
	ret0, _ := ret[0].(string)
	return ret0
}

// GetIP indicates an expected call of GetIP.
func (mr *MockWorkerNodeInterfaceMockRecorder) GetIP() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetIP", reflect.TypeOf((*MockWorkerNodeInterface)(nil).GetIP))
}

// GetLastHeartBeat mocks base method.
func (m *MockWorkerNodeInterface) GetLastHeartBeat() time.Time {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetLastHeartBeat")
	ret0, _ := ret[0].(time.Time)
	return ret0
}

// GetLastHeartBeat indicates an expected call of GetLastHeartBeat.
func (mr *MockWorkerNodeInterfaceMockRecorder) GetLastHeartBeat() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetLastHeartBeat", reflect.TypeOf((*MockWorkerNodeInterface)(nil).GetLastHeartBeat))
}

// GetMemory mocks base method.
func (m *MockWorkerNodeInterface) GetMemory() uint64 {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetMemory")
	ret0, _ := ret[0].(uint64)
	return ret0
}

// GetMemory indicates an expected call of GetMemory.
func (mr *MockWorkerNodeInterfaceMockRecorder) GetMemory() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetMemory", reflect.TypeOf((*MockWorkerNodeInterface)(nil).GetMemory))
}

// GetMemoryUsage mocks base method.
func (m *MockWorkerNodeInterface) GetMemoryUsage() uint64 {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetMemoryUsage")
	ret0, _ := ret[0].(uint64)
	return ret0
}

// GetMemoryUsage indicates an expected call of GetMemoryUsage.
func (mr *MockWorkerNodeInterfaceMockRecorder) GetMemoryUsage() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetMemoryUsage", reflect.TypeOf((*MockWorkerNodeInterface)(nil).GetMemoryUsage))
}

// GetName mocks base method.
func (m *MockWorkerNodeInterface) GetName() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetName")
	ret0, _ := ret[0].(string)
	return ret0
}

// GetName indicates an expected call of GetName.
func (mr *MockWorkerNodeInterfaceMockRecorder) GetName() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetName", reflect.TypeOf((*MockWorkerNodeInterface)(nil).GetName))
}

// GetPort mocks base method.
func (m *MockWorkerNodeInterface) GetPort() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetPort")
	ret0, _ := ret[0].(string)
	return ret0
}

// GetPort indicates an expected call of GetPort.
func (mr *MockWorkerNodeInterfaceMockRecorder) GetPort() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetPort", reflect.TypeOf((*MockWorkerNodeInterface)(nil).GetPort))
}

// GetSchedulability mocks base method.
func (m *MockWorkerNodeInterface) GetSchedulability() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetSchedulability")
	ret0, _ := ret[0].(bool)
	return ret0
}

// GetSchedulability indicates an expected call of GetSchedulability.
func (mr *MockWorkerNodeInterfaceMockRecorder) GetSchedulability() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetSchedulability", reflect.TypeOf((*MockWorkerNodeInterface)(nil).GetSchedulability))
}

// GetWorkerNodeConfiguration mocks base method.
func (m *MockWorkerNodeInterface) GetWorkerNodeConfiguration() core.WorkerNodeConfiguration {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetWorkerNodeConfiguration")
	ret0, _ := ret[0].(core.WorkerNodeConfiguration)
	return ret0
}

// GetWorkerNodeConfiguration indicates an expected call of GetWorkerNodeConfiguration.
func (mr *MockWorkerNodeInterfaceMockRecorder) GetWorkerNodeConfiguration() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetWorkerNodeConfiguration", reflect.TypeOf((*MockWorkerNodeInterface)(nil).GetWorkerNodeConfiguration))
}

// ListEndpoints mocks base method.
func (m *MockWorkerNodeInterface) ListEndpoints(arg0 context.Context, arg1 *emptypb.Empty, arg2 ...grpc.CallOption) (*proto.EndpointsList, error) {
	m.ctrl.T.Helper()
	varargs := []any{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "ListEndpoints", varargs...)
	ret0, _ := ret[0].(*proto.EndpointsList)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListEndpoints indicates an expected call of ListEndpoints.
func (mr *MockWorkerNodeInterfaceMockRecorder) ListEndpoints(arg0, arg1 any, arg2 ...any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]any{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListEndpoints", reflect.TypeOf((*MockWorkerNodeInterface)(nil).ListEndpoints), varargs...)
}

// SetCpuUsage mocks base method.
func (m *MockWorkerNodeInterface) SetCpuUsage(arg0 uint64) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SetCpuUsage", arg0)
}

// SetCpuUsage indicates an expected call of SetCpuUsage.
func (mr *MockWorkerNodeInterfaceMockRecorder) SetCpuUsage(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetCpuUsage", reflect.TypeOf((*MockWorkerNodeInterface)(nil).SetCpuUsage), arg0)
}

// SetMemoryUsage mocks base method.
func (m *MockWorkerNodeInterface) SetMemoryUsage(arg0 uint64) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SetMemoryUsage", arg0)
}

// SetMemoryUsage indicates an expected call of SetMemoryUsage.
func (mr *MockWorkerNodeInterfaceMockRecorder) SetMemoryUsage(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetMemoryUsage", reflect.TypeOf((*MockWorkerNodeInterface)(nil).SetMemoryUsage), arg0)
}

// SetSchedulability mocks base method.
func (m *MockWorkerNodeInterface) SetSchedulability(arg0 bool) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SetSchedulability", arg0)
}

// SetSchedulability indicates an expected call of SetSchedulability.
func (mr *MockWorkerNodeInterfaceMockRecorder) SetSchedulability(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetSchedulability", reflect.TypeOf((*MockWorkerNodeInterface)(nil).SetSchedulability), arg0)
}

// UpdateLastHearBeat mocks base method.
func (m *MockWorkerNodeInterface) UpdateLastHearBeat() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "UpdateLastHearBeat")
}

// UpdateLastHearBeat indicates an expected call of UpdateLastHearBeat.
func (mr *MockWorkerNodeInterfaceMockRecorder) UpdateLastHearBeat() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateLastHearBeat", reflect.TypeOf((*MockWorkerNodeInterface)(nil).UpdateLastHearBeat))
}

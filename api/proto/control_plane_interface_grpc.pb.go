// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.3.0
// - protoc             v3.19.1
// source: control_plane_interface.proto

package proto

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

const (
	CpiInterface_OnMetricsReceive_FullMethodName    = "/data_plane.CpiInterface/OnMetricsReceive"
	CpiInterface_ListServices_FullMethodName        = "/data_plane.CpiInterface/ListServices"
	CpiInterface_RegisterDataplane_FullMethodName   = "/data_plane.CpiInterface/RegisterDataplane"
	CpiInterface_RegisterService_FullMethodName     = "/data_plane.CpiInterface/RegisterService"
	CpiInterface_RegisterNode_FullMethodName        = "/data_plane.CpiInterface/RegisterNode"
	CpiInterface_NodeHeartbeat_FullMethodName       = "/data_plane.CpiInterface/NodeHeartbeat"
	CpiInterface_DeregisterDataplane_FullMethodName = "/data_plane.CpiInterface/DeregisterDataplane"
)

// CpiInterfaceClient is the client API for CpiInterface service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type CpiInterfaceClient interface {
	OnMetricsReceive(ctx context.Context, in *AutoscalingMetric, opts ...grpc.CallOption) (*ActionStatus, error)
	ListServices(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*ServiceList, error)
	RegisterDataplane(ctx context.Context, in *DataplaneInfo, opts ...grpc.CallOption) (*ActionStatus, error)
	RegisterService(ctx context.Context, in *ServiceInfo, opts ...grpc.CallOption) (*ActionStatus, error)
	RegisterNode(ctx context.Context, in *NodeInfo, opts ...grpc.CallOption) (*ActionStatus, error)
	NodeHeartbeat(ctx context.Context, in *NodeHeartbeatMessage, opts ...grpc.CallOption) (*ActionStatus, error)
	DeregisterDataplane(ctx context.Context, in *DataplaneInfo, opts ...grpc.CallOption) (*ActionStatus, error)
}

type cpiInterfaceClient struct {
	cc grpc.ClientConnInterface
}

func NewCpiInterfaceClient(cc grpc.ClientConnInterface) CpiInterfaceClient {
	return &cpiInterfaceClient{cc}
}

func (c *cpiInterfaceClient) OnMetricsReceive(ctx context.Context, in *AutoscalingMetric, opts ...grpc.CallOption) (*ActionStatus, error) {
	out := new(ActionStatus)
	err := c.cc.Invoke(ctx, CpiInterface_OnMetricsReceive_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *cpiInterfaceClient) ListServices(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*ServiceList, error) {
	out := new(ServiceList)
	err := c.cc.Invoke(ctx, CpiInterface_ListServices_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *cpiInterfaceClient) RegisterDataplane(ctx context.Context, in *DataplaneInfo, opts ...grpc.CallOption) (*ActionStatus, error) {
	out := new(ActionStatus)
	err := c.cc.Invoke(ctx, CpiInterface_RegisterDataplane_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *cpiInterfaceClient) RegisterService(ctx context.Context, in *ServiceInfo, opts ...grpc.CallOption) (*ActionStatus, error) {
	out := new(ActionStatus)
	err := c.cc.Invoke(ctx, CpiInterface_RegisterService_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *cpiInterfaceClient) RegisterNode(ctx context.Context, in *NodeInfo, opts ...grpc.CallOption) (*ActionStatus, error) {
	out := new(ActionStatus)
	err := c.cc.Invoke(ctx, CpiInterface_RegisterNode_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *cpiInterfaceClient) NodeHeartbeat(ctx context.Context, in *NodeHeartbeatMessage, opts ...grpc.CallOption) (*ActionStatus, error) {
	out := new(ActionStatus)
	err := c.cc.Invoke(ctx, CpiInterface_NodeHeartbeat_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *cpiInterfaceClient) DeregisterDataplane(ctx context.Context, in *DataplaneInfo, opts ...grpc.CallOption) (*ActionStatus, error) {
	out := new(ActionStatus)
	err := c.cc.Invoke(ctx, CpiInterface_DeregisterDataplane_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// CpiInterfaceServer is the server API for CpiInterface service.
// All implementations must embed UnimplementedCpiInterfaceServer
// for forward compatibility
type CpiInterfaceServer interface {
	OnMetricsReceive(context.Context, *AutoscalingMetric) (*ActionStatus, error)
	ListServices(context.Context, *emptypb.Empty) (*ServiceList, error)
	RegisterDataplane(context.Context, *DataplaneInfo) (*ActionStatus, error)
	RegisterService(context.Context, *ServiceInfo) (*ActionStatus, error)
	RegisterNode(context.Context, *NodeInfo) (*ActionStatus, error)
	NodeHeartbeat(context.Context, *NodeHeartbeatMessage) (*ActionStatus, error)
	DeregisterDataplane(context.Context, *DataplaneInfo) (*ActionStatus, error)
	mustEmbedUnimplementedCpiInterfaceServer()
}

// UnimplementedCpiInterfaceServer must be embedded to have forward compatible implementations.
type UnimplementedCpiInterfaceServer struct {
}

func (UnimplementedCpiInterfaceServer) OnMetricsReceive(context.Context, *AutoscalingMetric) (*ActionStatus, error) {
	return nil, status.Errorf(codes.Unimplemented, "method OnMetricsReceive not implemented")
}
func (UnimplementedCpiInterfaceServer) ListServices(context.Context, *emptypb.Empty) (*ServiceList, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListServices not implemented")
}
func (UnimplementedCpiInterfaceServer) RegisterDataplane(context.Context, *DataplaneInfo) (*ActionStatus, error) {
	return nil, status.Errorf(codes.Unimplemented, "method RegisterDataplane not implemented")
}
func (UnimplementedCpiInterfaceServer) RegisterService(context.Context, *ServiceInfo) (*ActionStatus, error) {
	return nil, status.Errorf(codes.Unimplemented, "method RegisterService not implemented")
}
func (UnimplementedCpiInterfaceServer) RegisterNode(context.Context, *NodeInfo) (*ActionStatus, error) {
	return nil, status.Errorf(codes.Unimplemented, "method RegisterNode not implemented")
}
func (UnimplementedCpiInterfaceServer) NodeHeartbeat(context.Context, *NodeHeartbeatMessage) (*ActionStatus, error) {
	return nil, status.Errorf(codes.Unimplemented, "method NodeHeartbeat not implemented")
}
func (UnimplementedCpiInterfaceServer) DeregisterDataplane(context.Context, *DataplaneInfo) (*ActionStatus, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DeregisterDataplane not implemented")
}
func (UnimplementedCpiInterfaceServer) mustEmbedUnimplementedCpiInterfaceServer() {}

// UnsafeCpiInterfaceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to CpiInterfaceServer will
// result in compilation errors.
type UnsafeCpiInterfaceServer interface {
	mustEmbedUnimplementedCpiInterfaceServer()
}

func RegisterCpiInterfaceServer(s grpc.ServiceRegistrar, srv CpiInterfaceServer) {
	s.RegisterService(&CpiInterface_ServiceDesc, srv)
}

func _CpiInterface_OnMetricsReceive_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(AutoscalingMetric)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CpiInterfaceServer).OnMetricsReceive(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: CpiInterface_OnMetricsReceive_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CpiInterfaceServer).OnMetricsReceive(ctx, req.(*AutoscalingMetric))
	}
	return interceptor(ctx, in, info, handler)
}

func _CpiInterface_ListServices_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(emptypb.Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CpiInterfaceServer).ListServices(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: CpiInterface_ListServices_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CpiInterfaceServer).ListServices(ctx, req.(*emptypb.Empty))
	}
	return interceptor(ctx, in, info, handler)
}

func _CpiInterface_RegisterDataplane_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DataplaneInfo)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CpiInterfaceServer).RegisterDataplane(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: CpiInterface_RegisterDataplane_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CpiInterfaceServer).RegisterDataplane(ctx, req.(*DataplaneInfo))
	}
	return interceptor(ctx, in, info, handler)
}

func _CpiInterface_RegisterService_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ServiceInfo)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CpiInterfaceServer).RegisterService(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: CpiInterface_RegisterService_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CpiInterfaceServer).RegisterService(ctx, req.(*ServiceInfo))
	}
	return interceptor(ctx, in, info, handler)
}

func _CpiInterface_RegisterNode_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(NodeInfo)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CpiInterfaceServer).RegisterNode(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: CpiInterface_RegisterNode_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CpiInterfaceServer).RegisterNode(ctx, req.(*NodeInfo))
	}
	return interceptor(ctx, in, info, handler)
}

func _CpiInterface_NodeHeartbeat_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(NodeHeartbeatMessage)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CpiInterfaceServer).NodeHeartbeat(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: CpiInterface_NodeHeartbeat_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CpiInterfaceServer).NodeHeartbeat(ctx, req.(*NodeHeartbeatMessage))
	}
	return interceptor(ctx, in, info, handler)
}

func _CpiInterface_DeregisterDataplane_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DataplaneInfo)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CpiInterfaceServer).DeregisterDataplane(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: CpiInterface_DeregisterDataplane_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CpiInterfaceServer).DeregisterDataplane(ctx, req.(*DataplaneInfo))
	}
	return interceptor(ctx, in, info, handler)
}

// CpiInterface_ServiceDesc is the grpc.ServiceDesc for CpiInterface service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var CpiInterface_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "data_plane.CpiInterface",
	HandlerType: (*CpiInterfaceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "OnMetricsReceive",
			Handler:    _CpiInterface_OnMetricsReceive_Handler,
		},
		{
			MethodName: "ListServices",
			Handler:    _CpiInterface_ListServices_Handler,
		},
		{
			MethodName: "RegisterDataplane",
			Handler:    _CpiInterface_RegisterDataplane_Handler,
		},
		{
			MethodName: "RegisterService",
			Handler:    _CpiInterface_RegisterService_Handler,
		},
		{
			MethodName: "RegisterNode",
			Handler:    _CpiInterface_RegisterNode_Handler,
		},
		{
			MethodName: "NodeHeartbeat",
			Handler:    _CpiInterface_NodeHeartbeat_Handler,
		},
		{
			MethodName: "DeregisterDataplane",
			Handler:    _CpiInterface_DeregisterDataplane_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "control_plane_interface.proto",
}

const (
	WorkerNodeInterface_CreateSandbox_FullMethodName = "/data_plane.WorkerNodeInterface/CreateSandbox"
	WorkerNodeInterface_DeleteSandbox_FullMethodName = "/data_plane.WorkerNodeInterface/DeleteSandbox"
)

// WorkerNodeInterfaceClient is the client API for WorkerNodeInterface service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type WorkerNodeInterfaceClient interface {
	CreateSandbox(ctx context.Context, in *ServiceInfo, opts ...grpc.CallOption) (*SandboxCreationStatus, error)
	DeleteSandbox(ctx context.Context, in *SandboxID, opts ...grpc.CallOption) (*ActionStatus, error)
}

type workerNodeInterfaceClient struct {
	cc grpc.ClientConnInterface
}

func NewWorkerNodeInterfaceClient(cc grpc.ClientConnInterface) WorkerNodeInterfaceClient {
	return &workerNodeInterfaceClient{cc}
}

func (c *workerNodeInterfaceClient) CreateSandbox(ctx context.Context, in *ServiceInfo, opts ...grpc.CallOption) (*SandboxCreationStatus, error) {
	out := new(SandboxCreationStatus)
	err := c.cc.Invoke(ctx, WorkerNodeInterface_CreateSandbox_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *workerNodeInterfaceClient) DeleteSandbox(ctx context.Context, in *SandboxID, opts ...grpc.CallOption) (*ActionStatus, error) {
	out := new(ActionStatus)
	err := c.cc.Invoke(ctx, WorkerNodeInterface_DeleteSandbox_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// WorkerNodeInterfaceServer is the server API for WorkerNodeInterface service.
// All implementations must embed UnimplementedWorkerNodeInterfaceServer
// for forward compatibility
type WorkerNodeInterfaceServer interface {
	CreateSandbox(context.Context, *ServiceInfo) (*SandboxCreationStatus, error)
	DeleteSandbox(context.Context, *SandboxID) (*ActionStatus, error)
	mustEmbedUnimplementedWorkerNodeInterfaceServer()
}

// UnimplementedWorkerNodeInterfaceServer must be embedded to have forward compatible implementations.
type UnimplementedWorkerNodeInterfaceServer struct {
}

func (UnimplementedWorkerNodeInterfaceServer) CreateSandbox(context.Context, *ServiceInfo) (*SandboxCreationStatus, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreateSandbox not implemented")
}
func (UnimplementedWorkerNodeInterfaceServer) DeleteSandbox(context.Context, *SandboxID) (*ActionStatus, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DeleteSandbox not implemented")
}
func (UnimplementedWorkerNodeInterfaceServer) mustEmbedUnimplementedWorkerNodeInterfaceServer() {}

// UnsafeWorkerNodeInterfaceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to WorkerNodeInterfaceServer will
// result in compilation errors.
type UnsafeWorkerNodeInterfaceServer interface {
	mustEmbedUnimplementedWorkerNodeInterfaceServer()
}

func RegisterWorkerNodeInterfaceServer(s grpc.ServiceRegistrar, srv WorkerNodeInterfaceServer) {
	s.RegisterService(&WorkerNodeInterface_ServiceDesc, srv)
}

func _WorkerNodeInterface_CreateSandbox_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ServiceInfo)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(WorkerNodeInterfaceServer).CreateSandbox(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: WorkerNodeInterface_CreateSandbox_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(WorkerNodeInterfaceServer).CreateSandbox(ctx, req.(*ServiceInfo))
	}
	return interceptor(ctx, in, info, handler)
}

func _WorkerNodeInterface_DeleteSandbox_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SandboxID)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(WorkerNodeInterfaceServer).DeleteSandbox(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: WorkerNodeInterface_DeleteSandbox_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(WorkerNodeInterfaceServer).DeleteSandbox(ctx, req.(*SandboxID))
	}
	return interceptor(ctx, in, info, handler)
}

// WorkerNodeInterface_ServiceDesc is the grpc.ServiceDesc for WorkerNodeInterface service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var WorkerNodeInterface_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "data_plane.WorkerNodeInterface",
	HandlerType: (*WorkerNodeInterfaceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "CreateSandbox",
			Handler:    _WorkerNodeInterface_CreateSandbox_Handler,
		},
		{
			MethodName: "DeleteSandbox",
			Handler:    _WorkerNodeInterface_DeleteSandbox_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "control_plane_interface.proto",
}

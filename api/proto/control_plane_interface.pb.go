// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.25.0-devel
// 	protoc        v3.14.0
// source: control_plane_interface.proto

package proto

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	durationpb "google.golang.org/protobuf/types/known/durationpb"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type NodeInfo struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	NodeID     string `protobuf:"bytes,1,opt,name=NodeID,proto3" json:"NodeID,omitempty"`
	IP         string `protobuf:"bytes,2,opt,name=IP,proto3" json:"IP,omitempty"`
	Port       int32  `protobuf:"varint,3,opt,name=Port,proto3" json:"Port,omitempty"`
	CpuCores   int32  `protobuf:"varint,4,opt,name=CpuCores,proto3" json:"CpuCores,omitempty"`
	MemorySize uint64 `protobuf:"varint,5,opt,name=MemorySize,proto3" json:"MemorySize,omitempty"`
}

func (x *NodeInfo) Reset() {
	*x = NodeInfo{}
	if protoimpl.UnsafeEnabled {
		mi := &file_control_plane_interface_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *NodeInfo) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*NodeInfo) ProtoMessage() {}

func (x *NodeInfo) ProtoReflect() protoreflect.Message {
	mi := &file_control_plane_interface_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use NodeInfo.ProtoReflect.Descriptor instead.
func (*NodeInfo) Descriptor() ([]byte, []int) {
	return file_control_plane_interface_proto_rawDescGZIP(), []int{0}
}

func (x *NodeInfo) GetNodeID() string {
	if x != nil {
		return x.NodeID
	}
	return ""
}

func (x *NodeInfo) GetIP() string {
	if x != nil {
		return x.IP
	}
	return ""
}

func (x *NodeInfo) GetPort() int32 {
	if x != nil {
		return x.Port
	}
	return 0
}

func (x *NodeInfo) GetCpuCores() int32 {
	if x != nil {
		return x.CpuCores
	}
	return 0
}

func (x *NodeInfo) GetMemorySize() uint64 {
	if x != nil {
		return x.MemorySize
	}
	return 0
}

type NodeHeartbeatMessage struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	NodeID      string `protobuf:"bytes,1,opt,name=NodeID,proto3" json:"NodeID,omitempty"`
	CpuUsage    int32  `protobuf:"varint,2,opt,name=CpuUsage,proto3" json:"CpuUsage,omitempty"`
	MemoryUsage int32  `protobuf:"varint,3,opt,name=MemoryUsage,proto3" json:"MemoryUsage,omitempty"`
}

func (x *NodeHeartbeatMessage) Reset() {
	*x = NodeHeartbeatMessage{}
	if protoimpl.UnsafeEnabled {
		mi := &file_control_plane_interface_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *NodeHeartbeatMessage) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*NodeHeartbeatMessage) ProtoMessage() {}

func (x *NodeHeartbeatMessage) ProtoReflect() protoreflect.Message {
	mi := &file_control_plane_interface_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use NodeHeartbeatMessage.ProtoReflect.Descriptor instead.
func (*NodeHeartbeatMessage) Descriptor() ([]byte, []int) {
	return file_control_plane_interface_proto_rawDescGZIP(), []int{1}
}

func (x *NodeHeartbeatMessage) GetNodeID() string {
	if x != nil {
		return x.NodeID
	}
	return ""
}

func (x *NodeHeartbeatMessage) GetCpuUsage() int32 {
	if x != nil {
		return x.CpuUsage
	}
	return 0
}

func (x *NodeHeartbeatMessage) GetMemoryUsage() int32 {
	if x != nil {
		return x.MemoryUsage
	}
	return 0
}

type AutoscalingMetric struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	ServiceName string  `protobuf:"bytes,1,opt,name=ServiceName,proto3" json:"ServiceName,omitempty"`
	Metric      float32 `protobuf:"fixed32,2,opt,name=Metric,proto3" json:"Metric,omitempty"`
}

func (x *AutoscalingMetric) Reset() {
	*x = AutoscalingMetric{}
	if protoimpl.UnsafeEnabled {
		mi := &file_control_plane_interface_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *AutoscalingMetric) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*AutoscalingMetric) ProtoMessage() {}

func (x *AutoscalingMetric) ProtoReflect() protoreflect.Message {
	mi := &file_control_plane_interface_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use AutoscalingMetric.ProtoReflect.Descriptor instead.
func (*AutoscalingMetric) Descriptor() ([]byte, []int) {
	return file_control_plane_interface_proto_rawDescGZIP(), []int{2}
}

func (x *AutoscalingMetric) GetServiceName() string {
	if x != nil {
		return x.ServiceName
	}
	return ""
}

func (x *AutoscalingMetric) GetMetric() float32 {
	if x != nil {
		return x.Metric
	}
	return 0
}

type DataplaneInfo struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	APIPort   int32 `protobuf:"varint,1,opt,name=APIPort,proto3" json:"APIPort,omitempty"`
	ProxyPort int32 `protobuf:"varint,2,opt,name=ProxyPort,proto3" json:"ProxyPort,omitempty"`
}

func (x *DataplaneInfo) Reset() {
	*x = DataplaneInfo{}
	if protoimpl.UnsafeEnabled {
		mi := &file_control_plane_interface_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *DataplaneInfo) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DataplaneInfo) ProtoMessage() {}

func (x *DataplaneInfo) ProtoReflect() protoreflect.Message {
	mi := &file_control_plane_interface_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DataplaneInfo.ProtoReflect.Descriptor instead.
func (*DataplaneInfo) Descriptor() ([]byte, []int) {
	return file_control_plane_interface_proto_rawDescGZIP(), []int{3}
}

func (x *DataplaneInfo) GetAPIPort() int32 {
	if x != nil {
		return x.APIPort
	}
	return 0
}

func (x *DataplaneInfo) GetProxyPort() int32 {
	if x != nil {
		return x.ProxyPort
	}
	return 0
}

type SandboxID struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	ID       string `protobuf:"bytes,1,opt,name=ID,proto3" json:"ID,omitempty"`
	HostPort int32  `protobuf:"varint,2,opt,name=HostPort,proto3" json:"HostPort,omitempty"`
}

func (x *SandboxID) Reset() {
	*x = SandboxID{}
	if protoimpl.UnsafeEnabled {
		mi := &file_control_plane_interface_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *SandboxID) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SandboxID) ProtoMessage() {}

func (x *SandboxID) ProtoReflect() protoreflect.Message {
	mi := &file_control_plane_interface_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use SandboxID.ProtoReflect.Descriptor instead.
func (*SandboxID) Descriptor() ([]byte, []int) {
	return file_control_plane_interface_proto_rawDescGZIP(), []int{4}
}

func (x *SandboxID) GetID() string {
	if x != nil {
		return x.ID
	}
	return ""
}

func (x *SandboxID) GetHostPort() int32 {
	if x != nil {
		return x.HostPort
	}
	return 0
}

type SandboxCreationStatus struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Success          bool                      `protobuf:"varint,1,opt,name=Success,proto3" json:"Success,omitempty"`
	ID               string                    `protobuf:"bytes,2,opt,name=ID,proto3" json:"ID,omitempty"`
	PortMappings     *PortMapping              `protobuf:"bytes,3,opt,name=PortMappings,proto3" json:"PortMappings,omitempty"`
	LatencyBreakdown *SandboxCreationBreakdown `protobuf:"bytes,5,opt,name=LatencyBreakdown,proto3" json:"LatencyBreakdown,omitempty"`
}

func (x *SandboxCreationStatus) Reset() {
	*x = SandboxCreationStatus{}
	if protoimpl.UnsafeEnabled {
		mi := &file_control_plane_interface_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *SandboxCreationStatus) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SandboxCreationStatus) ProtoMessage() {}

func (x *SandboxCreationStatus) ProtoReflect() protoreflect.Message {
	mi := &file_control_plane_interface_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use SandboxCreationStatus.ProtoReflect.Descriptor instead.
func (*SandboxCreationStatus) Descriptor() ([]byte, []int) {
	return file_control_plane_interface_proto_rawDescGZIP(), []int{5}
}

func (x *SandboxCreationStatus) GetSuccess() bool {
	if x != nil {
		return x.Success
	}
	return false
}

func (x *SandboxCreationStatus) GetID() string {
	if x != nil {
		return x.ID
	}
	return ""
}

func (x *SandboxCreationStatus) GetPortMappings() *PortMapping {
	if x != nil {
		return x.PortMappings
	}
	return nil
}

func (x *SandboxCreationStatus) GetLatencyBreakdown() *SandboxCreationBreakdown {
	if x != nil {
		return x.LatencyBreakdown
	}
	return nil
}

type SandboxCreationBreakdown struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Total           *durationpb.Duration `protobuf:"bytes,1,opt,name=Total,proto3" json:"Total,omitempty"`
	ImageFetch      *durationpb.Duration `protobuf:"bytes,2,opt,name=ImageFetch,proto3" json:"ImageFetch,omitempty"`
	ContainerCreate *durationpb.Duration `protobuf:"bytes,3,opt,name=ContainerCreate,proto3" json:"ContainerCreate,omitempty"`
	CNI             *durationpb.Duration `protobuf:"bytes,4,opt,name=CNI,proto3" json:"CNI,omitempty"`
	ContainerStart  *durationpb.Duration `protobuf:"bytes,5,opt,name=ContainerStart,proto3" json:"ContainerStart,omitempty"`
	Iptables        *durationpb.Duration `protobuf:"bytes,6,opt,name=Iptables,proto3" json:"Iptables,omitempty"`
}

func (x *SandboxCreationBreakdown) Reset() {
	*x = SandboxCreationBreakdown{}
	if protoimpl.UnsafeEnabled {
		mi := &file_control_plane_interface_proto_msgTypes[6]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *SandboxCreationBreakdown) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SandboxCreationBreakdown) ProtoMessage() {}

func (x *SandboxCreationBreakdown) ProtoReflect() protoreflect.Message {
	mi := &file_control_plane_interface_proto_msgTypes[6]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use SandboxCreationBreakdown.ProtoReflect.Descriptor instead.
func (*SandboxCreationBreakdown) Descriptor() ([]byte, []int) {
	return file_control_plane_interface_proto_rawDescGZIP(), []int{6}
}

func (x *SandboxCreationBreakdown) GetTotal() *durationpb.Duration {
	if x != nil {
		return x.Total
	}
	return nil
}

func (x *SandboxCreationBreakdown) GetImageFetch() *durationpb.Duration {
	if x != nil {
		return x.ImageFetch
	}
	return nil
}

func (x *SandboxCreationBreakdown) GetContainerCreate() *durationpb.Duration {
	if x != nil {
		return x.ContainerCreate
	}
	return nil
}

func (x *SandboxCreationBreakdown) GetCNI() *durationpb.Duration {
	if x != nil {
		return x.CNI
	}
	return nil
}

func (x *SandboxCreationBreakdown) GetContainerStart() *durationpb.Duration {
	if x != nil {
		return x.ContainerStart
	}
	return nil
}

func (x *SandboxCreationBreakdown) GetIptables() *durationpb.Duration {
	if x != nil {
		return x.Iptables
	}
	return nil
}

var File_control_plane_interface_proto protoreflect.FileDescriptor

var file_control_plane_interface_proto_rawDesc = []byte{
	0x0a, 0x1d, 0x63, 0x6f, 0x6e, 0x74, 0x72, 0x6f, 0x6c, 0x5f, 0x70, 0x6c, 0x61, 0x6e, 0x65, 0x5f,
	0x69, 0x6e, 0x74, 0x65, 0x72, 0x66, 0x61, 0x63, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12,
	0x0a, 0x64, 0x61, 0x74, 0x61, 0x5f, 0x70, 0x6c, 0x61, 0x6e, 0x65, 0x1a, 0x0c, 0x63, 0x6f, 0x6d,
	0x6d, 0x6f, 0x6e, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x1b, 0x67, 0x6f, 0x6f, 0x67, 0x6c,
	0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x65, 0x6d, 0x70, 0x74, 0x79,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x1e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x64, 0x75, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x82, 0x01, 0x0a, 0x08, 0x4e, 0x6f, 0x64, 0x65, 0x49,
	0x6e, 0x66, 0x6f, 0x12, 0x16, 0x0a, 0x06, 0x4e, 0x6f, 0x64, 0x65, 0x49, 0x44, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x06, 0x4e, 0x6f, 0x64, 0x65, 0x49, 0x44, 0x12, 0x0e, 0x0a, 0x02, 0x49,
	0x50, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x02, 0x49, 0x50, 0x12, 0x12, 0x0a, 0x04, 0x50,
	0x6f, 0x72, 0x74, 0x18, 0x03, 0x20, 0x01, 0x28, 0x05, 0x52, 0x04, 0x50, 0x6f, 0x72, 0x74, 0x12,
	0x1a, 0x0a, 0x08, 0x43, 0x70, 0x75, 0x43, 0x6f, 0x72, 0x65, 0x73, 0x18, 0x04, 0x20, 0x01, 0x28,
	0x05, 0x52, 0x08, 0x43, 0x70, 0x75, 0x43, 0x6f, 0x72, 0x65, 0x73, 0x12, 0x1e, 0x0a, 0x0a, 0x4d,
	0x65, 0x6d, 0x6f, 0x72, 0x79, 0x53, 0x69, 0x7a, 0x65, 0x18, 0x05, 0x20, 0x01, 0x28, 0x04, 0x52,
	0x0a, 0x4d, 0x65, 0x6d, 0x6f, 0x72, 0x79, 0x53, 0x69, 0x7a, 0x65, 0x22, 0x6c, 0x0a, 0x14, 0x4e,
	0x6f, 0x64, 0x65, 0x48, 0x65, 0x61, 0x72, 0x74, 0x62, 0x65, 0x61, 0x74, 0x4d, 0x65, 0x73, 0x73,
	0x61, 0x67, 0x65, 0x12, 0x16, 0x0a, 0x06, 0x4e, 0x6f, 0x64, 0x65, 0x49, 0x44, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x06, 0x4e, 0x6f, 0x64, 0x65, 0x49, 0x44, 0x12, 0x1a, 0x0a, 0x08, 0x43,
	0x70, 0x75, 0x55, 0x73, 0x61, 0x67, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x05, 0x52, 0x08, 0x43,
	0x70, 0x75, 0x55, 0x73, 0x61, 0x67, 0x65, 0x12, 0x20, 0x0a, 0x0b, 0x4d, 0x65, 0x6d, 0x6f, 0x72,
	0x79, 0x55, 0x73, 0x61, 0x67, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x05, 0x52, 0x0b, 0x4d, 0x65,
	0x6d, 0x6f, 0x72, 0x79, 0x55, 0x73, 0x61, 0x67, 0x65, 0x22, 0x4d, 0x0a, 0x11, 0x41, 0x75, 0x74,
	0x6f, 0x73, 0x63, 0x61, 0x6c, 0x69, 0x6e, 0x67, 0x4d, 0x65, 0x74, 0x72, 0x69, 0x63, 0x12, 0x20,
	0x0a, 0x0b, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x4e, 0x61, 0x6d, 0x65, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x0b, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x4e, 0x61, 0x6d, 0x65,
	0x12, 0x16, 0x0a, 0x06, 0x4d, 0x65, 0x74, 0x72, 0x69, 0x63, 0x18, 0x02, 0x20, 0x01, 0x28, 0x02,
	0x52, 0x06, 0x4d, 0x65, 0x74, 0x72, 0x69, 0x63, 0x22, 0x47, 0x0a, 0x0d, 0x44, 0x61, 0x74, 0x61,
	0x70, 0x6c, 0x61, 0x6e, 0x65, 0x49, 0x6e, 0x66, 0x6f, 0x12, 0x18, 0x0a, 0x07, 0x41, 0x50, 0x49,
	0x50, 0x6f, 0x72, 0x74, 0x18, 0x01, 0x20, 0x01, 0x28, 0x05, 0x52, 0x07, 0x41, 0x50, 0x49, 0x50,
	0x6f, 0x72, 0x74, 0x12, 0x1c, 0x0a, 0x09, 0x50, 0x72, 0x6f, 0x78, 0x79, 0x50, 0x6f, 0x72, 0x74,
	0x18, 0x02, 0x20, 0x01, 0x28, 0x05, 0x52, 0x09, 0x50, 0x72, 0x6f, 0x78, 0x79, 0x50, 0x6f, 0x72,
	0x74, 0x22, 0x37, 0x0a, 0x09, 0x53, 0x61, 0x6e, 0x64, 0x62, 0x6f, 0x78, 0x49, 0x44, 0x12, 0x0e,
	0x0a, 0x02, 0x49, 0x44, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x02, 0x49, 0x44, 0x12, 0x1a,
	0x0a, 0x08, 0x48, 0x6f, 0x73, 0x74, 0x50, 0x6f, 0x72, 0x74, 0x18, 0x02, 0x20, 0x01, 0x28, 0x05,
	0x52, 0x08, 0x48, 0x6f, 0x73, 0x74, 0x50, 0x6f, 0x72, 0x74, 0x22, 0xd0, 0x01, 0x0a, 0x15, 0x53,
	0x61, 0x6e, 0x64, 0x62, 0x6f, 0x78, 0x43, 0x72, 0x65, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x53, 0x74,
	0x61, 0x74, 0x75, 0x73, 0x12, 0x18, 0x0a, 0x07, 0x53, 0x75, 0x63, 0x63, 0x65, 0x73, 0x73, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x08, 0x52, 0x07, 0x53, 0x75, 0x63, 0x63, 0x65, 0x73, 0x73, 0x12, 0x0e,
	0x0a, 0x02, 0x49, 0x44, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x02, 0x49, 0x44, 0x12, 0x3b,
	0x0a, 0x0c, 0x50, 0x6f, 0x72, 0x74, 0x4d, 0x61, 0x70, 0x70, 0x69, 0x6e, 0x67, 0x73, 0x18, 0x03,
	0x20, 0x01, 0x28, 0x0b, 0x32, 0x17, 0x2e, 0x64, 0x61, 0x74, 0x61, 0x5f, 0x70, 0x6c, 0x61, 0x6e,
	0x65, 0x2e, 0x50, 0x6f, 0x72, 0x74, 0x4d, 0x61, 0x70, 0x70, 0x69, 0x6e, 0x67, 0x52, 0x0c, 0x50,
	0x6f, 0x72, 0x74, 0x4d, 0x61, 0x70, 0x70, 0x69, 0x6e, 0x67, 0x73, 0x12, 0x50, 0x0a, 0x10, 0x4c,
	0x61, 0x74, 0x65, 0x6e, 0x63, 0x79, 0x42, 0x72, 0x65, 0x61, 0x6b, 0x64, 0x6f, 0x77, 0x6e, 0x18,
	0x05, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x24, 0x2e, 0x64, 0x61, 0x74, 0x61, 0x5f, 0x70, 0x6c, 0x61,
	0x6e, 0x65, 0x2e, 0x53, 0x61, 0x6e, 0x64, 0x62, 0x6f, 0x78, 0x43, 0x72, 0x65, 0x61, 0x74, 0x69,
	0x6f, 0x6e, 0x42, 0x72, 0x65, 0x61, 0x6b, 0x64, 0x6f, 0x77, 0x6e, 0x52, 0x10, 0x4c, 0x61, 0x74,
	0x65, 0x6e, 0x63, 0x79, 0x42, 0x72, 0x65, 0x61, 0x6b, 0x64, 0x6f, 0x77, 0x6e, 0x22, 0xf2, 0x02,
	0x0a, 0x18, 0x53, 0x61, 0x6e, 0x64, 0x62, 0x6f, 0x78, 0x43, 0x72, 0x65, 0x61, 0x74, 0x69, 0x6f,
	0x6e, 0x42, 0x72, 0x65, 0x61, 0x6b, 0x64, 0x6f, 0x77, 0x6e, 0x12, 0x2f, 0x0a, 0x05, 0x54, 0x6f,
	0x74, 0x61, 0x6c, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x19, 0x2e, 0x67, 0x6f, 0x6f, 0x67,
	0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x44, 0x75, 0x72, 0x61,
	0x74, 0x69, 0x6f, 0x6e, 0x52, 0x05, 0x54, 0x6f, 0x74, 0x61, 0x6c, 0x12, 0x39, 0x0a, 0x0a, 0x49,
	0x6d, 0x61, 0x67, 0x65, 0x46, 0x65, 0x74, 0x63, 0x68, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32,
	0x19, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75,
	0x66, 0x2e, 0x44, 0x75, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x0a, 0x49, 0x6d, 0x61, 0x67,
	0x65, 0x46, 0x65, 0x74, 0x63, 0x68, 0x12, 0x43, 0x0a, 0x0f, 0x43, 0x6f, 0x6e, 0x74, 0x61, 0x69,
	0x6e, 0x65, 0x72, 0x43, 0x72, 0x65, 0x61, 0x74, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b, 0x32,
	0x19, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75,
	0x66, 0x2e, 0x44, 0x75, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x0f, 0x43, 0x6f, 0x6e, 0x74,
	0x61, 0x69, 0x6e, 0x65, 0x72, 0x43, 0x72, 0x65, 0x61, 0x74, 0x65, 0x12, 0x2b, 0x0a, 0x03, 0x43,
	0x4e, 0x49, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x19, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c,
	0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x44, 0x75, 0x72, 0x61, 0x74,
	0x69, 0x6f, 0x6e, 0x52, 0x03, 0x43, 0x4e, 0x49, 0x12, 0x41, 0x0a, 0x0e, 0x43, 0x6f, 0x6e, 0x74,
	0x61, 0x69, 0x6e, 0x65, 0x72, 0x53, 0x74, 0x61, 0x72, 0x74, 0x18, 0x05, 0x20, 0x01, 0x28, 0x0b,
	0x32, 0x19, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62,
	0x75, 0x66, 0x2e, 0x44, 0x75, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x0e, 0x43, 0x6f, 0x6e,
	0x74, 0x61, 0x69, 0x6e, 0x65, 0x72, 0x53, 0x74, 0x61, 0x72, 0x74, 0x12, 0x35, 0x0a, 0x08, 0x49,
	0x70, 0x74, 0x61, 0x62, 0x6c, 0x65, 0x73, 0x18, 0x06, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x19, 0x2e,
	0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e,
	0x44, 0x75, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x08, 0x49, 0x70, 0x74, 0x61, 0x62, 0x6c,
	0x65, 0x73, 0x32, 0xb9, 0x03, 0x0a, 0x0c, 0x43, 0x70, 0x69, 0x49, 0x6e, 0x74, 0x65, 0x72, 0x66,
	0x61, 0x63, 0x65, 0x12, 0x4b, 0x0a, 0x10, 0x4f, 0x6e, 0x4d, 0x65, 0x74, 0x72, 0x69, 0x63, 0x73,
	0x52, 0x65, 0x63, 0x65, 0x69, 0x76, 0x65, 0x12, 0x1d, 0x2e, 0x64, 0x61, 0x74, 0x61, 0x5f, 0x70,
	0x6c, 0x61, 0x6e, 0x65, 0x2e, 0x41, 0x75, 0x74, 0x6f, 0x73, 0x63, 0x61, 0x6c, 0x69, 0x6e, 0x67,
	0x4d, 0x65, 0x74, 0x72, 0x69, 0x63, 0x1a, 0x18, 0x2e, 0x64, 0x61, 0x74, 0x61, 0x5f, 0x70, 0x6c,
	0x61, 0x6e, 0x65, 0x2e, 0x41, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73,
	0x12, 0x3f, 0x0a, 0x0c, 0x4c, 0x69, 0x73, 0x74, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x73,
	0x12, 0x16, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62,
	0x75, 0x66, 0x2e, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x1a, 0x17, 0x2e, 0x64, 0x61, 0x74, 0x61, 0x5f,
	0x70, 0x6c, 0x61, 0x6e, 0x65, 0x2e, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x4c, 0x69, 0x73,
	0x74, 0x12, 0x48, 0x0a, 0x11, 0x52, 0x65, 0x67, 0x69, 0x73, 0x74, 0x65, 0x72, 0x44, 0x61, 0x74,
	0x61, 0x70, 0x6c, 0x61, 0x6e, 0x65, 0x12, 0x19, 0x2e, 0x64, 0x61, 0x74, 0x61, 0x5f, 0x70, 0x6c,
	0x61, 0x6e, 0x65, 0x2e, 0x44, 0x61, 0x74, 0x61, 0x70, 0x6c, 0x61, 0x6e, 0x65, 0x49, 0x6e, 0x66,
	0x6f, 0x1a, 0x18, 0x2e, 0x64, 0x61, 0x74, 0x61, 0x5f, 0x70, 0x6c, 0x61, 0x6e, 0x65, 0x2e, 0x41,
	0x63, 0x74, 0x69, 0x6f, 0x6e, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x12, 0x44, 0x0a, 0x0f, 0x52,
	0x65, 0x67, 0x69, 0x73, 0x74, 0x65, 0x72, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x12, 0x17,
	0x2e, 0x64, 0x61, 0x74, 0x61, 0x5f, 0x70, 0x6c, 0x61, 0x6e, 0x65, 0x2e, 0x53, 0x65, 0x72, 0x76,
	0x69, 0x63, 0x65, 0x49, 0x6e, 0x66, 0x6f, 0x1a, 0x18, 0x2e, 0x64, 0x61, 0x74, 0x61, 0x5f, 0x70,
	0x6c, 0x61, 0x6e, 0x65, 0x2e, 0x41, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x53, 0x74, 0x61, 0x74, 0x75,
	0x73, 0x12, 0x3e, 0x0a, 0x0c, 0x52, 0x65, 0x67, 0x69, 0x73, 0x74, 0x65, 0x72, 0x4e, 0x6f, 0x64,
	0x65, 0x12, 0x14, 0x2e, 0x64, 0x61, 0x74, 0x61, 0x5f, 0x70, 0x6c, 0x61, 0x6e, 0x65, 0x2e, 0x4e,
	0x6f, 0x64, 0x65, 0x49, 0x6e, 0x66, 0x6f, 0x1a, 0x18, 0x2e, 0x64, 0x61, 0x74, 0x61, 0x5f, 0x70,
	0x6c, 0x61, 0x6e, 0x65, 0x2e, 0x41, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x53, 0x74, 0x61, 0x74, 0x75,
	0x73, 0x12, 0x4b, 0x0a, 0x0d, 0x4e, 0x6f, 0x64, 0x65, 0x48, 0x65, 0x61, 0x72, 0x74, 0x62, 0x65,
	0x61, 0x74, 0x12, 0x20, 0x2e, 0x64, 0x61, 0x74, 0x61, 0x5f, 0x70, 0x6c, 0x61, 0x6e, 0x65, 0x2e,
	0x4e, 0x6f, 0x64, 0x65, 0x48, 0x65, 0x61, 0x72, 0x74, 0x62, 0x65, 0x61, 0x74, 0x4d, 0x65, 0x73,
	0x73, 0x61, 0x67, 0x65, 0x1a, 0x18, 0x2e, 0x64, 0x61, 0x74, 0x61, 0x5f, 0x70, 0x6c, 0x61, 0x6e,
	0x65, 0x2e, 0x41, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x32, 0xa4,
	0x01, 0x0a, 0x13, 0x57, 0x6f, 0x72, 0x6b, 0x65, 0x72, 0x4e, 0x6f, 0x64, 0x65, 0x49, 0x6e, 0x74,
	0x65, 0x72, 0x66, 0x61, 0x63, 0x65, 0x12, 0x4b, 0x0a, 0x0d, 0x43, 0x72, 0x65, 0x61, 0x74, 0x65,
	0x53, 0x61, 0x6e, 0x64, 0x62, 0x6f, 0x78, 0x12, 0x17, 0x2e, 0x64, 0x61, 0x74, 0x61, 0x5f, 0x70,
	0x6c, 0x61, 0x6e, 0x65, 0x2e, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x49, 0x6e, 0x66, 0x6f,
	0x1a, 0x21, 0x2e, 0x64, 0x61, 0x74, 0x61, 0x5f, 0x70, 0x6c, 0x61, 0x6e, 0x65, 0x2e, 0x53, 0x61,
	0x6e, 0x64, 0x62, 0x6f, 0x78, 0x43, 0x72, 0x65, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x53, 0x74, 0x61,
	0x74, 0x75, 0x73, 0x12, 0x40, 0x0a, 0x0d, 0x44, 0x65, 0x6c, 0x65, 0x74, 0x65, 0x53, 0x61, 0x6e,
	0x64, 0x62, 0x6f, 0x78, 0x12, 0x15, 0x2e, 0x64, 0x61, 0x74, 0x61, 0x5f, 0x70, 0x6c, 0x61, 0x6e,
	0x65, 0x2e, 0x53, 0x61, 0x6e, 0x64, 0x62, 0x6f, 0x78, 0x49, 0x44, 0x1a, 0x18, 0x2e, 0x64, 0x61,
	0x74, 0x61, 0x5f, 0x70, 0x6c, 0x61, 0x6e, 0x65, 0x2e, 0x41, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x53,
	0x74, 0x61, 0x74, 0x75, 0x73, 0x42, 0x2f, 0x5a, 0x2d, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e,
	0x63, 0x6f, 0x6d, 0x2f, 0x65, 0x74, 0x68, 0x2d, 0x65, 0x61, 0x73, 0x6c, 0x2f, 0x63, 0x6c, 0x75,
	0x73, 0x74, 0x65, 0x72, 0x5f, 0x6d, 0x61, 0x6e, 0x61, 0x67, 0x65, 0x72, 0x2f, 0x61, 0x70, 0x69,
	0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_control_plane_interface_proto_rawDescOnce sync.Once
	file_control_plane_interface_proto_rawDescData = file_control_plane_interface_proto_rawDesc
)

func file_control_plane_interface_proto_rawDescGZIP() []byte {
	file_control_plane_interface_proto_rawDescOnce.Do(func() {
		file_control_plane_interface_proto_rawDescData = protoimpl.X.CompressGZIP(file_control_plane_interface_proto_rawDescData)
	})
	return file_control_plane_interface_proto_rawDescData
}

var file_control_plane_interface_proto_msgTypes = make([]protoimpl.MessageInfo, 7)
var file_control_plane_interface_proto_goTypes = []interface{}{
	(*NodeInfo)(nil),                 // 0: data_plane.NodeInfo
	(*NodeHeartbeatMessage)(nil),     // 1: data_plane.NodeHeartbeatMessage
	(*AutoscalingMetric)(nil),        // 2: data_plane.AutoscalingMetric
	(*DataplaneInfo)(nil),            // 3: data_plane.DataplaneInfo
	(*SandboxID)(nil),                // 4: data_plane.SandboxID
	(*SandboxCreationStatus)(nil),    // 5: data_plane.SandboxCreationStatus
	(*SandboxCreationBreakdown)(nil), // 6: data_plane.SandboxCreationBreakdown
	(*PortMapping)(nil),              // 7: data_plane.PortMapping
	(*durationpb.Duration)(nil),      // 8: google.protobuf.Duration
	(*emptypb.Empty)(nil),            // 9: google.protobuf.Empty
	(*ServiceInfo)(nil),              // 10: data_plane.ServiceInfo
	(*ActionStatus)(nil),             // 11: data_plane.ActionStatus
	(*ServiceList)(nil),              // 12: data_plane.ServiceList
}
var file_control_plane_interface_proto_depIdxs = []int32{
	7,  // 0: data_plane.SandboxCreationStatus.PortMappings:type_name -> data_plane.PortMapping
	6,  // 1: data_plane.SandboxCreationStatus.LatencyBreakdown:type_name -> data_plane.SandboxCreationBreakdown
	8,  // 2: data_plane.SandboxCreationBreakdown.Total:type_name -> google.protobuf.Duration
	8,  // 3: data_plane.SandboxCreationBreakdown.ImageFetch:type_name -> google.protobuf.Duration
	8,  // 4: data_plane.SandboxCreationBreakdown.ContainerCreate:type_name -> google.protobuf.Duration
	8,  // 5: data_plane.SandboxCreationBreakdown.CNI:type_name -> google.protobuf.Duration
	8,  // 6: data_plane.SandboxCreationBreakdown.ContainerStart:type_name -> google.protobuf.Duration
	8,  // 7: data_plane.SandboxCreationBreakdown.Iptables:type_name -> google.protobuf.Duration
	2,  // 8: data_plane.CpiInterface.OnMetricsReceive:input_type -> data_plane.AutoscalingMetric
	9,  // 9: data_plane.CpiInterface.ListServices:input_type -> google.protobuf.Empty
	3,  // 10: data_plane.CpiInterface.RegisterDataplane:input_type -> data_plane.DataplaneInfo
	10, // 11: data_plane.CpiInterface.RegisterService:input_type -> data_plane.ServiceInfo
	0,  // 12: data_plane.CpiInterface.RegisterNode:input_type -> data_plane.NodeInfo
	1,  // 13: data_plane.CpiInterface.NodeHeartbeat:input_type -> data_plane.NodeHeartbeatMessage
	10, // 14: data_plane.WorkerNodeInterface.CreateSandbox:input_type -> data_plane.ServiceInfo
	4,  // 15: data_plane.WorkerNodeInterface.DeleteSandbox:input_type -> data_plane.SandboxID
	11, // 16: data_plane.CpiInterface.OnMetricsReceive:output_type -> data_plane.ActionStatus
	12, // 17: data_plane.CpiInterface.ListServices:output_type -> data_plane.ServiceList
	11, // 18: data_plane.CpiInterface.RegisterDataplane:output_type -> data_plane.ActionStatus
	11, // 19: data_plane.CpiInterface.RegisterService:output_type -> data_plane.ActionStatus
	11, // 20: data_plane.CpiInterface.RegisterNode:output_type -> data_plane.ActionStatus
	11, // 21: data_plane.CpiInterface.NodeHeartbeat:output_type -> data_plane.ActionStatus
	5,  // 22: data_plane.WorkerNodeInterface.CreateSandbox:output_type -> data_plane.SandboxCreationStatus
	11, // 23: data_plane.WorkerNodeInterface.DeleteSandbox:output_type -> data_plane.ActionStatus
	16, // [16:24] is the sub-list for method output_type
	8,  // [8:16] is the sub-list for method input_type
	8,  // [8:8] is the sub-list for extension type_name
	8,  // [8:8] is the sub-list for extension extendee
	0,  // [0:8] is the sub-list for field type_name
}

func init() { file_control_plane_interface_proto_init() }
func file_control_plane_interface_proto_init() {
	if File_control_plane_interface_proto != nil {
		return
	}
	file_common_proto_init()
	if !protoimpl.UnsafeEnabled {
		file_control_plane_interface_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*NodeInfo); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_control_plane_interface_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*NodeHeartbeatMessage); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_control_plane_interface_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*AutoscalingMetric); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_control_plane_interface_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*DataplaneInfo); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_control_plane_interface_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*SandboxID); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_control_plane_interface_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*SandboxCreationStatus); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_control_plane_interface_proto_msgTypes[6].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*SandboxCreationBreakdown); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_control_plane_interface_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   7,
			NumExtensions: 0,
			NumServices:   2,
		},
		GoTypes:           file_control_plane_interface_proto_goTypes,
		DependencyIndexes: file_control_plane_interface_proto_depIdxs,
		MessageInfos:      file_control_plane_interface_proto_msgTypes,
	}.Build()
	File_control_plane_interface_proto = out.File
	file_control_plane_interface_proto_rawDesc = nil
	file_control_plane_interface_proto_goTypes = nil
	file_control_plane_interface_proto_depIdxs = nil
}

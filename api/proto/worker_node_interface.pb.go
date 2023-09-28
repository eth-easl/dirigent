// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.30.0
// 	protoc        v3.19.1
// source: worker_node_interface.proto

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
		mi := &file_worker_node_interface_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *SandboxID) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SandboxID) ProtoMessage() {}

func (x *SandboxID) ProtoReflect() protoreflect.Message {
	mi := &file_worker_node_interface_proto_msgTypes[0]
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
	return file_worker_node_interface_proto_rawDescGZIP(), []int{0}
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
		mi := &file_worker_node_interface_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *SandboxCreationStatus) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SandboxCreationStatus) ProtoMessage() {}

func (x *SandboxCreationStatus) ProtoReflect() protoreflect.Message {
	mi := &file_worker_node_interface_proto_msgTypes[1]
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
	return file_worker_node_interface_proto_rawDescGZIP(), []int{1}
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

	Total                *durationpb.Duration `protobuf:"bytes,1,opt,name=Total,proto3" json:"Total,omitempty"`
	ImageFetch           *durationpb.Duration `protobuf:"bytes,2,opt,name=ImageFetch,proto3" json:"ImageFetch,omitempty"`
	SandboxCreate        *durationpb.Duration `protobuf:"bytes,3,opt,name=SandboxCreate,proto3" json:"SandboxCreate,omitempty"`
	NetworkSetup         *durationpb.Duration `protobuf:"bytes,4,opt,name=NetworkSetup,proto3" json:"NetworkSetup,omitempty"`
	SandboxStart         *durationpb.Duration `protobuf:"bytes,5,opt,name=SandboxStart,proto3" json:"SandboxStart,omitempty"`
	Iptables             *durationpb.Duration `protobuf:"bytes,6,opt,name=Iptables,proto3" json:"Iptables,omitempty"`
	DataplanePropagation *durationpb.Duration `protobuf:"bytes,7,opt,name=DataplanePropagation,proto3" json:"DataplanePropagation,omitempty"`
}

func (x *SandboxCreationBreakdown) Reset() {
	*x = SandboxCreationBreakdown{}
	if protoimpl.UnsafeEnabled {
		mi := &file_worker_node_interface_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *SandboxCreationBreakdown) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SandboxCreationBreakdown) ProtoMessage() {}

func (x *SandboxCreationBreakdown) ProtoReflect() protoreflect.Message {
	mi := &file_worker_node_interface_proto_msgTypes[2]
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
	return file_worker_node_interface_proto_rawDescGZIP(), []int{2}
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

func (x *SandboxCreationBreakdown) GetSandboxCreate() *durationpb.Duration {
	if x != nil {
		return x.SandboxCreate
	}
	return nil
}

func (x *SandboxCreationBreakdown) GetNetworkSetup() *durationpb.Duration {
	if x != nil {
		return x.NetworkSetup
	}
	return nil
}

func (x *SandboxCreationBreakdown) GetSandboxStart() *durationpb.Duration {
	if x != nil {
		return x.SandboxStart
	}
	return nil
}

func (x *SandboxCreationBreakdown) GetIptables() *durationpb.Duration {
	if x != nil {
		return x.Iptables
	}
	return nil
}

func (x *SandboxCreationBreakdown) GetDataplanePropagation() *durationpb.Duration {
	if x != nil {
		return x.DataplanePropagation
	}
	return nil
}

type EndpointsList struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Endpoint []*Endpoint `protobuf:"bytes,1,rep,name=endpoint,proto3" json:"endpoint,omitempty"`
}

func (x *EndpointsList) Reset() {
	*x = EndpointsList{}
	if protoimpl.UnsafeEnabled {
		mi := &file_worker_node_interface_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *EndpointsList) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*EndpointsList) ProtoMessage() {}

func (x *EndpointsList) ProtoReflect() protoreflect.Message {
	mi := &file_worker_node_interface_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use EndpointsList.ProtoReflect.Descriptor instead.
func (*EndpointsList) Descriptor() ([]byte, []int) {
	return file_worker_node_interface_proto_rawDescGZIP(), []int{3}
}

func (x *EndpointsList) GetEndpoint() []*Endpoint {
	if x != nil {
		return x.Endpoint
	}
	return nil
}

var File_worker_node_interface_proto protoreflect.FileDescriptor

var file_worker_node_interface_proto_rawDesc = []byte{
	0x0a, 0x1b, 0x77, 0x6f, 0x72, 0x6b, 0x65, 0x72, 0x5f, 0x6e, 0x6f, 0x64, 0x65, 0x5f, 0x69, 0x6e,
	0x74, 0x65, 0x72, 0x66, 0x61, 0x63, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x0a, 0x64,
	0x61, 0x74, 0x61, 0x5f, 0x70, 0x6c, 0x61, 0x6e, 0x65, 0x1a, 0x0c, 0x63, 0x6f, 0x6d, 0x6d, 0x6f,
	0x6e, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x1b, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x65, 0x6d, 0x70, 0x74, 0x79, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x1e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x64, 0x75, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x22, 0x37, 0x0a, 0x09, 0x53, 0x61, 0x6e, 0x64, 0x62, 0x6f, 0x78, 0x49,
	0x44, 0x12, 0x0e, 0x0a, 0x02, 0x49, 0x44, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x02, 0x49,
	0x44, 0x12, 0x1a, 0x0a, 0x08, 0x48, 0x6f, 0x73, 0x74, 0x50, 0x6f, 0x72, 0x74, 0x18, 0x02, 0x20,
	0x01, 0x28, 0x05, 0x52, 0x08, 0x48, 0x6f, 0x73, 0x74, 0x50, 0x6f, 0x72, 0x74, 0x22, 0xd0, 0x01,
	0x0a, 0x15, 0x53, 0x61, 0x6e, 0x64, 0x62, 0x6f, 0x78, 0x43, 0x72, 0x65, 0x61, 0x74, 0x69, 0x6f,
	0x6e, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x12, 0x18, 0x0a, 0x07, 0x53, 0x75, 0x63, 0x63, 0x65,
	0x73, 0x73, 0x18, 0x01, 0x20, 0x01, 0x28, 0x08, 0x52, 0x07, 0x53, 0x75, 0x63, 0x63, 0x65, 0x73,
	0x73, 0x12, 0x0e, 0x0a, 0x02, 0x49, 0x44, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x02, 0x49,
	0x44, 0x12, 0x3b, 0x0a, 0x0c, 0x50, 0x6f, 0x72, 0x74, 0x4d, 0x61, 0x70, 0x70, 0x69, 0x6e, 0x67,
	0x73, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x17, 0x2e, 0x64, 0x61, 0x74, 0x61, 0x5f, 0x70,
	0x6c, 0x61, 0x6e, 0x65, 0x2e, 0x50, 0x6f, 0x72, 0x74, 0x4d, 0x61, 0x70, 0x70, 0x69, 0x6e, 0x67,
	0x52, 0x0c, 0x50, 0x6f, 0x72, 0x74, 0x4d, 0x61, 0x70, 0x70, 0x69, 0x6e, 0x67, 0x73, 0x12, 0x50,
	0x0a, 0x10, 0x4c, 0x61, 0x74, 0x65, 0x6e, 0x63, 0x79, 0x42, 0x72, 0x65, 0x61, 0x6b, 0x64, 0x6f,
	0x77, 0x6e, 0x18, 0x05, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x24, 0x2e, 0x64, 0x61, 0x74, 0x61, 0x5f,
	0x70, 0x6c, 0x61, 0x6e, 0x65, 0x2e, 0x53, 0x61, 0x6e, 0x64, 0x62, 0x6f, 0x78, 0x43, 0x72, 0x65,
	0x61, 0x74, 0x69, 0x6f, 0x6e, 0x42, 0x72, 0x65, 0x61, 0x6b, 0x64, 0x6f, 0x77, 0x6e, 0x52, 0x10,
	0x4c, 0x61, 0x74, 0x65, 0x6e, 0x63, 0x79, 0x42, 0x72, 0x65, 0x61, 0x6b, 0x64, 0x6f, 0x77, 0x6e,
	0x22, 0xcb, 0x03, 0x0a, 0x18, 0x53, 0x61, 0x6e, 0x64, 0x62, 0x6f, 0x78, 0x43, 0x72, 0x65, 0x61,
	0x74, 0x69, 0x6f, 0x6e, 0x42, 0x72, 0x65, 0x61, 0x6b, 0x64, 0x6f, 0x77, 0x6e, 0x12, 0x2f, 0x0a,
	0x05, 0x54, 0x6f, 0x74, 0x61, 0x6c, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x19, 0x2e, 0x67,
	0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x44,
	0x75, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x05, 0x54, 0x6f, 0x74, 0x61, 0x6c, 0x12, 0x39,
	0x0a, 0x0a, 0x49, 0x6d, 0x61, 0x67, 0x65, 0x46, 0x65, 0x74, 0x63, 0x68, 0x18, 0x02, 0x20, 0x01,
	0x28, 0x0b, 0x32, 0x19, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x62, 0x75, 0x66, 0x2e, 0x44, 0x75, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x0a, 0x49,
	0x6d, 0x61, 0x67, 0x65, 0x46, 0x65, 0x74, 0x63, 0x68, 0x12, 0x3f, 0x0a, 0x0d, 0x53, 0x61, 0x6e,
	0x64, 0x62, 0x6f, 0x78, 0x43, 0x72, 0x65, 0x61, 0x74, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b,
	0x32, 0x19, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62,
	0x75, 0x66, 0x2e, 0x44, 0x75, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x0d, 0x53, 0x61, 0x6e,
	0x64, 0x62, 0x6f, 0x78, 0x43, 0x72, 0x65, 0x61, 0x74, 0x65, 0x12, 0x3d, 0x0a, 0x0c, 0x4e, 0x65,
	0x74, 0x77, 0x6f, 0x72, 0x6b, 0x53, 0x65, 0x74, 0x75, 0x70, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0b,
	0x32, 0x19, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62,
	0x75, 0x66, 0x2e, 0x44, 0x75, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x0c, 0x4e, 0x65, 0x74,
	0x77, 0x6f, 0x72, 0x6b, 0x53, 0x65, 0x74, 0x75, 0x70, 0x12, 0x3d, 0x0a, 0x0c, 0x53, 0x61, 0x6e,
	0x64, 0x62, 0x6f, 0x78, 0x53, 0x74, 0x61, 0x72, 0x74, 0x18, 0x05, 0x20, 0x01, 0x28, 0x0b, 0x32,
	0x19, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75,
	0x66, 0x2e, 0x44, 0x75, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x0c, 0x53, 0x61, 0x6e, 0x64,
	0x62, 0x6f, 0x78, 0x53, 0x74, 0x61, 0x72, 0x74, 0x12, 0x35, 0x0a, 0x08, 0x49, 0x70, 0x74, 0x61,
	0x62, 0x6c, 0x65, 0x73, 0x18, 0x06, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x19, 0x2e, 0x67, 0x6f, 0x6f,
	0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x44, 0x75, 0x72,
	0x61, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x08, 0x49, 0x70, 0x74, 0x61, 0x62, 0x6c, 0x65, 0x73, 0x12,
	0x4d, 0x0a, 0x14, 0x44, 0x61, 0x74, 0x61, 0x70, 0x6c, 0x61, 0x6e, 0x65, 0x50, 0x72, 0x6f, 0x70,
	0x61, 0x67, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x18, 0x07, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x19, 0x2e,
	0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e,
	0x44, 0x75, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x14, 0x44, 0x61, 0x74, 0x61, 0x70, 0x6c,
	0x61, 0x6e, 0x65, 0x50, 0x72, 0x6f, 0x70, 0x61, 0x67, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x22, 0x41,
	0x0a, 0x0d, 0x45, 0x6e, 0x64, 0x70, 0x6f, 0x69, 0x6e, 0x74, 0x73, 0x4c, 0x69, 0x73, 0x74, 0x12,
	0x30, 0x0a, 0x08, 0x65, 0x6e, 0x64, 0x70, 0x6f, 0x69, 0x6e, 0x74, 0x18, 0x01, 0x20, 0x03, 0x28,
	0x0b, 0x32, 0x14, 0x2e, 0x64, 0x61, 0x74, 0x61, 0x5f, 0x70, 0x6c, 0x61, 0x6e, 0x65, 0x2e, 0x45,
	0x6e, 0x64, 0x70, 0x6f, 0x69, 0x6e, 0x74, 0x52, 0x08, 0x65, 0x6e, 0x64, 0x70, 0x6f, 0x69, 0x6e,
	0x74, 0x32, 0xe8, 0x01, 0x0a, 0x13, 0x57, 0x6f, 0x72, 0x6b, 0x65, 0x72, 0x4e, 0x6f, 0x64, 0x65,
	0x49, 0x6e, 0x74, 0x65, 0x72, 0x66, 0x61, 0x63, 0x65, 0x12, 0x4b, 0x0a, 0x0d, 0x43, 0x72, 0x65,
	0x61, 0x74, 0x65, 0x53, 0x61, 0x6e, 0x64, 0x62, 0x6f, 0x78, 0x12, 0x17, 0x2e, 0x64, 0x61, 0x74,
	0x61, 0x5f, 0x70, 0x6c, 0x61, 0x6e, 0x65, 0x2e, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x49,
	0x6e, 0x66, 0x6f, 0x1a, 0x21, 0x2e, 0x64, 0x61, 0x74, 0x61, 0x5f, 0x70, 0x6c, 0x61, 0x6e, 0x65,
	0x2e, 0x53, 0x61, 0x6e, 0x64, 0x62, 0x6f, 0x78, 0x43, 0x72, 0x65, 0x61, 0x74, 0x69, 0x6f, 0x6e,
	0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x12, 0x40, 0x0a, 0x0d, 0x44, 0x65, 0x6c, 0x65, 0x74, 0x65,
	0x53, 0x61, 0x6e, 0x64, 0x62, 0x6f, 0x78, 0x12, 0x15, 0x2e, 0x64, 0x61, 0x74, 0x61, 0x5f, 0x70,
	0x6c, 0x61, 0x6e, 0x65, 0x2e, 0x53, 0x61, 0x6e, 0x64, 0x62, 0x6f, 0x78, 0x49, 0x44, 0x1a, 0x18,
	0x2e, 0x64, 0x61, 0x74, 0x61, 0x5f, 0x70, 0x6c, 0x61, 0x6e, 0x65, 0x2e, 0x41, 0x63, 0x74, 0x69,
	0x6f, 0x6e, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x12, 0x42, 0x0a, 0x0d, 0x4c, 0x69, 0x73, 0x74,
	0x45, 0x6e, 0x64, 0x70, 0x6f, 0x69, 0x6e, 0x74, 0x73, 0x12, 0x16, 0x2e, 0x67, 0x6f, 0x6f, 0x67,
	0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x45, 0x6d, 0x70, 0x74,
	0x79, 0x1a, 0x19, 0x2e, 0x64, 0x61, 0x74, 0x61, 0x5f, 0x70, 0x6c, 0x61, 0x6e, 0x65, 0x2e, 0x45,
	0x6e, 0x64, 0x70, 0x6f, 0x69, 0x6e, 0x74, 0x73, 0x4c, 0x69, 0x73, 0x74, 0x42, 0x2f, 0x5a, 0x2d,
	0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x65, 0x74, 0x68, 0x2d, 0x65,
	0x61, 0x73, 0x6c, 0x2f, 0x63, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x5f, 0x6d, 0x61, 0x6e, 0x61,
	0x67, 0x65, 0x72, 0x2f, 0x61, 0x70, 0x69, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x06, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_worker_node_interface_proto_rawDescOnce sync.Once
	file_worker_node_interface_proto_rawDescData = file_worker_node_interface_proto_rawDesc
)

func file_worker_node_interface_proto_rawDescGZIP() []byte {
	file_worker_node_interface_proto_rawDescOnce.Do(func() {
		file_worker_node_interface_proto_rawDescData = protoimpl.X.CompressGZIP(file_worker_node_interface_proto_rawDescData)
	})
	return file_worker_node_interface_proto_rawDescData
}

var file_worker_node_interface_proto_msgTypes = make([]protoimpl.MessageInfo, 4)
var file_worker_node_interface_proto_goTypes = []interface{}{
	(*SandboxID)(nil),                // 0: data_plane.SandboxID
	(*SandboxCreationStatus)(nil),    // 1: data_plane.SandboxCreationStatus
	(*SandboxCreationBreakdown)(nil), // 2: data_plane.SandboxCreationBreakdown
	(*EndpointsList)(nil),            // 3: data_plane.EndpointsList
	(*PortMapping)(nil),              // 4: data_plane.PortMapping
	(*durationpb.Duration)(nil),      // 5: google.protobuf.Duration
	(*Endpoint)(nil),                 // 6: data_plane.Endpoint
	(*ServiceInfo)(nil),              // 7: data_plane.ServiceInfo
	(*emptypb.Empty)(nil),            // 8: google.protobuf.Empty
	(*ActionStatus)(nil),             // 9: data_plane.ActionStatus
}
var file_worker_node_interface_proto_depIdxs = []int32{
	4,  // 0: data_plane.SandboxCreationStatus.PortMappings:type_name -> data_plane.PortMapping
	2,  // 1: data_plane.SandboxCreationStatus.LatencyBreakdown:type_name -> data_plane.SandboxCreationBreakdown
	5,  // 2: data_plane.SandboxCreationBreakdown.Total:type_name -> google.protobuf.Duration
	5,  // 3: data_plane.SandboxCreationBreakdown.ImageFetch:type_name -> google.protobuf.Duration
	5,  // 4: data_plane.SandboxCreationBreakdown.SandboxCreate:type_name -> google.protobuf.Duration
	5,  // 5: data_plane.SandboxCreationBreakdown.NetworkSetup:type_name -> google.protobuf.Duration
	5,  // 6: data_plane.SandboxCreationBreakdown.SandboxStart:type_name -> google.protobuf.Duration
	5,  // 7: data_plane.SandboxCreationBreakdown.Iptables:type_name -> google.protobuf.Duration
	5,  // 8: data_plane.SandboxCreationBreakdown.DataplanePropagation:type_name -> google.protobuf.Duration
	6,  // 9: data_plane.EndpointsList.endpoint:type_name -> data_plane.Endpoint
	7,  // 10: data_plane.WorkerNodeInterface.CreateSandbox:input_type -> data_plane.ServiceInfo
	0,  // 11: data_plane.WorkerNodeInterface.DeleteSandbox:input_type -> data_plane.SandboxID
	8,  // 12: data_plane.WorkerNodeInterface.ListEndpoints:input_type -> google.protobuf.Empty
	1,  // 13: data_plane.WorkerNodeInterface.CreateSandbox:output_type -> data_plane.SandboxCreationStatus
	9,  // 14: data_plane.WorkerNodeInterface.DeleteSandbox:output_type -> data_plane.ActionStatus
	3,  // 15: data_plane.WorkerNodeInterface.ListEndpoints:output_type -> data_plane.EndpointsList
	13, // [13:16] is the sub-list for method output_type
	10, // [10:13] is the sub-list for method input_type
	10, // [10:10] is the sub-list for extension type_name
	10, // [10:10] is the sub-list for extension extendee
	0,  // [0:10] is the sub-list for field type_name
}

func init() { file_worker_node_interface_proto_init() }
func file_worker_node_interface_proto_init() {
	if File_worker_node_interface_proto != nil {
		return
	}
	file_common_proto_init()
	if !protoimpl.UnsafeEnabled {
		file_worker_node_interface_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
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
		file_worker_node_interface_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
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
		file_worker_node_interface_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
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
		file_worker_node_interface_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*EndpointsList); i {
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
			RawDescriptor: file_worker_node_interface_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   4,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_worker_node_interface_proto_goTypes,
		DependencyIndexes: file_worker_node_interface_proto_depIdxs,
		MessageInfos:      file_worker_node_interface_proto_msgTypes,
	}.Build()
	File_worker_node_interface_proto = out.File
	file_worker_node_interface_proto_rawDesc = nil
	file_worker_node_interface_proto_goTypes = nil
	file_worker_node_interface_proto_depIdxs = nil
}

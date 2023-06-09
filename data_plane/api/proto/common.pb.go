// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.26.0
// 	protoc        v3.17.3
// source: common.proto

package proto

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type L4Protocol int32

const (
	L4Protocol_TCP L4Protocol = 0
	L4Protocol_UDP L4Protocol = 1
)

// Enum value maps for L4Protocol.
var (
	L4Protocol_name = map[int32]string{
		0: "TCP",
		1: "UDP",
	}
	L4Protocol_value = map[string]int32{
		"TCP": 0,
		"UDP": 1,
	}
)

func (x L4Protocol) Enum() *L4Protocol {
	p := new(L4Protocol)
	*p = x
	return p
}

func (x L4Protocol) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (L4Protocol) Descriptor() protoreflect.EnumDescriptor {
	return file_common_proto_enumTypes[0].Descriptor()
}

func (L4Protocol) Type() protoreflect.EnumType {
	return &file_common_proto_enumTypes[0]
}

func (x L4Protocol) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use L4Protocol.Descriptor instead.
func (L4Protocol) EnumDescriptor() ([]byte, []int) {
	return file_common_proto_rawDescGZIP(), []int{0}
}

type PortMapping struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	HostPort  int32      `protobuf:"varint,1,opt,name=HostPort,proto3" json:"HostPort,omitempty"`
	GuestPort int32      `protobuf:"varint,2,opt,name=GuestPort,proto3" json:"GuestPort,omitempty"`
	Protocol  L4Protocol `protobuf:"varint,3,opt,name=Protocol,proto3,enum=data_plane.L4Protocol" json:"Protocol,omitempty"`
}

func (x *PortMapping) Reset() {
	*x = PortMapping{}
	if protoimpl.UnsafeEnabled {
		mi := &file_common_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *PortMapping) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*PortMapping) ProtoMessage() {}

func (x *PortMapping) ProtoReflect() protoreflect.Message {
	mi := &file_common_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use PortMapping.ProtoReflect.Descriptor instead.
func (*PortMapping) Descriptor() ([]byte, []int) {
	return file_common_proto_rawDescGZIP(), []int{0}
}

func (x *PortMapping) GetHostPort() int32 {
	if x != nil {
		return x.HostPort
	}
	return 0
}

func (x *PortMapping) GetGuestPort() int32 {
	if x != nil {
		return x.GuestPort
	}
	return 0
}

func (x *PortMapping) GetProtocol() L4Protocol {
	if x != nil {
		return x.Protocol
	}
	return L4Protocol_TCP
}

type AutoscalingConfiguration struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	ScalingUpperBound                    int32   `protobuf:"varint,1,opt,name=ScalingUpperBound,proto3" json:"ScalingUpperBound,omitempty"`
	ScalingLowerBound                    int32   `protobuf:"varint,2,opt,name=ScalingLowerBound,proto3" json:"ScalingLowerBound,omitempty"`
	PanicThresholdPercentage             float32 `protobuf:"fixed32,3,opt,name=PanicThresholdPercentage,proto3" json:"PanicThresholdPercentage,omitempty"`
	MaxScaleUpRate                       float32 `protobuf:"fixed32,4,opt,name=MaxScaleUpRate,proto3" json:"MaxScaleUpRate,omitempty"`
	MaxScaleDownRate                     float32 `protobuf:"fixed32,5,opt,name=MaxScaleDownRate,proto3" json:"MaxScaleDownRate,omitempty"`
	ContainerConcurrency                 int32   `protobuf:"varint,6,opt,name=ContainerConcurrency,proto3" json:"ContainerConcurrency,omitempty"`
	ContainerConcurrencyTargetPercentage int32   `protobuf:"varint,7,opt,name=ContainerConcurrencyTargetPercentage,proto3" json:"ContainerConcurrencyTargetPercentage,omitempty"`
	StableWindowWidthSeconds             int32   `protobuf:"varint,8,opt,name=StableWindowWidthSeconds,proto3" json:"StableWindowWidthSeconds,omitempty"`
	PanicWindowWidthSeconds              int32   `protobuf:"varint,9,opt,name=PanicWindowWidthSeconds,proto3" json:"PanicWindowWidthSeconds,omitempty"`
	ScalingPeriodSeconds                 int32   `protobuf:"varint,10,opt,name=ScalingPeriodSeconds,proto3" json:"ScalingPeriodSeconds,omitempty"`
}

func (x *AutoscalingConfiguration) Reset() {
	*x = AutoscalingConfiguration{}
	if protoimpl.UnsafeEnabled {
		mi := &file_common_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *AutoscalingConfiguration) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*AutoscalingConfiguration) ProtoMessage() {}

func (x *AutoscalingConfiguration) ProtoReflect() protoreflect.Message {
	mi := &file_common_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use AutoscalingConfiguration.ProtoReflect.Descriptor instead.
func (*AutoscalingConfiguration) Descriptor() ([]byte, []int) {
	return file_common_proto_rawDescGZIP(), []int{1}
}

func (x *AutoscalingConfiguration) GetScalingUpperBound() int32 {
	if x != nil {
		return x.ScalingUpperBound
	}
	return 0
}

func (x *AutoscalingConfiguration) GetScalingLowerBound() int32 {
	if x != nil {
		return x.ScalingLowerBound
	}
	return 0
}

func (x *AutoscalingConfiguration) GetPanicThresholdPercentage() float32 {
	if x != nil {
		return x.PanicThresholdPercentage
	}
	return 0
}

func (x *AutoscalingConfiguration) GetMaxScaleUpRate() float32 {
	if x != nil {
		return x.MaxScaleUpRate
	}
	return 0
}

func (x *AutoscalingConfiguration) GetMaxScaleDownRate() float32 {
	if x != nil {
		return x.MaxScaleDownRate
	}
	return 0
}

func (x *AutoscalingConfiguration) GetContainerConcurrency() int32 {
	if x != nil {
		return x.ContainerConcurrency
	}
	return 0
}

func (x *AutoscalingConfiguration) GetContainerConcurrencyTargetPercentage() int32 {
	if x != nil {
		return x.ContainerConcurrencyTargetPercentage
	}
	return 0
}

func (x *AutoscalingConfiguration) GetStableWindowWidthSeconds() int32 {
	if x != nil {
		return x.StableWindowWidthSeconds
	}
	return 0
}

func (x *AutoscalingConfiguration) GetPanicWindowWidthSeconds() int32 {
	if x != nil {
		return x.PanicWindowWidthSeconds
	}
	return 0
}

func (x *AutoscalingConfiguration) GetScalingPeriodSeconds() int32 {
	if x != nil {
		return x.ScalingPeriodSeconds
	}
	return 0
}

type ServiceInfo struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Name              string                    `protobuf:"bytes,1,opt,name=Name,proto3" json:"Name,omitempty"`
	Image             string                    `protobuf:"bytes,2,opt,name=Image,proto3" json:"Image,omitempty"`
	PortForwarding    *PortMapping              `protobuf:"bytes,3,opt,name=PortForwarding,proto3" json:"PortForwarding,omitempty"`
	AutoscalingConfig *AutoscalingConfiguration `protobuf:"bytes,4,opt,name=AutoscalingConfig,proto3" json:"AutoscalingConfig,omitempty"`
}

func (x *ServiceInfo) Reset() {
	*x = ServiceInfo{}
	if protoimpl.UnsafeEnabled {
		mi := &file_common_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ServiceInfo) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ServiceInfo) ProtoMessage() {}

func (x *ServiceInfo) ProtoReflect() protoreflect.Message {
	mi := &file_common_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ServiceInfo.ProtoReflect.Descriptor instead.
func (*ServiceInfo) Descriptor() ([]byte, []int) {
	return file_common_proto_rawDescGZIP(), []int{2}
}

func (x *ServiceInfo) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *ServiceInfo) GetImage() string {
	if x != nil {
		return x.Image
	}
	return ""
}

func (x *ServiceInfo) GetPortForwarding() *PortMapping {
	if x != nil {
		return x.PortForwarding
	}
	return nil
}

func (x *ServiceInfo) GetAutoscalingConfig() *AutoscalingConfiguration {
	if x != nil {
		return x.AutoscalingConfig
	}
	return nil
}

type ServiceList struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Service []string `protobuf:"bytes,1,rep,name=Service,proto3" json:"Service,omitempty"`
}

func (x *ServiceList) Reset() {
	*x = ServiceList{}
	if protoimpl.UnsafeEnabled {
		mi := &file_common_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ServiceList) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ServiceList) ProtoMessage() {}

func (x *ServiceList) ProtoReflect() protoreflect.Message {
	mi := &file_common_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ServiceList.ProtoReflect.Descriptor instead.
func (*ServiceList) Descriptor() ([]byte, []int) {
	return file_common_proto_rawDescGZIP(), []int{3}
}

func (x *ServiceList) GetService() []string {
	if x != nil {
		return x.Service
	}
	return nil
}

type ActionStatus struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Success bool   `protobuf:"varint,1,opt,name=Success,proto3" json:"Success,omitempty"`
	Message string `protobuf:"bytes,2,opt,name=Message,proto3" json:"Message,omitempty"`
}

func (x *ActionStatus) Reset() {
	*x = ActionStatus{}
	if protoimpl.UnsafeEnabled {
		mi := &file_common_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ActionStatus) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ActionStatus) ProtoMessage() {}

func (x *ActionStatus) ProtoReflect() protoreflect.Message {
	mi := &file_common_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ActionStatus.ProtoReflect.Descriptor instead.
func (*ActionStatus) Descriptor() ([]byte, []int) {
	return file_common_proto_rawDescGZIP(), []int{4}
}

func (x *ActionStatus) GetSuccess() bool {
	if x != nil {
		return x.Success
	}
	return false
}

func (x *ActionStatus) GetMessage() string {
	if x != nil {
		return x.Message
	}
	return ""
}

var File_common_proto protoreflect.FileDescriptor

var file_common_proto_rawDesc = []byte{
	0x0a, 0x0c, 0x63, 0x6f, 0x6d, 0x6d, 0x6f, 0x6e, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x0a,
	0x64, 0x61, 0x74, 0x61, 0x5f, 0x70, 0x6c, 0x61, 0x6e, 0x65, 0x22, 0x7b, 0x0a, 0x0b, 0x50, 0x6f,
	0x72, 0x74, 0x4d, 0x61, 0x70, 0x70, 0x69, 0x6e, 0x67, 0x12, 0x1a, 0x0a, 0x08, 0x48, 0x6f, 0x73,
	0x74, 0x50, 0x6f, 0x72, 0x74, 0x18, 0x01, 0x20, 0x01, 0x28, 0x05, 0x52, 0x08, 0x48, 0x6f, 0x73,
	0x74, 0x50, 0x6f, 0x72, 0x74, 0x12, 0x1c, 0x0a, 0x09, 0x47, 0x75, 0x65, 0x73, 0x74, 0x50, 0x6f,
	0x72, 0x74, 0x18, 0x02, 0x20, 0x01, 0x28, 0x05, 0x52, 0x09, 0x47, 0x75, 0x65, 0x73, 0x74, 0x50,
	0x6f, 0x72, 0x74, 0x12, 0x32, 0x0a, 0x08, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x63, 0x6f, 0x6c, 0x18,
	0x03, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x16, 0x2e, 0x64, 0x61, 0x74, 0x61, 0x5f, 0x70, 0x6c, 0x61,
	0x6e, 0x65, 0x2e, 0x4c, 0x34, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x63, 0x6f, 0x6c, 0x52, 0x08, 0x50,
	0x72, 0x6f, 0x74, 0x6f, 0x63, 0x6f, 0x6c, 0x22, 0xb8, 0x04, 0x0a, 0x18, 0x41, 0x75, 0x74, 0x6f,
	0x73, 0x63, 0x61, 0x6c, 0x69, 0x6e, 0x67, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x75, 0x72, 0x61,
	0x74, 0x69, 0x6f, 0x6e, 0x12, 0x2c, 0x0a, 0x11, 0x53, 0x63, 0x61, 0x6c, 0x69, 0x6e, 0x67, 0x55,
	0x70, 0x70, 0x65, 0x72, 0x42, 0x6f, 0x75, 0x6e, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x05, 0x52,
	0x11, 0x53, 0x63, 0x61, 0x6c, 0x69, 0x6e, 0x67, 0x55, 0x70, 0x70, 0x65, 0x72, 0x42, 0x6f, 0x75,
	0x6e, 0x64, 0x12, 0x2c, 0x0a, 0x11, 0x53, 0x63, 0x61, 0x6c, 0x69, 0x6e, 0x67, 0x4c, 0x6f, 0x77,
	0x65, 0x72, 0x42, 0x6f, 0x75, 0x6e, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x05, 0x52, 0x11, 0x53,
	0x63, 0x61, 0x6c, 0x69, 0x6e, 0x67, 0x4c, 0x6f, 0x77, 0x65, 0x72, 0x42, 0x6f, 0x75, 0x6e, 0x64,
	0x12, 0x3a, 0x0a, 0x18, 0x50, 0x61, 0x6e, 0x69, 0x63, 0x54, 0x68, 0x72, 0x65, 0x73, 0x68, 0x6f,
	0x6c, 0x64, 0x50, 0x65, 0x72, 0x63, 0x65, 0x6e, 0x74, 0x61, 0x67, 0x65, 0x18, 0x03, 0x20, 0x01,
	0x28, 0x02, 0x52, 0x18, 0x50, 0x61, 0x6e, 0x69, 0x63, 0x54, 0x68, 0x72, 0x65, 0x73, 0x68, 0x6f,
	0x6c, 0x64, 0x50, 0x65, 0x72, 0x63, 0x65, 0x6e, 0x74, 0x61, 0x67, 0x65, 0x12, 0x26, 0x0a, 0x0e,
	0x4d, 0x61, 0x78, 0x53, 0x63, 0x61, 0x6c, 0x65, 0x55, 0x70, 0x52, 0x61, 0x74, 0x65, 0x18, 0x04,
	0x20, 0x01, 0x28, 0x02, 0x52, 0x0e, 0x4d, 0x61, 0x78, 0x53, 0x63, 0x61, 0x6c, 0x65, 0x55, 0x70,
	0x52, 0x61, 0x74, 0x65, 0x12, 0x2a, 0x0a, 0x10, 0x4d, 0x61, 0x78, 0x53, 0x63, 0x61, 0x6c, 0x65,
	0x44, 0x6f, 0x77, 0x6e, 0x52, 0x61, 0x74, 0x65, 0x18, 0x05, 0x20, 0x01, 0x28, 0x02, 0x52, 0x10,
	0x4d, 0x61, 0x78, 0x53, 0x63, 0x61, 0x6c, 0x65, 0x44, 0x6f, 0x77, 0x6e, 0x52, 0x61, 0x74, 0x65,
	0x12, 0x32, 0x0a, 0x14, 0x43, 0x6f, 0x6e, 0x74, 0x61, 0x69, 0x6e, 0x65, 0x72, 0x43, 0x6f, 0x6e,
	0x63, 0x75, 0x72, 0x72, 0x65, 0x6e, 0x63, 0x79, 0x18, 0x06, 0x20, 0x01, 0x28, 0x05, 0x52, 0x14,
	0x43, 0x6f, 0x6e, 0x74, 0x61, 0x69, 0x6e, 0x65, 0x72, 0x43, 0x6f, 0x6e, 0x63, 0x75, 0x72, 0x72,
	0x65, 0x6e, 0x63, 0x79, 0x12, 0x52, 0x0a, 0x24, 0x43, 0x6f, 0x6e, 0x74, 0x61, 0x69, 0x6e, 0x65,
	0x72, 0x43, 0x6f, 0x6e, 0x63, 0x75, 0x72, 0x72, 0x65, 0x6e, 0x63, 0x79, 0x54, 0x61, 0x72, 0x67,
	0x65, 0x74, 0x50, 0x65, 0x72, 0x63, 0x65, 0x6e, 0x74, 0x61, 0x67, 0x65, 0x18, 0x07, 0x20, 0x01,
	0x28, 0x05, 0x52, 0x24, 0x43, 0x6f, 0x6e, 0x74, 0x61, 0x69, 0x6e, 0x65, 0x72, 0x43, 0x6f, 0x6e,
	0x63, 0x75, 0x72, 0x72, 0x65, 0x6e, 0x63, 0x79, 0x54, 0x61, 0x72, 0x67, 0x65, 0x74, 0x50, 0x65,
	0x72, 0x63, 0x65, 0x6e, 0x74, 0x61, 0x67, 0x65, 0x12, 0x3a, 0x0a, 0x18, 0x53, 0x74, 0x61, 0x62,
	0x6c, 0x65, 0x57, 0x69, 0x6e, 0x64, 0x6f, 0x77, 0x57, 0x69, 0x64, 0x74, 0x68, 0x53, 0x65, 0x63,
	0x6f, 0x6e, 0x64, 0x73, 0x18, 0x08, 0x20, 0x01, 0x28, 0x05, 0x52, 0x18, 0x53, 0x74, 0x61, 0x62,
	0x6c, 0x65, 0x57, 0x69, 0x6e, 0x64, 0x6f, 0x77, 0x57, 0x69, 0x64, 0x74, 0x68, 0x53, 0x65, 0x63,
	0x6f, 0x6e, 0x64, 0x73, 0x12, 0x38, 0x0a, 0x17, 0x50, 0x61, 0x6e, 0x69, 0x63, 0x57, 0x69, 0x6e,
	0x64, 0x6f, 0x77, 0x57, 0x69, 0x64, 0x74, 0x68, 0x53, 0x65, 0x63, 0x6f, 0x6e, 0x64, 0x73, 0x18,
	0x09, 0x20, 0x01, 0x28, 0x05, 0x52, 0x17, 0x50, 0x61, 0x6e, 0x69, 0x63, 0x57, 0x69, 0x6e, 0x64,
	0x6f, 0x77, 0x57, 0x69, 0x64, 0x74, 0x68, 0x53, 0x65, 0x63, 0x6f, 0x6e, 0x64, 0x73, 0x12, 0x32,
	0x0a, 0x14, 0x53, 0x63, 0x61, 0x6c, 0x69, 0x6e, 0x67, 0x50, 0x65, 0x72, 0x69, 0x6f, 0x64, 0x53,
	0x65, 0x63, 0x6f, 0x6e, 0x64, 0x73, 0x18, 0x0a, 0x20, 0x01, 0x28, 0x05, 0x52, 0x14, 0x53, 0x63,
	0x61, 0x6c, 0x69, 0x6e, 0x67, 0x50, 0x65, 0x72, 0x69, 0x6f, 0x64, 0x53, 0x65, 0x63, 0x6f, 0x6e,
	0x64, 0x73, 0x22, 0xcc, 0x01, 0x0a, 0x0b, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x49, 0x6e,
	0x66, 0x6f, 0x12, 0x12, 0x0a, 0x04, 0x4e, 0x61, 0x6d, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x04, 0x4e, 0x61, 0x6d, 0x65, 0x12, 0x14, 0x0a, 0x05, 0x49, 0x6d, 0x61, 0x67, 0x65, 0x18,
	0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x49, 0x6d, 0x61, 0x67, 0x65, 0x12, 0x3f, 0x0a, 0x0e,
	0x50, 0x6f, 0x72, 0x74, 0x46, 0x6f, 0x72, 0x77, 0x61, 0x72, 0x64, 0x69, 0x6e, 0x67, 0x18, 0x03,
	0x20, 0x01, 0x28, 0x0b, 0x32, 0x17, 0x2e, 0x64, 0x61, 0x74, 0x61, 0x5f, 0x70, 0x6c, 0x61, 0x6e,
	0x65, 0x2e, 0x50, 0x6f, 0x72, 0x74, 0x4d, 0x61, 0x70, 0x70, 0x69, 0x6e, 0x67, 0x52, 0x0e, 0x50,
	0x6f, 0x72, 0x74, 0x46, 0x6f, 0x72, 0x77, 0x61, 0x72, 0x64, 0x69, 0x6e, 0x67, 0x12, 0x52, 0x0a,
	0x11, 0x41, 0x75, 0x74, 0x6f, 0x73, 0x63, 0x61, 0x6c, 0x69, 0x6e, 0x67, 0x43, 0x6f, 0x6e, 0x66,
	0x69, 0x67, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x24, 0x2e, 0x64, 0x61, 0x74, 0x61, 0x5f,
	0x70, 0x6c, 0x61, 0x6e, 0x65, 0x2e, 0x41, 0x75, 0x74, 0x6f, 0x73, 0x63, 0x61, 0x6c, 0x69, 0x6e,
	0x67, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x75, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x11,
	0x41, 0x75, 0x74, 0x6f, 0x73, 0x63, 0x61, 0x6c, 0x69, 0x6e, 0x67, 0x43, 0x6f, 0x6e, 0x66, 0x69,
	0x67, 0x22, 0x27, 0x0a, 0x0b, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x4c, 0x69, 0x73, 0x74,
	0x12, 0x18, 0x0a, 0x07, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x18, 0x01, 0x20, 0x03, 0x28,
	0x09, 0x52, 0x07, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x22, 0x42, 0x0a, 0x0c, 0x41, 0x63,
	0x74, 0x69, 0x6f, 0x6e, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x12, 0x18, 0x0a, 0x07, 0x53, 0x75,
	0x63, 0x63, 0x65, 0x73, 0x73, 0x18, 0x01, 0x20, 0x01, 0x28, 0x08, 0x52, 0x07, 0x53, 0x75, 0x63,
	0x63, 0x65, 0x73, 0x73, 0x12, 0x18, 0x0a, 0x07, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x18,
	0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x2a, 0x1e,
	0x0a, 0x0a, 0x4c, 0x34, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x63, 0x6f, 0x6c, 0x12, 0x07, 0x0a, 0x03,
	0x54, 0x43, 0x50, 0x10, 0x00, 0x12, 0x07, 0x0a, 0x03, 0x55, 0x44, 0x50, 0x10, 0x01, 0x42, 0x3a,
	0x5a, 0x38, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x65, 0x74, 0x68,
	0x2d, 0x65, 0x61, 0x73, 0x6c, 0x2f, 0x63, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x5f, 0x6d, 0x61,
	0x6e, 0x61, 0x67, 0x65, 0x72, 0x2f, 0x64, 0x61, 0x74, 0x61, 0x5f, 0x70, 0x6c, 0x61, 0x6e, 0x65,
	0x2f, 0x61, 0x70, 0x69, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x33,
}

var (
	file_common_proto_rawDescOnce sync.Once
	file_common_proto_rawDescData = file_common_proto_rawDesc
)

func file_common_proto_rawDescGZIP() []byte {
	file_common_proto_rawDescOnce.Do(func() {
		file_common_proto_rawDescData = protoimpl.X.CompressGZIP(file_common_proto_rawDescData)
	})
	return file_common_proto_rawDescData
}

var file_common_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_common_proto_msgTypes = make([]protoimpl.MessageInfo, 5)
var file_common_proto_goTypes = []interface{}{
	(L4Protocol)(0),                  // 0: data_plane.L4Protocol
	(*PortMapping)(nil),              // 1: data_plane.PortMapping
	(*AutoscalingConfiguration)(nil), // 2: data_plane.AutoscalingConfiguration
	(*ServiceInfo)(nil),              // 3: data_plane.ServiceInfo
	(*ServiceList)(nil),              // 4: data_plane.ServiceList
	(*ActionStatus)(nil),             // 5: data_plane.ActionStatus
}
var file_common_proto_depIdxs = []int32{
	0, // 0: data_plane.PortMapping.Protocol:type_name -> data_plane.L4Protocol
	1, // 1: data_plane.ServiceInfo.PortForwarding:type_name -> data_plane.PortMapping
	2, // 2: data_plane.ServiceInfo.AutoscalingConfig:type_name -> data_plane.AutoscalingConfiguration
	3, // [3:3] is the sub-list for method output_type
	3, // [3:3] is the sub-list for method input_type
	3, // [3:3] is the sub-list for extension type_name
	3, // [3:3] is the sub-list for extension extendee
	0, // [0:3] is the sub-list for field type_name
}

func init() { file_common_proto_init() }
func file_common_proto_init() {
	if File_common_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_common_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*PortMapping); i {
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
		file_common_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*AutoscalingConfiguration); i {
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
		file_common_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ServiceInfo); i {
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
		file_common_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ServiceList); i {
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
		file_common_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ActionStatus); i {
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
			RawDescriptor: file_common_proto_rawDesc,
			NumEnums:      1,
			NumMessages:   5,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_common_proto_goTypes,
		DependencyIndexes: file_common_proto_depIdxs,
		EnumInfos:         file_common_proto_enumTypes,
		MessageInfos:      file_common_proto_msgTypes,
	}.Build()
	File_common_proto = out.File
	file_common_proto_rawDesc = nil
	file_common_proto_goTypes = nil
	file_common_proto_depIdxs = nil
}
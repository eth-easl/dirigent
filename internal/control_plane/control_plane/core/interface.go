package core

import (
	"cluster_manager/pkg/synchronization"
	"cluster_manager/proto"
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type DataPlaneInterface interface {
	InitializeDataPlaneConnection(host string, port string) error
	AddDeployment(context.Context, *proto.ServiceInfo) (*proto.DeploymentUpdateSuccess, error)
	AddWorkflowDeployment(context.Context, *proto.WorkflowInfo) (*proto.DeploymentUpdateSuccess, error)
	UpdateDeployment(context.Context, *proto.ServiceInfo) (*proto.DeploymentUpdateSuccess, error)
	UpdateEndpointList(context.Context, *proto.DeploymentEndpointPatch) (*proto.DeploymentUpdateSuccess, error)
	DeleteDeployment(context.Context, *proto.ServiceInfo) (*proto.DeploymentUpdateSuccess, error)
	DeleteWorkflowDeployment(context.Context, *proto.WorkflowObjectIdentifier) (*proto.DeploymentUpdateSuccess, error)
	ReceiveRouteUpdate(context.Context, *proto.RouteUpdate, ...grpc.CallOption) (*proto.ActionStatus, error)
	DrainSandbox(context.Context, *proto.DeploymentEndpointPatch) (*proto.DeploymentUpdateSuccess, error)
	ResetMeasurements(context.Context, *emptypb.Empty) (*proto.ActionStatus, error)
	GetLastHeartBeat() time.Time
	UpdateHeartBeat()
	GetIP() string
	GetApiPort() string
	GetProxyPort() string
}

type WorkerNodeInterface interface {
	ConnectToWorker() proto.WorkerNodeInterfaceClient
	CreateSandbox(context.Context, *proto.ServiceInfo, ...grpc.CallOption) (*proto.SandboxCreationStatus, error)
	DeleteSandbox(context.Context, *proto.SandboxID, ...grpc.CallOption) (*proto.ActionStatus, error)
	CreateTaskSandbox(context.Context, *proto.WorkflowTaskInfo, ...grpc.CallOption) (*proto.SandboxCreationStatus, error)
	ListEndpoints(context.Context, *emptypb.Empty, ...grpc.CallOption) (*proto.EndpointsList, error)
	PrepullImage(context.Context, *proto.ImageInfo, ...grpc.CallOption) (*proto.ActionStatus, error)
	ReceiveRouteUpdate(context.Context, *proto.RouteUpdate, ...grpc.CallOption) (*proto.ActionStatus, error)
	GetName() string
	GetLastHeartBeat() time.Time
	GetWorkerNodeConfiguration() WorkerNodeConfiguration
	UpdateLastHearBeat()
	SetCpuUsed(uint64)
	SetMemoryUsed(uint64)
	AddUsage(cpu, memory uint64)
	GetCpuAvailable() uint64
	GetCpuUsed() uint64
	GetMemoryAvailable() uint64
	GetMemoryUsed() uint64
	GetIP() string
	GetPort() string
	GetCIDR() string
	GetEndpointMap() synchronization.SyncStructure[*Endpoint, string]
	SetSchedulability(bool)
	GetSchedulability() bool
	AddImage(string) bool
	RemoveImage(string) bool
	HasImage(string) bool
}

type AutoscalingInterface interface {
	PanicPoke(functionName string, previousValue int32)
	Poke(functionName string, previousValue int32)
	Stop(functionName string)
}

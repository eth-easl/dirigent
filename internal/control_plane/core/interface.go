package core

import (
	"cluster_manager/api/proto"
	"context"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"time"
)

type DataPlaneInterface interface {
	InitializeDataPlaneConnection(host string, port string) error
	AddDeployment(context.Context, *proto.ServiceInfo) (*proto.DeploymentUpdateSuccess, error)
	UpdateEndpointList(context.Context, *proto.DeploymentEndpointPatch) (*proto.DeploymentUpdateSuccess, error)
	DeleteDeployment(context.Context, *proto.ServiceInfo) (*proto.DeploymentUpdateSuccess, error)
	ResetMeasurements(context.Context, *emptypb.Empty) (*proto.ActionStatus, error)
	GetLastHeartBeat() time.Time
	UpdateHeartBeat()
	GetIP() string
	GetApiPort() string
	GetProxyPort() string
}

type WorkerNodeInterface interface {
	GetAPI() proto.WorkerNodeInterfaceClient
	CreateSandbox(context.Context, *proto.ServiceInfo, ...grpc.CallOption) (*proto.SandboxCreationStatus, error)
	DeleteSandbox(context.Context, *proto.SandboxID, ...grpc.CallOption) (*proto.ActionStatus, error)
	ListEndpoints(context.Context, *emptypb.Empty, ...grpc.CallOption) (*proto.EndpointsList, error)
	GetName() string
	GetLastHeartBeat() time.Time
	GetWorkerNodeConfiguration() WorkerNodeConfiguration
	UpdateLastHearBeat()
	SetCpuUsage(uint64)
	SetMemoryUsage(uint64)
	GetMemory() uint64
	GetCpuCores() uint64
	GetCpuUsage() uint64
	GetMemoryUsage() uint64
	GetIP() string
	GetPort() string
}

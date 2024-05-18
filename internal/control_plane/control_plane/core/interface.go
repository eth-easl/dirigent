package core

import (
	"cluster_manager/pkg/synchronization"
	"cluster_manager/proto"
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
	GetEndpointMap() synchronization.SyncStructure[*Endpoint, string]
	SetSchedulability(bool)
	GetSchedulability() bool
}

type AutoscalingInterface interface {
	Poke(functionName string)
	// TODO: I should set this in per functions state -> this goes in per function state
	// ForwardDataplaneMetrics(*proto.MetricsPredictiveAutoscaler)
	// TODO: Same for this function
	// ForwardRPSMetrics()
	Stop(functionName string)
}

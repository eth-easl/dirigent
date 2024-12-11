package persistence

import (
	"cluster_manager/proto"
	"context"
)

type PersistenceLayer interface {
	StoreDataPlaneInformation(ctx context.Context, dataplaneInfo *proto.DataplaneInformation) error
	DeleteDataPlaneInformation(ctx context.Context, dataplaneInfo *proto.DataplaneInformation) error
	GetDataPlaneInformation(ctx context.Context) ([]*proto.DataplaneInformation, error)
	StoreWorkerNodeInformation(ctx context.Context, workerNodeInfo *proto.NodeInfo) error
	DeleteWorkerNodeInformation(ctx context.Context, name string) error
	GetWorkerNodeInformation(ctx context.Context) ([]*proto.NodeInfo, error)
	StoreServiceInformation(ctx context.Context, serviceInfo *proto.ServiceInfo) error
	DeleteServiceInformation(ctx context.Context, serviceInfo *proto.ServiceInfo) error
	GetServiceInformation(ctx context.Context) ([]*proto.ServiceInfo, error)
	SetLeader(ctx context.Context) error
	StoreWorkflowTaskInformation(ctx context.Context, wfTaskInfo *proto.WorkflowTaskInfo) error
	DeleteWorkflowTaskInformation(ctx context.Context, name string) error
	GetWorkflowTaskInformation(ctx context.Context) ([]*proto.WorkflowTaskInfo, error)
	StoreWorkflowInformation(ctx context.Context, wfTaskInfo *proto.WorkflowInfo) error
	DeleteWorkflowInformation(ctx context.Context, name string) error
	GetWorkflowInformation(ctx context.Context) ([]*proto.WorkflowInfo, error)
}

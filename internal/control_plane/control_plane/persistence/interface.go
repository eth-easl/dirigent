package persistence

import (
	"cluster_manager/internal/control_plane/control_plane/core"
	"cluster_manager/proto"
	"context"
)

type PersistenceLayer interface {
	StoreDataPlaneInformation(ctx context.Context, dataplaneInfo *proto.DataplaneInformation) error
	DeleteDataPlaneInformation(ctx context.Context, dataplaneInfo *proto.DataplaneInformation) error
	GetDataPlaneInformation(ctx context.Context) ([]*proto.DataplaneInformation, error)
	StoreWorkerNodeInformation(ctx context.Context, workerNodeInfo *proto.WorkerNodeInformation) error
	DeleteWorkerNodeInformation(ctx context.Context, name string) error
	GetWorkerNodeInformation(ctx context.Context) ([]*proto.WorkerNodeInformation, error)
	StoreServiceInformation(ctx context.Context, serviceInfo *proto.ServiceInfo) error
	DeleteServiceInformation(ctx context.Context, serviceInfo *proto.ServiceInfo) error
	GetServiceInformation(ctx context.Context) ([]*proto.ServiceInfo, error)
	SetLeader(ctx context.Context) error
	StoreEndpoint(ctx context.Context, endpoint core.Endpoint) error
}

package persistence

import (
	"cluster_manager/api/proto"
	"context"
	"time"
)

type PersistenceLayer interface {
	StoreDataPlaneInformation(ctx context.Context, dataplaneInfo *proto.DataplaneInformation, timestamp time.Time) error
	DeleteDataPlaneInformation(ctx context.Context, dataplaneInfo *proto.DataplaneInformation, timestamp time.Time) error
	GetDataPlaneInformation(ctx context.Context) ([]*proto.DataplaneInformation, error)
	StoreWorkerNodeInformation(ctx context.Context, workerNodeInfo *proto.WorkerNodeInformation, timestamp time.Time) error
	DeleteWorkerNodeInformation(ctx context.Context, name string, timestamp time.Time) error
	GetWorkerNodeInformation(ctx context.Context) ([]*proto.WorkerNodeInformation, error)
	StoreServiceInformation(ctx context.Context, serviceInfo *proto.ServiceInfo, timestamp time.Time) error
	DeleteServiceInformation(ctx context.Context, serviceInfo *proto.ServiceInfo, timestamp time.Time) error
	GetServiceInformation(ctx context.Context) ([]*proto.ServiceInfo, error)
}

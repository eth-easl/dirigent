package persistence

import (
	"cluster_manager/api/proto"
	"context"
	"time"
)

func NewEmptyPeristenceLayer() *EmptyPersistence {
	return &EmptyPersistence{}
}

type EmptyPersistence struct{}

func (e *EmptyPersistence) StoreDataPlaneInformation(ctx context.Context, dataplaneInfo *proto.DataplaneInformation, timestamp time.Time) error {
	return nil
}

func (e *EmptyPersistence) DeleteDataPlaneInformation(ctx context.Context, dataplaneInfo *proto.DataplaneInformation, timestamp time.Time) error {
	return nil
}

func (e *EmptyPersistence) GetDataPlaneInformation(ctx context.Context) ([]*proto.DataplaneInformation, error) {
	return make([]*proto.DataplaneInformation, 0), nil
}

func (e *EmptyPersistence) StoreWorkerNodeInformation(ctx context.Context, workerNodeInfo *proto.WorkerNodeInformation, timestamp time.Time) error {
	return nil
}

func (e *EmptyPersistence) DeleteWorkerNodeInformation(ctx context.Context, name string, timestamp time.Time) error {
	return nil
}

func (e *EmptyPersistence) GetWorkerNodeInformation(ctx context.Context) ([]*proto.WorkerNodeInformation, error) {
	return make([]*proto.WorkerNodeInformation, 0), nil
}

func (e *EmptyPersistence) StoreServiceInformation(ctx context.Context, serviceInfo *proto.ServiceInfo, timestamp time.Time) error {
	return nil
}

func (e *EmptyPersistence) DeleteServiceInformation(ctx context.Context, serviceInfo *proto.ServiceInfo, timestamp time.Time) error {
	return nil
}

func (e *EmptyPersistence) GetServiceInformation(ctx context.Context) ([]*proto.ServiceInfo, error) {
	return make([]*proto.ServiceInfo, 0), nil
}

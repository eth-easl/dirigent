package persistence

import (
	"cluster_manager/api/proto"
	"context"
)

func NewEmptyPeristenceLayer() *EmptyPersistence {
	return &EmptyPersistence{}
}

type EmptyPersistence struct{}

func (e *EmptyPersistence) StoreDataPlaneInformation(_ context.Context, _ *proto.DataplaneInformation) error {
	return nil
}

func (e *EmptyPersistence) DeleteDataPlaneInformation(_ context.Context, _ *proto.DataplaneInformation) error {
	return nil
}

func (e *EmptyPersistence) GetDataPlaneInformation(_ context.Context) ([]*proto.DataplaneInformation, error) {
	return make([]*proto.DataplaneInformation, 0), nil
}

func (e *EmptyPersistence) StoreWorkerNodeInformation(_ context.Context, _ *proto.WorkerNodeInformation) error {
	return nil
}

func (e *EmptyPersistence) DeleteWorkerNodeInformation(_ context.Context, _ string) error {
	return nil
}

func (e *EmptyPersistence) GetWorkerNodeInformation(_ context.Context) ([]*proto.WorkerNodeInformation, error) {
	return make([]*proto.WorkerNodeInformation, 0), nil
}

func (e *EmptyPersistence) StoreServiceInformation(_ context.Context, _ *proto.ServiceInfo) error {
	return nil
}

func (e *EmptyPersistence) DeleteServiceInformation(_ context.Context, _ *proto.ServiceInfo) error {
	return nil
}

func (e *EmptyPersistence) GetServiceInformation(_ context.Context) ([]*proto.ServiceInfo, error) {
	return make([]*proto.ServiceInfo, 0), nil
}

func (e *EmptyPersistence) SetLeader(_ context.Context) error {
	return nil
}

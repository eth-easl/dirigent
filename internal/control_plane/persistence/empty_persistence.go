package persistence

import (
	"cluster_manager/api/proto"
	"context"
)

func NewEmptyPeristenceLayer() *emptyPersistence {
	return &emptyPersistence{}
}

type emptyPersistence struct{}

func (emptyPersistence) StoreDataPlaneInformation(ctx context.Context, dataplaneInfo *proto.DataplaneInformation) error {
	return nil
}

func (emptyPersistence) DeleteDataPlaneInformation(ctx context.Context, dataplaneInfo *proto.DataplaneInformation) error {
	return nil
}

func (emptyPersistence) GetDataPlaneInformation(ctx context.Context) ([]*proto.DataplaneInformation, error) {
	return make([]*proto.DataplaneInformation, 0), nil
}

func (emptyPersistence) StoreWorkerNodeInformation(ctx context.Context, workerNodeInfo *proto.WorkerNodeInformation) error {
	return nil
}

func (emptyPersistence) DeleteWorkerNodeInformation(ctx context.Context, workerNodeInfo *proto.WorkerNodeInformation) error {
	return nil
}

func (emptyPersistence) GetWorkerNodeInformation(ctx context.Context) ([]*proto.WorkerNodeInformation, error) {
	return make([]*proto.WorkerNodeInformation, 0), nil
}

func (emptyPersistence) StoreServiceInformation(ctx context.Context, serviceInfo *proto.ServiceInfo) error {
	return nil
}

func (emptyPersistence) GetServiceInformation(ctx context.Context) ([]*proto.ServiceInfo, error) {
	return make([]*proto.ServiceInfo, 0), nil
}

func (emptyPersistence) UpdateEndpoints(ctx context.Context, serviceName string, endpoints []*proto.Endpoint) error {
	return nil
}

func (emptyPersistence) DeleteEndpoint(ctx context.Context, serviceName string, workerNodeName string) error {
	return nil
}

func (emptyPersistence) GetEndpoints(ctx context.Context) ([]*proto.Endpoint, []string, error) {
	return make([]*proto.Endpoint, 0), make([]string, 0), nil
}

func (emptyPersistence) StoreSerialized(ctx context.Context, controlPlane []byte) error {
	return nil
}

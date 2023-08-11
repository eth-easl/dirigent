package persistence

import (
	"cluster_manager/api/proto"
	"context"
	"time"
)

type PersistenceLayer interface {
	StoreDataPlaneInformation(ctx context.Context, dataplaneInfo *proto.DataplaneInformation) error
	DeleteDataPlaneInformation(ctx context.Context, dataplaneInfo *proto.DataplaneInformation) error
	GetDataPlaneInformation(ctx context.Context) ([]*proto.DataplaneInformation, error)
	StoreWorkerNodeInformation(ctx context.Context, workerNodeInfo *proto.WorkerNodeInformation) error
	DeleteWorkerNodeInformation(ctx context.Context, workerNodeInfo *proto.WorkerNodeInformation) error
	GetWorkerNodeInformation(ctx context.Context) ([]*proto.WorkerNodeInformation, error)
	StoreServiceInformation(ctx context.Context, serviceInfo *proto.ServiceInfo) error
	StoreServiceInformationProto(ctx context.Context, serviceInfo *proto.ServiceInfo) error
	GetServiceInformation(ctx context.Context) ([]*proto.ServiceInfo, error)
	UpdateEndpoints(ctx context.Context, serviceName string, endpoints []*proto.Endpoint) (map[*proto.Endpoint]time.Duration, error)
	DeleteEndpoint(ctx context.Context, serviceName string, workerNodeName string) error
	GetEndpoints(ctx context.Context) ([]*proto.Endpoint, []string, error)
	StoreSerialized(ctx context.Context, controlPlane []byte) error
}

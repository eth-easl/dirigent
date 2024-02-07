package request_persistence

import (
	"cluster_manager/internal/data_plane/proxy/requests"
	"context"
	"time"
)

type emptyRequestPersistence struct {
}

func CreateEmptyRequestPersistence() RequestPersistence {
	return &emptyRequestPersistence{}
}

func (empty *emptyRequestPersistence) PersistBufferedRequest(ctx context.Context, request *requests.BufferedRequest) (error, time.Duration, time.Duration) {
	return nil, 0, 0
}

func (empty *emptyRequestPersistence) PersistBufferedResponse(ctx context.Context, response *requests.BufferedResponse) error {
	return nil
}

func (empty *emptyRequestPersistence) ScanBufferedRequests(ctx context.Context) ([]*requests.BufferedRequest, error) {
	return make([]*requests.BufferedRequest, 0), nil
}

func (empty *emptyRequestPersistence) ScanBufferedResponses(ctx context.Context) ([]*requests.BufferedResponse, error) {
	return make([]*requests.BufferedResponse, 0), nil
}

func (empty *emptyRequestPersistence) DeleteBufferedRequest(ctx context.Context, code string) error {
	return nil
}

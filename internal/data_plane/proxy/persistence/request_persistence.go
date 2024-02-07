package request_persistence

import (
	"cluster_manager/internal/data_plane/proxy/requests"
	"context"
	"time"
)

type RequestPersistence interface {
	PersistBufferedRequest(ctx context.Context, request *requests.BufferedRequest) (error, time.Duration, time.Duration)
	PersistBufferedResponse(ctx context.Context, response *requests.BufferedResponse) error
	ScanBufferedRequests(ctx context.Context) ([]*requests.BufferedRequest, error)
	ScanBufferedResponses(ctx context.Context) ([]*requests.BufferedResponse, error)
	DeleteBufferedRequest(ctx context.Context, code string) error
}

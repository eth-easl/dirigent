package request_persistence

import (
	"cluster_manager/internal/data_plane/proxy/requests"
	"cluster_manager/pkg/config"
	"context"
	"encoding/json"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

type RequestPersistence interface {
	PersistBufferedRequest(ctx context.Context, request *requests.BufferedRequest) error
	PersistBufferedResponse(ctx context.Context, response *requests.BufferedResponse) error
}

type emptyRequestPersistence struct {
}

func CreateEmptyRequestPersistence() RequestPersistence {
	return &emptyRequestPersistence{}
}

func (empty *emptyRequestPersistence) PersistBufferedRequest(ctx context.Context, request *requests.BufferedRequest) error {
	return nil
}

func (empty *emptyRequestPersistence) PersistBufferedResponse(ctx context.Context, response *requests.BufferedResponse) error {
	return nil
}

type requestRedisClient struct {
	RedisClient *redis.Client
}

func CreateRequestRedisClient(ctx context.Context, redisLogin config.RedisConf) (RequestPersistence, error) {

	redisClient := redis.NewClient(&redis.Options{
		Addr:     redisLogin.Address,
		Password: redisLogin.Password,
		DB:       redisLogin.Db,
	})

	if redisLogin.FullPersistence {
		logrus.Warn("Modifications")
		if err := redisClient.ConfigSet(ctx, "appendonly", "yes").Err(); err != nil {
			return &requestRedisClient{}, err
		}

		if err := redisClient.ConfigSet(ctx, "appendfsync", "always").Err(); err != nil {
			return &requestRedisClient{}, err
		}
	} else {
		if err := redisClient.ConfigSet(ctx, "appendonly", "no").Err(); err != nil {
			return &requestRedisClient{}, err
		}

		if err := redisClient.ConfigSet(ctx, "appendfsync", "everysec").Err(); err != nil {
			return &requestRedisClient{}, err
		}
	}

	return &requestRedisClient{
		RedisClient: redisClient,
	}, redisClient.Ping(ctx).Err()
}

func (driver *requestRedisClient) PersistBufferedRequest(ctx context.Context, bufferedRequest *requests.BufferedRequest) error {
	data, err := json.Marshal(bufferedRequest)
	if err != nil {
		return err
	}

	return driver.RedisClient.HSet(ctx, bufferedRequest.Code, "data", data).Err()
}

func (driver *requestRedisClient) PersistBufferedResponse(ctx context.Context, bufferedResponse *requests.BufferedResponse) error {
	data, err := json.Marshal(bufferedResponse)
	if err != nil {
		return err
	}

	return driver.RedisClient.HSet(ctx, bufferedResponse.Code, "data", data).Err()
}

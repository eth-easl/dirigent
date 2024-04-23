package request_persistence

import (
	"cluster_manager/internal/data_plane/proxy/requests"
	"cluster_manager/pkg/config"
	"cluster_manager/pkg/redis_helpers"
	"context"
	"encoding/json"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"time"
)

const (
	bufferedRequestPrefix  string = "req_"
	bufferedResponsePrefix string = "res_"
)

type requestRedisClient struct {
	RedisClient *redis.Client
}

func CreateRequestRedisClient(ctx context.Context, redisLogin config.RedisConf) (RequestPersistence, error) {
	redisClient, _, err := redis_helpers.CreateRedisConnector(ctx, redisLogin)

	return &requestRedisClient{RedisClient: redisClient}, err
}

func (driver *requestRedisClient) PersistBufferedRequest(ctx context.Context, bufferedRequest *requests.BufferedRequest) (error, time.Duration, time.Duration) {
	startSerialization := time.Now()
	data, err := json.Marshal(bufferedRequest)
	if err != nil {
		return err, 0, 0
	}
	durationSerialization := time.Since(startSerialization)

	startPersistence := time.Now()
	if err = driver.RedisClient.HSet(ctx, bufferedRequestPrefix+bufferedRequest.Code, "data", data).Err(); err != nil {
		return err, 0, 0
	}
	durationPersistence := time.Since(startPersistence)

	return nil, durationSerialization, durationPersistence
}

func (driver *requestRedisClient) PersistBufferedResponse(ctx context.Context, bufferedResponse *requests.BufferedResponse) error {
	data, err := json.Marshal(bufferedResponse)
	if err != nil {
		return err
	}

	return driver.RedisClient.HSet(ctx, bufferedResponsePrefix+bufferedResponse.UniqueCodeIdentifier, "data", data).Err()
}

// TODO: Deduplicate code
func (driver *requestRedisClient) ScanBufferedRequests(ctx context.Context) ([]*requests.BufferedRequest, error) {
	logrus.Trace("get buffered requests from the database")

	keys, err := redis_helpers.ScanKeys(ctx, driver.RedisClient, bufferedRequestPrefix)
	if err != nil {
		return nil, err
	}

	req := make([]*requests.BufferedRequest, 0)

	for _, key := range keys {
		fields, err := driver.RedisClient.HGetAll(ctx, key).Result()
		if err != nil {
			return nil, err
		}

		request := &requests.BufferedRequest{}

		err = json.Unmarshal([]byte(fields["data"]), request)

		if err != nil {
			panic(err)
		}

		req = append(req, request)
	}

	logrus.Tracef("Found %d buffered request(s) in the database", len(req))

	return req, nil
}

// TODO: Deduplicate code
func (driver *requestRedisClient) ScanBufferedResponses(ctx context.Context) ([]*requests.BufferedResponse, error) {
	logrus.Trace("get buffered responses from the database")

	keys, err := redis_helpers.ScanKeys(ctx, driver.RedisClient, bufferedResponsePrefix)
	if err != nil {
		return nil, err
	}

	responses := make([]*requests.BufferedResponse, 0)

	for _, key := range keys {
		fields, err := driver.RedisClient.HGetAll(ctx, key).Result()
		if err != nil {
			return nil, err
		}

		response := &requests.BufferedResponse{}

		err = json.Unmarshal([]byte(fields["data"]), response)

		if err != nil {
			panic(err)
		}

		responses = append(responses, response)
	}

	logrus.Tracef("Found %d buffered response(s) in the database", len(responses))

	return responses, nil
}

func (driver *requestRedisClient) DeleteBufferedRequest(ctx context.Context, code string) error {
	logrus.Tracef("Delete buffered request with code %s", code)
	return driver.RedisClient.Del(ctx, bufferedRequestPrefix+code, "data").Err()
}

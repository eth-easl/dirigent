package request_persistence

import (
	"cluster_manager/internal/data_plane/proxy/requests"
	"cluster_manager/pkg/config"
	"context"
	"encoding/json"
	"fmt"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"time"
)

type RequestPersistence interface {
	PersistBufferedRequest(ctx context.Context, request *requests.BufferedRequest) (error, time.Duration, time.Duration)
	PersistBufferedResponse(ctx context.Context, response *requests.BufferedResponse) error
	ScanBufferedRequests(ctx context.Context) ([]*requests.BufferedRequest, error)
	ScanBufferedResponses(ctx context.Context) ([]*requests.BufferedResponse, error)
	DeleteBufferedRequest(ctx context.Context, code string) error
}

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

const (
	bufferedRequestPrefix  string = "req_"
	bufferedResponsePredix string = "res_"
)

type requestRedisClient struct {
	RedisClient *redis.Client
}

// TODO: Deduplicate the code
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

	return driver.RedisClient.HSet(ctx, bufferedResponsePredix+bufferedResponse.Code, "data", data).Err()
}

// TODO: Deduplicate code
func (driver *requestRedisClient) ScanBufferedRequests(ctx context.Context) ([]*requests.BufferedRequest, error) {
	logrus.Trace("get buffered requests from the database")

	keys, err := driver.scanKeys(ctx, bufferedRequestPrefix)
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

	keys, err := driver.scanKeys(ctx, bufferedResponsePredix)
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

// TODO: Deduplicate code
func (driver *requestRedisClient) scanKeys(ctx context.Context, prefix string) ([]string, error) {
	var (
		cursor uint64
		n      int
	)

	output := make([]string, 0)

	for {
		var (
			keys []string
			err  error
		)

		keys, cursor, err = driver.RedisClient.Scan(ctx, cursor, fmt.Sprintf("%s*", prefix), 10).Result()
		if err != nil {
			panic(err)
		}

		n += len(keys)

		output = append(output, keys...)

		if cursor == 0 {
			break
		}
	}

	return output, nil
}

func (driver *requestRedisClient) DeleteBufferedRequest(ctx context.Context, code string) error {
	logrus.Tracef("delete buffered request with code %s", code)
	return driver.RedisClient.Del(ctx, bufferedRequestPrefix+code, "data").Err()
}

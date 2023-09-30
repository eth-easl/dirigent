package persistence

import (
	"cluster_manager/api/proto"
	"cluster_manager/pkg/config"
	"context"
	"fmt"
	proto2 "github.com/golang/protobuf/proto"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

const (
	dataplanePrefix string = "dataplane"
	workerPrefix    string = "worker"
	servicePrefix   string = "service"
)

type RedisClient struct {
	RedisClient redis.Client
}

func CreateRedisClient(ctx context.Context, redisLogin config.RedisConf) (*RedisClient, error) {
	redisClient := redis.NewClient(&redis.Options{
		Addr:     redisLogin.Address,
		Password: redisLogin.Password,
		DB:       redisLogin.Db,
	})

	if redisLogin.FullPersistence {
		logrus.Warn("Modifications")
		if err := redisClient.ConfigSet(ctx, "appendonly", "yes").Err(); err != nil {
			return &RedisClient{}, err
		}

		if err := redisClient.ConfigSet(ctx, "appendfsync", "always").Err(); err != nil {
			return &RedisClient{}, err
		}
	} else {
		if err := redisClient.ConfigSet(ctx, "appendonly", "no").Err(); err != nil {
			return &RedisClient{}, err
		}

		if err := redisClient.ConfigSet(ctx, "appendfsync", "everysec").Err(); err != nil {
			return &RedisClient{}, err
		}
	}

	return &RedisClient{RedisClient: *redisClient}, redisClient.Ping(ctx).Err()
}

func (driver *RedisClient) StoreDataPlaneInformation(ctx context.Context, dataplaneInfo *proto.DataplaneInformation) error {
	logrus.Trace("store dataplane information in the database")

	data, err := proto2.Marshal(dataplaneInfo)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("%s:%s", dataplanePrefix, dataplaneInfo.Address)

	return driver.RedisClient.HSet(ctx, key, "data", data).Err()
}

func (driver *RedisClient) DeleteDataPlaneInformation(ctx context.Context, dataplaneInfo *proto.DataplaneInformation) error {
	logrus.Trace("delete dataplane information in the database")

	key := fmt.Sprintf("%s:%s", dataplanePrefix, dataplaneInfo.Address)

	return driver.RedisClient.Del(ctx, key).Err()
}

func (driver *RedisClient) GetDataPlaneInformation(ctx context.Context) ([]*proto.DataplaneInformation, error) {
	logrus.Trace("get dataplane information from the database")

	keys, err := driver.scanKeys(ctx, dataplanePrefix)
	if err != nil {
		return nil, err
	}

	dataPlanes := make([]*proto.DataplaneInformation, 0)

	for _, key := range keys {
		fields, err := driver.RedisClient.HGetAll(ctx, key).Result()
		if err != nil {
			return nil, err
		}

		dataPlaneInfo := &proto.DataplaneInformation{}
		err = proto2.Unmarshal([]byte(fields["data"]), dataPlaneInfo)

		if err != nil {
			panic(err)
		}

		dataPlanes = append(dataPlanes, dataPlaneInfo)
	}

	logrus.Tracef("Found %d dataplane(s) in the database", len(dataPlanes))

	return dataPlanes, nil
}

func (driver *RedisClient) StoreWorkerNodeInformation(ctx context.Context, workerNodeInfo *proto.WorkerNodeInformation) error {
	logrus.Trace("store worker node information in the database")

	data, err := proto2.Marshal(workerNodeInfo)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("%s:%s", workerPrefix, workerNodeInfo.Name)

	return driver.RedisClient.HSet(ctx, key, "data", data).Err()
}

func (driver *RedisClient) DeleteWorkerNodeInformation(ctx context.Context, name string) error {
	logrus.Trace("delete worker node information in the database")

	key := fmt.Sprintf("%s:%s", workerPrefix, name)

	return driver.RedisClient.Del(ctx, key, "data").Err()
}

func (driver *RedisClient) GetWorkerNodeInformation(ctx context.Context) ([]*proto.WorkerNodeInformation, error) {
	logrus.Trace("get workers information from the database")

	keys, err := driver.scanKeys(ctx, workerPrefix)
	if err != nil {
		return nil, err
	}

	workers := make([]*proto.WorkerNodeInformation, 0)

	for _, key := range keys {
		fields, err := driver.RedisClient.HGetAll(ctx, key).Result()
		if err != nil {
			return nil, err
		}

		workerNodeInfo := &proto.WorkerNodeInformation{}

		err = proto2.Unmarshal([]byte(fields["data"]), workerNodeInfo)
		if err != nil {
			panic(err)
		}

		workers = append(workers, workerNodeInfo)
	}

	logrus.Tracef("Found %d worker(s) in the database", len(workers))

	return workers, nil
}

func (driver *RedisClient) StoreServiceInformation(ctx context.Context, serviceInfo *proto.ServiceInfo) error {
	logrus.Trace("store service information in the database")

	data, err := proto2.Marshal(serviceInfo)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("%s:%s", servicePrefix, serviceInfo.Name)

	return driver.RedisClient.HSet(ctx, key, "data", data).Err()
}

func (driver *RedisClient) GetServiceInformation(ctx context.Context) ([]*proto.ServiceInfo, error) {
	logrus.Trace("get services information from the database")

	keys, err := driver.scanKeys(ctx, servicePrefix)
	if err != nil {
		return nil, err
	}

	services := make([]*proto.ServiceInfo, 0)

	for _, key := range keys {
		fields, err := driver.RedisClient.HGetAll(ctx, key).Result()
		if err != nil {
			return nil, err
		}

		serviceInfo := &proto.ServiceInfo{}

		err = proto2.Unmarshal([]byte(fields["data"]), serviceInfo)

		if err != nil {
			panic(err)
		}

		services = append(services, serviceInfo)
	}

	logrus.Tracef("Found %d service(s) in the database", len(services))

	return services, nil
}

func (driver *RedisClient) scanKeys(ctx context.Context, prefix string) ([]string, error) {
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

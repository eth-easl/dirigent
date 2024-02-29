package persistence

import (
	"cluster_manager/api/proto"
	"cluster_manager/pkg/config"
	"cluster_manager/pkg/redis_helpers"
	"context"
	"fmt"
	proto2 "github.com/golang/protobuf/proto"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"net"
	"sync"
	"time"
)

const (
	dataplanePrefix string = "dataplane"
	workerPrefix    string = "worker"
	servicePrefix   string = "service"
)

type RedisClient struct {
	RedisClient  *redis.Client
	OtherClients []*redis.Client

	Addr string
	Port string

	dataplaneTimestamp map[string]time.Time
	dataplaneLock      sync.Mutex

	workerTimestamp map[string]time.Time
	workerLock      sync.Mutex

	serviceTimestamp map[string]time.Time
	serviceLock      sync.Mutex
}

func CreateRedisClient(ctx context.Context, redisLogin config.RedisConf) (*RedisClient, error) {
	redisClient, otherClients, err := redis_helpers.CreateRedisConnector(ctx, redisLogin)

	address, port, err := net.SplitHostPort(redisLogin.DockerAddress)
	if err != nil {
		logrus.Fatalf("Invalid Redis parameters (address).")
	}

	return &RedisClient{
		RedisClient:        redisClient,
		OtherClients:       otherClients,
		Addr:               address,
		Port:               port,
		dataplaneTimestamp: make(map[string]time.Time),
		dataplaneLock:      sync.Mutex{},
		workerTimestamp:    make(map[string]time.Time),
		workerLock:         sync.Mutex{},
		serviceTimestamp:   make(map[string]time.Time),
		serviceLock:        sync.Mutex{},
	}, err
}

func (driver *RedisClient) StoreDataPlaneInformation(ctx context.Context, dataplaneInfo *proto.DataplaneInformation, timestamp time.Time) error {
	driver.dataplaneLock.Lock()
	defer driver.dataplaneLock.Unlock()

	if timestamp.Before(driver.dataplaneTimestamp[dataplaneInfo.Address]) {
		return nil
	} else {
		driver.dataplaneTimestamp[dataplaneInfo.Address] = timestamp
	}

	logrus.Trace("store dataplane information in the database")

	data, err := proto2.Marshal(dataplaneInfo)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("%s:%s", dataplanePrefix, dataplaneInfo.Address)

	return driver.RedisClient.HSet(ctx, key, "data", data).Err()
}

func (driver *RedisClient) DeleteDataPlaneInformation(ctx context.Context, dataplaneInfo *proto.DataplaneInformation, timestamp time.Time) error {
	driver.dataplaneLock.Lock()
	defer driver.dataplaneLock.Unlock()

	if timestamp.Before(driver.dataplaneTimestamp[dataplaneInfo.Address]) {
		return nil
	} else {
		driver.dataplaneTimestamp[dataplaneInfo.Address] = timestamp
	}

	logrus.Trace("delete dataplane information in the database")

	key := fmt.Sprintf("%s:%s", dataplanePrefix, dataplaneInfo.Address)

	return driver.RedisClient.Del(ctx, key).Err()
}

func (driver *RedisClient) GetDataPlaneInformation(ctx context.Context) ([]*proto.DataplaneInformation, error) {
	logrus.Trace("get dataplane information from the database")

	keys, err := redis_helpers.ScanKeys(ctx, driver.RedisClient, dataplanePrefix)
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

func (driver *RedisClient) StoreWorkerNodeInformation(ctx context.Context, workerNodeInfo *proto.WorkerNodeInformation, timestamp time.Time) error {
	driver.workerLock.Lock()
	defer driver.workerLock.Unlock()

	if timestamp.Before(driver.workerTimestamp[workerNodeInfo.Name]) {
		return nil
	} else {
		driver.workerTimestamp[workerNodeInfo.Name] = timestamp
	}

	logrus.Trace("store worker node information in the database")

	data, err := proto2.Marshal(workerNodeInfo)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("%s:%s", workerPrefix, workerNodeInfo.Name)

	return driver.RedisClient.HSet(ctx, key, "data", data).Err()
}

func (driver *RedisClient) DeleteWorkerNodeInformation(ctx context.Context, name string, timestamp time.Time) error {
	driver.workerLock.Lock()
	defer driver.workerLock.Unlock()

	if timestamp.Before(driver.workerTimestamp[name]) {
		return nil
	} else {
		driver.workerTimestamp[name] = timestamp
	}

	logrus.Trace("delete worker node information in the database")

	key := fmt.Sprintf("%s:%s", workerPrefix, name)

	return driver.RedisClient.Del(ctx, key, "data").Err()
}

func (driver *RedisClient) GetWorkerNodeInformation(ctx context.Context) ([]*proto.WorkerNodeInformation, error) {
	logrus.Trace("get workers information from the database")

	keys, err := redis_helpers.ScanKeys(ctx, driver.RedisClient, workerPrefix)
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

func (driver *RedisClient) StoreServiceInformation(ctx context.Context, serviceInfo *proto.ServiceInfo, timestamp time.Time) error {
	driver.serviceLock.Lock()
	defer driver.serviceLock.Unlock()

	if timestamp.Before(driver.serviceTimestamp[serviceInfo.Name]) {
		return nil
	} else {
		driver.serviceTimestamp[serviceInfo.Name] = timestamp
	}

	logrus.Trace("store service information in the database")

	data, err := proto2.Marshal(serviceInfo)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("%s:%s", servicePrefix, serviceInfo.Name)

	return driver.RedisClient.HSet(ctx, key, "data", data).Err()
}

func (driver *RedisClient) DeleteServiceInformation(ctx context.Context, serviceInfo *proto.ServiceInfo, timestamp time.Time) error {
	driver.serviceLock.Lock()
	defer driver.serviceLock.Unlock()

	if timestamp.Before(driver.serviceTimestamp[serviceInfo.Name]) {
		return nil
	} else {
		driver.serviceTimestamp[serviceInfo.Name] = timestamp
	}

	logrus.Trace("delete service information in the database")

	key := fmt.Sprintf("%s:%s", servicePrefix, serviceInfo.Name)

	return driver.RedisClient.Del(ctx, key, "data").Err()
}

func (driver *RedisClient) GetServiceInformation(ctx context.Context) ([]*proto.ServiceInfo, error) {
	logrus.Trace("get services information from the database")

	keys, err := redis_helpers.ScanKeys(ctx, driver.RedisClient, servicePrefix)
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

func (driver *RedisClient) SetLeader(ctx context.Context) error {
	if err := driver.RedisClient.SlaveOf(ctx, "no", "one").Err(); err != nil {
		return err
	}

	// TODO: Improve this part - make sure at least quorum is okay
	for _, otherClient := range driver.OtherClients {
		if err := otherClient.SlaveOf(ctx, driver.Addr, driver.Port).Err(); err != nil {
			logrus.Errorf("Failed set slave of %s:%s : %s", driver.Addr, driver.Port, err.Error())
		}
	}

	logrus.Info("Set local redis instance as the new redis master")

	return nil
}

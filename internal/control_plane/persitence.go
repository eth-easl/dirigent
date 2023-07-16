package control_plane

import (
	"cluster_manager/api/proto"
	"cluster_manager/pkg/config"
	"context"
	"fmt"
	proto2 "github.com/golang/protobuf/proto"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/json"
	"strings"
)

const (
	dataplanePrefix string = "dataplane"
	workerPrefix    string = "worker"
	servicePrefix   string = "service"
	endpointPrefix  string = "endpoint"
)

type RedisClient struct {
	redisClient *redis.Client
}

func CreateRedisClient(ctx context.Context, redisLogin config.RedisLogin) (RedisClient, error) {
	redisClient := redis.NewClient(&redis.Options{
		Addr:     redisLogin.Address,
		Password: redisLogin.Password,
		DB:       redisLogin.Db,
	})

	return RedisClient{redisClient: redisClient}, redisClient.Ping(ctx).Err()
}

type DataPlaneInformation struct {
	Address   string `redis:"address"`
	ApiPort   string `redis:"apiPort"`
	ProxyPort string `redis:"proxyPort"`
}

type WorkerNodeInformation struct {
	Name     string `redis:"name"`
	Ip       string `redis:"ip"`
	Port     string `redis:"port"`
	CpuCores string `redis:"cpuCores"`
	Memory   string `redis:"memory"`
}

type ServiceInformation struct {
	Name                                 string           `redis:"name"`
	Image                                string           `redis:"image"`
	HostPort                             int32            `redis:"hostPort"`
	GuestPort                            int32            `redis:"guestPort"`
	Protocol                             proto.L4Protocol `redis:"protocol"`
	ScalingUpperBound                    int32            `edis:"scalingUpperBound"`
	ScalingLowerBound                    int32            `redis:"scalingLowerBound"`
	PanicThresholdPercentage             float32          `redis:"panicThresholdPercentage"`
	MaxScaleUpRate                       float32          `redis:"maxScaleUpRate"`
	MaxScaleDownRate                     float32          `redis:"maxScaleDownRate"`
	ContainerConcurrency                 int32            `redis:"containerConcurrency"`
	ContainerConcurrencyTargetPercentage int32            `redis:"containerConcurrencyTargetPercentage"`
	StableWindowWidthSeconds             int32            `redis:"stableWindowWidthSeconds"`
	PanicWindowWidthSeconds              int32            `redis:"panicWindowWidthSeconds"`
	ScalingPeriodSeconds                 int32            `redis:"scalingPeriodSeconds"`
}

type EndpointInformation struct {
	SandboxId string `redis:"sandboxId"`
	URL       string `redis:"uRL"`
	NodeName  string `redis:"nodeName"`
	HostPort  int32  `redis:"hostPort"`
}

func (driver *RedisClient) scanKeys(ctx context.Context, prefix string) ([]string, error) {
	var (
		cursor uint64
		n      int
	)

	output := make([]string, 0)

	for {
		var keys []string
		var err error
		keys, cursor, err = driver.redisClient.Scan(ctx, cursor, fmt.Sprintf("%s*", prefix), 10).Result()
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

func (driver *RedisClient) StoreDataPlaneInformation(ctx context.Context, dataplaneInfo *proto.DataplaneInformation) error {
	logrus.Trace("store dataplane information in the database")

	data, err := proto2.Marshal(dataplaneInfo)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("%s:%s", dataplanePrefix, dataplaneInfo.Address)

	return driver.redisClient.HSet(ctx, key, "data", data).Err()
}

func (driver *RedisClient) GetDataPlaneInformation(ctx context.Context) ([]*proto.DataplaneInformation, error) {
	logrus.Trace("get dataplane information from the database")

	keys, err := driver.scanKeys(ctx, dataplanePrefix)
	if err != nil {
		return nil, err
	}

	dataPlanes := make([]*proto.DataplaneInformation, 0)

	for _, key := range keys {
		fields, err := driver.redisClient.HGetAll(ctx, key).Result()
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

	return driver.redisClient.HSet(ctx, key, "data", data).Err()
}

func (driver *RedisClient) GetWorkerNodeInformation(ctx context.Context) ([]*proto.WorkerNodeInformation, error) {
	logrus.Trace("get workers information from the database")

	keys, err := driver.scanKeys(ctx, workerPrefix)
	if err != nil {
		return nil, err
	}

	workers := make([]*proto.WorkerNodeInformation, 0)

	for _, key := range keys {
		fields, err := driver.redisClient.HGetAll(ctx, key).Result()
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

	key := fmt.Sprintf("%s:%s", servicePrefix, data)

	return driver.redisClient.HSet(ctx, key, "data", data).Err()
}

func (driver *RedisClient) StoreServiceInformationProto(ctx context.Context, serviceInfo *proto.ServiceInfo) error {
	logrus.Trace("store service information in the database")

	data, err := proto2.Marshal(serviceInfo)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("%s:%s", servicePrefix, data)

	return driver.redisClient.HSet(ctx, key, "data", data).Err()
}

func (driver *RedisClient) StoreServiceInformationJSON(ctx context.Context, serviceInfo *proto.ServiceInfo) error {
	logrus.Trace("store service information in the database")

	data, err := json.Marshal(serviceInfo)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("%s:%s", servicePrefix, data)

	return driver.redisClient.HSet(ctx, key, "data", data).Err()
}

func (driver *RedisClient) GetServiceInformation(ctx context.Context) ([]*proto.ServiceInfo, error) {
	logrus.Trace("get services information from the database")

	keys, err := driver.scanKeys(ctx, servicePrefix)
	if err != nil {
		return nil, err
	}

	services := make([]*proto.ServiceInfo, 0)

	for _, key := range keys {
		fields, err := driver.redisClient.HGetAll(ctx, key).Result()
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

func (driver *RedisClient) UpdateEndpoints(ctx context.Context, serviceName string, endpoints []*proto.Endpoint) error {
	logrus.Trace("store endpoints information in the database")

	key := fmt.Sprintf("%s:%s:*", endpointPrefix, serviceName)
	err := driver.DeleteEndpoints(ctx, key)
	if err != nil {
		return err
	}

	return driver.StoreEndpoints(ctx, serviceName, endpoints)
}

func (driver *RedisClient) StoreEndpoints(ctx context.Context, serviceName string, endpoints []*proto.Endpoint) error {
	for _, endpoint := range endpoints {
		key := fmt.Sprintf("%s:%s:%s", endpointPrefix, serviceName, endpoint.NodeName)

		data, err := proto2.Marshal(endpoint)
		if err != nil {
			return err
		}

		err = driver.redisClient.HSet(ctx, key, data).Err()
		if err != nil {
			driver.DeleteEndpoints(ctx, serviceName)
			return err
		}
	}

	return nil
}

func (driver *RedisClient) DeleteEndpoints(ctx context.Context, key string) error {
	return driver.redisClient.Del(ctx, key).Err()
}

func (driver *RedisClient) GetEndpoints(ctx context.Context) ([]*proto.Endpoint, []string, error) {
	logrus.Trace("get endpoints information from the database")

	keys, err := driver.scanKeys(ctx, endpointPrefix)
	if err != nil {
		return nil, nil, err
	}

	endpoints := make([]*proto.Endpoint, 0)
	services := make([]string, 0)

	for _, key := range keys {
		fields, err := driver.redisClient.HGetAll(ctx, key).Result()
		if err != nil {
			return nil, nil, err
		}

		endpoint := &proto.Endpoint{}
		err = proto2.Unmarshal([]byte(fields["data"]), endpoint)
		if err != nil {
			panic(err)
		}

		endpoints = append(endpoints, endpoint)
		services = append(services, strings.Split(key, ":")[1])
	}

	logrus.Tracef("Found %d endpoint(s) in the database", len(endpoints))

	return endpoints, services, nil
}

package control_plane

import (
	"cluster_manager/api/proto"
	"cluster_manager/pkg/config"
	"context"
	"errors"
	"fmt"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"strconv"
)

const (
	address   string = "address"
	apiPort   string = "apiPort"
	proxyPort string = "proxyPort"

	name     string = "name"
	ip       string = "ip"
	port     string = "port"
	cpuCores string = "cpuCores"
	memory   string = "memory"

	image                                string = "image"
	hostPort                             string = "hostport"
	guestPort                            string = "guestport"
	protocol                             string = "protocol"
	scalingUpperBound                    string = "scalingupperbound"
	scalingLowerBound                    string = "scalinglowerbound"
	panicThresholdPercentage             string = "panicthresholdpercentage"
	maxScaleUpRate                       string = "maxscaleuprate"
	maxScaleDownRate                     string = "maxscaledownrate"
	ContainerConcurrency                 string = "containerconcurrency"
	containerConcurrencyTargetPercentage string = "containerconcurrencytargetpercentage"
	stableWindowWidthSeconds             string = "stabledinwowidthseconds"
	panicWindowWidthSeconds              string = "panicwindowswidthseconds"
	scalingPeriodSeconds                 string = "scalingperiodseconds"
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
	ScalingUpperBound                    int32            `redis:"scalingUpperBound"`
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
	var cursor uint64
	var n int
	var keys []string
	for {
		var err error
		keys, cursor, err = driver.redisClient.Scan(ctx, cursor, prefix, 10).Result()
		if err != nil {
			return keys, err
		}
		n += len(keys)
		if cursor == 0 {
			break
		}
	}

	return keys, nil
}

func (driver *RedisClient) StoreDataPlaneInformation(ctx context.Context, key string, dataplaneInfo DataPlaneInformation) error {
	logrus.Trace("store dataplane information in the database")
	return driver.redisClient.HSet(ctx, key, dataplaneInfo).Err()
}

func (driver *RedisClient) GetDataPlaneInformation(ctx context.Context) ([]*DataPlaneInformation, error) {
	logrus.Trace("get dataplane information from the database")

	keys, err := driver.scanKeys(ctx, "dataplane*")
	if err != nil {
		return nil, err
	}

	dataPlanes := make([]*DataPlaneInformation, 0)

	for _, key := range keys {
		fields, err := driver.redisClient.HGetAll(ctx, key).Result()
		if err != nil {
			return nil, err
		}

		dataPlanes = append(dataPlanes, &DataPlaneInformation{
			Address:   fields[address],
			ApiPort:   fields[apiPort],
			ProxyPort: fields[proxyPort],
		})
	}

	logrus.Tracef("Found %d dataplane(s) in the database", len(dataPlanes))

	return dataPlanes, nil
}

func (driver *RedisClient) StoreWorkerNodeInformation(ctx context.Context, key string, workerNodeInfo WorkerNodeInformation) error {
	logrus.Trace("store worker node information in the database")
	return driver.redisClient.HSet(ctx, key, workerNodeInfo).Err()
}

func (driver *RedisClient) GetWorkerNodeInformation(ctx context.Context, key string) (*WorkerNodeInformation, error) {
	fields, err := driver.redisClient.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	return &WorkerNodeInformation{
		Name:     fields[name],
		Ip:       fields[ip],
		Port:     fields[port],
		CpuCores: fields[cpuCores],
		Memory:   fields[memory],
	}, nil
}

func (driver *RedisClient) StoreServiceInformation(ctx context.Context, key string, serviceInfo *proto.ServiceInfo) error {
	if serviceInfo == nil || serviceInfo.AutoscalingConfig == nil || serviceInfo.PortForwarding == nil {
		return errors.New("Struct to save is incomplete")
	}

	return driver.redisClient.HSet(ctx, key,
		name, serviceInfo.Name,
		image, serviceInfo.Image,
		hostPort, serviceInfo.PortForwarding.HostPort,
		guestPort, serviceInfo.PortForwarding.GuestPort,
		protocol, serviceInfo.PortForwarding.Protocol,
		scalingUpperBound, serviceInfo.AutoscalingConfig.ScalingUpperBound,
		scalingLowerBound, serviceInfo.AutoscalingConfig.ScalingLowerBound,
		panicThresholdPercentage, serviceInfo.AutoscalingConfig.PanicThresholdPercentage,
		maxScaleUpRate, serviceInfo.AutoscalingConfig.MaxScaleUpRate,
		maxScaleDownRate, serviceInfo.AutoscalingConfig.MaxScaleDownRate,
		ContainerConcurrency, serviceInfo.AutoscalingConfig.ContainerConcurrency,
		containerConcurrencyTargetPercentage, serviceInfo.AutoscalingConfig.ContainerConcurrencyTargetPercentage,
		stableWindowWidthSeconds, serviceInfo.AutoscalingConfig.StableWindowWidthSeconds,
		panicWindowWidthSeconds, serviceInfo.AutoscalingConfig.PanicWindowWidthSeconds,
		scalingPeriodSeconds, serviceInfo.AutoscalingConfig.ScalingPeriodSeconds).Err()
}

func (driver *RedisClient) GetServiceInformation(ctx context.Context, key string) (*proto.ServiceInfo, error) {
	fields, err := driver.redisClient.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	hostPort, err := strconv.Atoi(fields[hostPort])
	guestPort, err := strconv.Atoi(fields[guestPort])
	protocol, err := strconv.Atoi(fields[protocol])

	scalingUpperBound, err := strconv.Atoi(fields[scalingUpperBound])
	scalingLowerBound, err := strconv.Atoi(fields[scalingLowerBound])
	panicThresholdPercentage, err := strconv.ParseFloat(fields[panicThresholdPercentage], 32)
	maxScaleUpRate, err := strconv.ParseFloat(fields[maxScaleUpRate], 32)
	maxScaleDownRate, err := strconv.ParseFloat(fields[maxScaleDownRate], 32)
	containerConcurrency, err := strconv.Atoi(fields[ContainerConcurrency])
	containerConcurrencyTargetPercentage, err := strconv.Atoi(fields[containerConcurrencyTargetPercentage])
	stableWindowWidthSeconds, err := strconv.Atoi(fields[stableWindowWidthSeconds])
	panicWindowWidthSeconds, err := strconv.Atoi(fields[panicWindowWidthSeconds])
	scalingPeriodSeconds, err := strconv.Atoi(fields[scalingPeriodSeconds])

	return &proto.ServiceInfo{
		Name:  fields[name],
		Image: fields[image],
		PortForwarding: &proto.PortMapping{
			HostPort:  int32(hostPort),
			GuestPort: int32(guestPort),
			Protocol:  proto.L4Protocol(protocol),
		},
		AutoscalingConfig: &proto.AutoscalingConfiguration{
			ScalingUpperBound:                    int32(scalingUpperBound),
			ScalingLowerBound:                    int32(scalingLowerBound),
			PanicThresholdPercentage:             float32(panicThresholdPercentage),
			MaxScaleUpRate:                       float32(maxScaleUpRate),
			MaxScaleDownRate:                     float32(maxScaleDownRate),
			ContainerConcurrency:                 int32(containerConcurrency),
			ContainerConcurrencyTargetPercentage: int32(containerConcurrencyTargetPercentage),
			StableWindowWidthSeconds:             int32(stableWindowWidthSeconds),
			PanicWindowWidthSeconds:              int32(panicWindowWidthSeconds),
			ScalingPeriodSeconds:                 int32(scalingPeriodSeconds),
		},
	}, nil
}

func (driver *RedisClient) UpdateEndpoints(ctx context.Context, key string, endpoints []*Endpoint) error {
	err := driver.DeleteEndpoints(ctx, key)
	if err != nil {
		return err
	}

	return driver.StoreEndpoints(ctx, key, endpoints)
}

func (driver *RedisClient) StoreEndpoints(ctx context.Context, key string, endpoints []*Endpoint) error {
	for _, endpoint := range endpoints {
		err := driver.redisClient.HSet(ctx, fmt.Sprintf("%s:%s", key, endpoint.Node.Name), EndpointInformation{
			SandboxId: endpoint.SandboxID,
			URL:       endpoint.URL,
			NodeName:  endpoint.Node.Name,
			HostPort:  endpoint.HostPort,
		}).Err()
		if err != nil {
			driver.DeleteEndpoints(ctx, key)
			return err
		}
	}

	return nil
}

func (driver *RedisClient) DeleteEndpoints(ctx context.Context, key string) error {
	return driver.redisClient.Del(ctx, key).Err()
}

func (driver *RedisClient) GetEndpoints(ctx context.Context, key string) ([]*EndpointInformation, error) {
	return make([]*EndpointInformation, 0), nil
}

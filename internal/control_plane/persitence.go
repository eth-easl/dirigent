package control_plane

import (
	"cluster_manager/api/proto"
	"cluster_manager/pkg/config"
	"context"
	"errors"
	"github.com/redis/go-redis/v9"
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

func (driver *RedisClient) StoreDataPlaneInformation(ctx context.Context, key string, dataplaneInfo DataPlaneInformation) error {
	return driver.redisClient.HMSet(ctx, key, dataplaneInfo).Err()
}

func (driver *RedisClient) GetDataPlaneInformation(ctx context.Context, key string) (*DataPlaneInformation, error) {
	fields, err := driver.redisClient.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	return &DataPlaneInformation{
		Address:   fields[address],
		ApiPort:   fields[apiPort],
		ProxyPort: fields[proxyPort],
	}, nil
}

func (driver *RedisClient) StoreWorkerNodeInformation(ctx context.Context, key string, workerNodeInfo WorkerNodeInformation) error {
	return driver.redisClient.HMSet(ctx, key, workerNodeInfo).Err()
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

	return driver.redisClient.HMSet(ctx, key,
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

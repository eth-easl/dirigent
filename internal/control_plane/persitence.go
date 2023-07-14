package control_plane

import (
	"cluster_manager/api/proto"
	"cluster_manager/pkg/config"
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
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

	SandboxId string = "sandboxId"
	URL       string = "uRL"
	NodeName  string = "nodeName"
	HostPort  string = "hostPort"

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
	var (
		cursor uint64
		n      int
		keys   []string
	)

	for {
		var err error
		keys, cursor, err = driver.redisClient.Scan(ctx, cursor, fmt.Sprintf("%s*", prefix), 10).Result()

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

func (driver *RedisClient) StoreDataPlaneInformation(ctx context.Context, dataplaneInfo DataPlaneInformation) error {
	logrus.Trace("store dataplane information in the database")

	key := fmt.Sprintf("%s:%s", dataplanePrefix, dataplaneInfo.Address)

	return driver.redisClient.HSet(ctx, key, dataplaneInfo).Err()
}

func (driver *RedisClient) GetDataPlaneInformation(ctx context.Context) ([]*DataPlaneInformation, error) {
	logrus.Trace("get dataplane information from the database")

	keys, err := driver.scanKeys(ctx, dataplanePrefix)
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

func (driver *RedisClient) StoreWorkerNodeInformation(ctx context.Context, workerNodeInfo WorkerNodeInformation) error {
	logrus.Trace("store worker node information in the database")

	key := fmt.Sprintf("%s:%s", workerPrefix, workerNodeInfo.Name)

	return driver.redisClient.HSet(ctx, key, workerNodeInfo).Err()
}

func (driver *RedisClient) GetWorkerNodeInformation(ctx context.Context) ([]*WorkerNodeInformation, error) {
	logrus.Trace("get workers information from the database")

	keys, err := driver.scanKeys(ctx, workerPrefix)
	if err != nil {
		return nil, err
	}

	workers := make([]*WorkerNodeInformation, 0)

	for _, key := range keys {
		fields, err := driver.redisClient.HGetAll(ctx, key).Result()
		if err != nil {
			return nil, err
		}

		workers = append(workers, &WorkerNodeInformation{
			Name:     fields[name],
			Ip:       fields[ip],
			Port:     fields[port],
			CpuCores: fields[cpuCores],
			Memory:   fields[memory],
		})
	}

	logrus.Tracef("Found %d worker(s) in the database", len(workers))

	return workers, nil
}

func (driver *RedisClient) StoreServiceInformation(ctx context.Context, serviceInfo *proto.ServiceInfo) error {
	logrus.Trace("store service information in the database")

	if serviceInfo == nil || serviceInfo.AutoscalingConfig == nil || serviceInfo.PortForwarding == nil {
		return errors.New("Struct to save is incomplete")
	}

	key := fmt.Sprintf("%s:%s", servicePrefix, serviceInfo.Name)

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

		hostPort, _ := strconv.Atoi(fields[hostPort])
		guestPort, _ := strconv.Atoi(fields[guestPort])
		protocol, _ := strconv.Atoi(fields[protocol])

		scalingUpperBound, _ := strconv.Atoi(fields[scalingUpperBound])
		scalingLowerBound, _ := strconv.Atoi(fields[scalingLowerBound])
		panicThresholdPercentage, _ := strconv.ParseFloat(fields[panicThresholdPercentage], 32)
		maxScaleUpRate, _ := strconv.ParseFloat(fields[maxScaleUpRate], 32)
		maxScaleDownRate, _ := strconv.ParseFloat(fields[maxScaleDownRate], 32)
		containerConcurrency, _ := strconv.Atoi(fields[ContainerConcurrency])
		containerConcurrencyTargetPercentage, _ := strconv.Atoi(fields[containerConcurrencyTargetPercentage])
		stableWindowWidthSeconds, _ := strconv.Atoi(fields[stableWindowWidthSeconds])
		panicWindowWidthSeconds, _ := strconv.Atoi(fields[panicWindowWidthSeconds])
		scalingPeriodSeconds, _ := strconv.Atoi(fields[scalingPeriodSeconds])

		services = append(services, &proto.ServiceInfo{
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
		})
	}

	logrus.Tracef("Found %d service(s) in the database", len(services))

	return services, nil
}

func (driver *RedisClient) UpdateEndpoints(ctx context.Context, serviceName string, endpoints []*Endpoint) error {
	logrus.Trace("store endpoints information in the database")

	key := fmt.Sprintf("%s:%s:*", endpointPrefix, serviceName)
	err := driver.DeleteEndpoints(ctx, key)
	if err != nil {
		return err
	}

	return driver.StoreEndpoints(ctx, serviceName, endpoints)
}

func (driver *RedisClient) StoreEndpoints(ctx context.Context, serviceName string, endpoints []*Endpoint) error {
	for _, endpoint := range endpoints {
		key := fmt.Sprintf("%s:%s:%s", endpointPrefix, serviceName, endpoint.Node.Name)

		err := driver.redisClient.HSet(ctx, key, EndpointInformation{
			SandboxId: endpoint.SandboxID,
			URL:       endpoint.URL,
			NodeName:  endpoint.Node.Name,
			HostPort:  endpoint.HostPort,
		}).Err()
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

func (driver *RedisClient) GetEndpoints(ctx context.Context) ([]*EndpointInformation, []string, error) {
	logrus.Trace("get endpoints information from the database")

	keys, err := driver.scanKeys(ctx, endpointPrefix)
	if err != nil {
		return nil, nil, err
	}

	endpoints := make([]*EndpointInformation, 0)
	services := make([]string, 0)

	for _, key := range keys {
		fields, err := driver.redisClient.HGetAll(ctx, key).Result()
		if err != nil {
			return nil, nil, err
		}

		hostPort, _ := strconv.Atoi(fields[HostPort])

		endpoints = append(endpoints, &EndpointInformation{
			SandboxId: fields[SandboxId],
			URL:       fields[URL],
			NodeName:  fields[NodeName],
			HostPort:  int32(hostPort),
		})

		services = append(services, strings.Split(key, ":")[1])
	}

	logrus.Tracef("Found %d endpoint(s) in the database", len(endpoints))

	return endpoints, services, nil
}

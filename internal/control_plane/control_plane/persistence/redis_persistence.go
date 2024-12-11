package persistence

import (
	"cluster_manager/pkg/config"
	"cluster_manager/pkg/redis_helpers"
	"cluster_manager/proto"
	"context"
	"fmt"
	"net"

	proto2 "github.com/golang/protobuf/proto"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

const (
	dataplanePrefix string = "dataplane"
	workerPrefix    string = "worker"
	servicePrefix   string = "service"
	taskPrefix      string = "task"
	workflowPrefix  string = "workflow"
)

type RedisClient struct {
	RedisClient  *redis.Client
	OtherClients []*redis.Client

	Addr string
	Port string
}

func CreateRedisClient(ctx context.Context, redisLogin config.RedisConf) (*RedisClient, error) {
	redisClient, otherClients, err := redis_helpers.CreateRedisConnector(ctx, redisLogin)
	if err != nil {
		logrus.Fatalf("Failed to create Redis client")
	}

	address, port, err := net.SplitHostPort(redisLogin.AddressFromDockerNetwork)
	if err != nil {
		logrus.Fatalf("Invalid Redis parameters (address).")
	}

	return &RedisClient{
		RedisClient:  redisClient,
		OtherClients: otherClients,
		Addr:         address,
		Port:         port,
	}, err
}

func (driver *RedisClient) WaitForQuorumWrite(ctx context.Context) *redis.IntCmd {
	// 2 nodes -> 1 slave  -> need 1 more for quorum
	// 3 nodes -> 2 slaves -> need 1 more for quorum
	// 4 nodes -> 3 slaves -> need 2 more for quorum
	// 5 nodes -> 4 slaves -> need 2 more for quorum
	// requiredWrites := int(math.Ceil(float64(len(driver.OtherClients) / 2)))

	return driver.RedisClient.Wait(ctx, len(driver.OtherClients), 0)
}

func (driver *RedisClient) StoreDataPlaneInformation(ctx context.Context, dataplaneInfo *proto.DataplaneInformation) error {
	logrus.Trace("store dataplane information in the database")

	data, err := proto2.Marshal(dataplaneInfo)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("%s:%s", dataplanePrefix, dataplaneInfo.Address)

	err = driver.RedisClient.HSet(ctx, key, "data", data).Err()
	if err != nil {
		return err
	}

	return driver.WaitForQuorumWrite(ctx).Err()
}

func (driver *RedisClient) DeleteDataPlaneInformation(ctx context.Context, dataplaneInfo *proto.DataplaneInformation) error {
	logrus.Trace("delete dataplane information in the database")

	key := fmt.Sprintf("%s:%s", dataplanePrefix, dataplaneInfo.Address)

	err := driver.RedisClient.Del(ctx, key).Err()
	if err != nil {
		return err
	}

	return driver.WaitForQuorumWrite(ctx).Err()
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
			logrus.Fatalf("Failed to unmarshal dataplane information: %v", err)
		}

		dataPlanes = append(dataPlanes, dataPlaneInfo)
	}

	logrus.Tracef("Found %d dataplane(s) in the database", len(dataPlanes))

	return dataPlanes, nil
}

func (driver *RedisClient) StoreWorkerNodeInformation(ctx context.Context, workerNodeInfo *proto.NodeInfo) error {
	logrus.Trace("store worker node information in the database")

	data, err := proto2.Marshal(workerNodeInfo)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("%s:%s", workerPrefix, workerNodeInfo.NodeID)

	err = driver.RedisClient.HSet(ctx, key, "data", data).Err()
	if err != nil {
		return err
	}

	return driver.WaitForQuorumWrite(ctx).Err()
}

func (driver *RedisClient) DeleteWorkerNodeInformation(ctx context.Context, name string) error {
	logrus.Trace("delete worker node information in the database")

	key := fmt.Sprintf("%s:%s", workerPrefix, name)

	err := driver.RedisClient.Del(ctx, key, "data").Err()
	if err != nil {
		return err
	}

	return driver.WaitForQuorumWrite(ctx).Err()
}

func (driver *RedisClient) GetWorkerNodeInformation(ctx context.Context) ([]*proto.NodeInfo, error) {
	logrus.Trace("get workers information from the database")

	keys, err := redis_helpers.ScanKeys(ctx, driver.RedisClient, workerPrefix)
	if err != nil {
		return nil, err
	}

	workers := make([]*proto.NodeInfo, 0)

	for _, key := range keys {
		fields, err := driver.RedisClient.HGetAll(ctx, key).Result()
		if err != nil {
			return nil, err
		}

		workerNodeInfo := &proto.NodeInfo{}

		err = proto2.Unmarshal([]byte(fields["data"]), workerNodeInfo)
		if err != nil {
			logrus.Fatalf("Failed to unmarshal worker information: %v", err)
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

	err = driver.RedisClient.HSet(ctx, key, "data", data).Err()
	if err != nil {
		return err
	}

	return driver.WaitForQuorumWrite(ctx).Err()
}

func (driver *RedisClient) DeleteServiceInformation(ctx context.Context, serviceInfo *proto.ServiceInfo) error {
	logrus.Trace("delete service information in the database")

	key := fmt.Sprintf("%s:%s", servicePrefix, serviceInfo.Name)

	err := driver.RedisClient.Del(ctx, key, "data").Err()
	if err != nil {
		return err
	}

	return driver.WaitForQuorumWrite(ctx).Err()
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
			logrus.Fatalf("Failed to unmarshal service information: %v", err)
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
			logrus.Errorf("Failed to set slave of %s:%s : %s", driver.Addr, driver.Port, err.Error())
		}
	}

	logrus.Info("Set local Redis instance as the new Redis master")

	return nil
}

func (driver *RedisClient) StoreWorkflowTaskInformation(ctx context.Context, wfTaskInfo *proto.WorkflowTaskInfo) error {
	logrus.Trace("store workflow task information in the database")

	data, err := proto2.Marshal(wfTaskInfo)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("%s:%s", taskPrefix, wfTaskInfo.Name)

	err = driver.RedisClient.HSet(ctx, key, "data", data).Err()
	if err != nil {
		return err
	}

	return driver.WaitForQuorumWrite(ctx).Err()
}

func (driver *RedisClient) DeleteWorkflowTaskInformation(ctx context.Context, name string) error {
	logrus.Trace("delete workflow task information in the database")

	key := fmt.Sprintf("%s:%s", taskPrefix, name)

	err := driver.RedisClient.Del(ctx, key, "data").Err()
	if err != nil {
		return err
	}

	return driver.WaitForQuorumWrite(ctx).Err()
}

func (driver *RedisClient) GetWorkflowTaskInformation(ctx context.Context) ([]*proto.WorkflowTaskInfo, error) {
	logrus.Trace("get workflow task information from the database")

	keys, err := redis_helpers.ScanKeys(ctx, driver.RedisClient, taskPrefix)
	if err != nil {
		return nil, err
	}

	tasks := make([]*proto.WorkflowTaskInfo, 0)

	for _, key := range keys {
		fields, err := driver.RedisClient.HGetAll(ctx, key).Result()
		if err != nil {
			return nil, err
		}

		taskInfo := &proto.WorkflowTaskInfo{}

		err = proto2.Unmarshal([]byte(fields["data"]), taskInfo)

		if err != nil {
			logrus.Fatalf("Failed to unmarshal workflow task information: %v", err)
		}

		tasks = append(tasks, taskInfo)
	}

	logrus.Tracef("Found %d workflow task(s) in the database", len(tasks))

	return tasks, nil
}

func (driver *RedisClient) StoreWorkflowInformation(ctx context.Context, wfInfo *proto.WorkflowInfo) error {
	logrus.Trace("store workflow information in the database")

	data, err := proto2.Marshal(wfInfo)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("%s:%s", workflowPrefix, wfInfo.Name)

	err = driver.RedisClient.HSet(ctx, key, "data", data).Err()
	if err != nil {
		return err
	}

	return driver.WaitForQuorumWrite(ctx).Err()
}

func (driver *RedisClient) DeleteWorkflowInformation(ctx context.Context, name string) error {
	logrus.Trace("delete workflow information in the database")

	key := fmt.Sprintf("%s:%s", workflowPrefix, name)

	err := driver.RedisClient.Del(ctx, key, "data").Err()
	if err != nil {
		return err
	}

	return driver.WaitForQuorumWrite(ctx).Err()
}

func (driver *RedisClient) GetWorkflowInformation(ctx context.Context) ([]*proto.WorkflowInfo, error) {
	logrus.Trace("get workflow information from the database")

	keys, err := redis_helpers.ScanKeys(ctx, driver.RedisClient, workflowPrefix)
	if err != nil {
		return nil, err
	}

	workflows := make([]*proto.WorkflowInfo, 0)

	for _, key := range keys {
		fields, err := driver.RedisClient.HGetAll(ctx, key).Result()
		if err != nil {
			return nil, err
		}

		wfInfo := &proto.WorkflowInfo{}

		err = proto2.Unmarshal([]byte(fields["data"]), wfInfo)

		if err != nil {
			logrus.Fatalf("Failed to unmarshal workflow information: %v", err)
		}

		workflows = append(workflows, wfInfo)
	}

	logrus.Tracef("Found %d workflow(s) in the database", len(workflows))

	return workflows, nil
}

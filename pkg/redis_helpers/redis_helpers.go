package redis_helpers

import (
	"cluster_manager/pkg/config"
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

func CreateRedisConnector(ctx context.Context, redisLogin config.RedisConf) (*redis.Client, error) {
	redisClient := redis.NewClient(&redis.Options{
		Addr:     redisLogin.Address,
		Password: redisLogin.Password,
		DB:       redisLogin.Db,
	})

	if redisLogin.FullPersistence {
		logrus.Warn("Modifications")
		if err := redisClient.ConfigSet(ctx, "appendonly", "yes").Err(); err != nil {
			return nil, err
		}

		if err := redisClient.ConfigSet(ctx, "appendfsync", "always").Err(); err != nil {
			return nil, err
		}
	} else {
		if err := redisClient.ConfigSet(ctx, "appendonly", "no").Err(); err != nil {
			return nil, err
		}

		if err := redisClient.ConfigSet(ctx, "appendfsync", "everysec").Err(); err != nil {
			return nil, err
		}
	}

	return redisClient, redisClient.Ping(ctx).Err()
}

func ScanKeys(ctx context.Context, client *redis.Client, prefix string) ([]string, error) {
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

		keys, cursor, err = client.Scan(ctx, cursor, fmt.Sprintf("%s*", prefix), 10).Result()
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

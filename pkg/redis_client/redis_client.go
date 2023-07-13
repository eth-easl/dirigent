package redis_client

import (
	"cluster_manager/pkg/config"
	"context"
	"github.com/redis/go-redis/v9"
)

func CreateRedisClient(ctx context.Context, redisLogin config.RedisLogin) (*redis.Client, error) {
	redisClient := redis.NewClient(&redis.Options{
		Addr:     redisLogin.Address,
		Password: redisLogin.Password,
		DB:       redisLogin.Db,
	})

	return redisClient, redisClient.Ping(ctx).Err()
}

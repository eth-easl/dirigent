package main

import (
	"cluster_manager/internal/control_plane/persistence"
	"cluster_manager/pkg/config"
	"context"
)

func main() {
	client, err := persistence.CreateRedisClient(context.Background(), config.RedisConf{
		Address:  "localhost:6379",
		Password: "",
		Db:       0,
	})

	err = client.RedisClient.FlushAll(context.Background()).Err()
	if err != nil {
		panic(err)
	}
}

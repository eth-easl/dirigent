package persistence

import (
	"cluster_manager/pkg/config"
	"context"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestFlushDB(t *testing.T) {
	client, err := CreateRedisClient(context.Background(), config.RedisLogin{
		Address:  "127.0.0.1:6379",
		Password: "",
		Db:       0,
	})

	assert.NoError(t, err, "Failed to connect to redis database")

	// Flush all values in the database
	err = client.redisClient.FlushAll(context.Background()).Err()
	assert.NoError(t, err, "Failed to flush all values from the database")
}

func TestFlushDBServer(t *testing.T) {
	client, err := CreateRedisClient(context.Background(), config.RedisLogin{
		Address:  "10.0.1.1:6379",
		Password: "",
		Db:       0,
	})

	assert.NoError(t, err, "Failed to connect to redis database")

	// Flush all values in the database
	err = client.redisClient.FlushAll(context.Background()).Err()
	assert.NoError(t, err, "Failed to flush all values from the database")
}

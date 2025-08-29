// file: db/redis.go

package db

import (
	"context"
	"fmt"
	"go-bank-api/config"
	"go-bank-api/logger"

	"github.com/redis/go-redis/v9"
)

// ConnectRedis initializes and returns a new Redis client.
// It uses the configuration from the loaded AppConfig.
func ConnectRedis() (*redis.Client, error) {
	cfg := config.AppConfig.Redis

	// Construct the address for the Redis server.
	redisAddr := fmt.Sprintf("%s:%s", cfg.Host, cfg.Port)

	// Create a new Redis client.
	rdb := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: cfg.Password, // set password
		DB:       0,            // use default DB
	})

	// Ping the Redis server to ensure a connection can be established.
	ctx := context.Background()
	if _, err := rdb.Ping(ctx).Result(); err != nil {
		logger.Log.WithError(err).Error("Failed to ping Redis")
		return nil, fmt.Errorf("failed to ping redis: %w", err)
	}

	logger.Log.WithField("address", redisAddr).Info("Redis connection established successfully")
	return rdb, nil
}

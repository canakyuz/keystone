package database

import (
	"context"
	"fmt"
	"log"

	"github.com/go-redis/redis/v8"
	"nexspaces-api/internal/config"
)

type RedisClient struct {
	*redis.Client
}

func NewRedis(cfg *config.RedisConfig) (*RedisClient, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	// Test connection
	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	log.Println("✅ Connected to Redis")
	return &RedisClient{rdb}, nil
}

func (r *RedisClient) Close() error {
	log.Println("🔌 Closing Redis connection")
	return r.Client.Close()
}

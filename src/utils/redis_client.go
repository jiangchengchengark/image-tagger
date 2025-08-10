package utils

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

var redis_logger = NewLogger("redis_client", "logs/redis_client.log")

type RedisClient struct {
	Client *redis.Client
}

// NewClient_redis 初始化 Redis 客户端
func NewClient_redis(addr, password string, db int) (*RedisClient, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Ping 验证连接
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("ping Redis failed: %v", err)
	}

	redis_logger.Printf("✅ Redis connected: %s (db: %d)", addr, db)

	return &RedisClient{
		Client: client,
	}, nil
}

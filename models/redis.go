package models

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/xiufeng-chen278/developer-platform-backend/config"
)

var redisClient *redis.Client

// InitRedis 初始化 redis 客户端并做一次 ping 检查。
func InitRedis(cfg *config.Config) (*redis.Client, error) {
	if cfg == nil {
		return nil, ErrMissingConfig
	}
	if redisClient != nil {
		return redisClient, nil
	}
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}
	redisClient = client
	return redisClient, nil
}

// GetRedis 返回初始化后的实例。
func GetRedis() *redis.Client {
	return redisClient
}

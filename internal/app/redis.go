package app

import (
	"context"
	"fmt"

	"github.com/kar1hsu/backplane/internal/pkg/cache"
	"github.com/redis/go-redis/v9"
)

var Redis *redis.Client

func InitRedis() error {
	cfg := Cfg.Redis

	Redis = redis.NewClient(&redis.Options{
		Addr:     cfg.Addr(),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	if err := Redis.Ping(context.Background()).Err(); err != nil {
		return fmt.Errorf("connect redis failed: %w", err)
	}

	cache.InitStore(cache.NewRedisStore(Redis, cfg.KeyPrefix))

	return nil
}

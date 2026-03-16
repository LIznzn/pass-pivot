package db

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"

	"pass-pivot/internal/config"
)

func OpenRedis(ctx context.Context, cfg config.Config) (*redis.Client, error) {
	if !cfg.RedisEnabled {
		return nil, nil
	}

	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.RedisHost, cfg.RedisPort),
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}
	return client, nil
}

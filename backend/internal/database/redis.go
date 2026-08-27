package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
	"gocommerce-backend/internal/config"
)

// InitRedis initializes and tests the connection to Redis cache and rate limiter server
func InitRedis(cfg *config.Config) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:         cfg.Redis.Addr(),
		Password:     cfg.Redis.Password,
		DB:           cfg.Redis.DB,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     20,
		MinIdleConns: 5,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Printf("⚠️ Redis connection warning: %v (rate limiting will fallback to in-memory/pass-through if unavailable)\n", err)
		return rdb, fmt.Errorf("redis ping failed: %w", err)
	}

	log.Printf("⚡ Connected to Redis Server at %s (DB: %d)\n", cfg.Redis.Addr(), cfg.Redis.DB)
	return rdb, nil
}

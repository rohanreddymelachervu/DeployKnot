package database

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

// Redis represents the Redis connection
type Redis struct {
	Client *redis.Client
	logger *logrus.Logger
}

// NewRedis creates a new Redis connection
func NewRedis(redisURL string, logger *logrus.Logger) (*Redis, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	client := redis.NewClient(opts)

	// Test the connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to ping Redis: %w", err)
	}

	logger.Info("Redis connection established")

	return &Redis{
		Client: client,
		logger: logger,
	}, nil
}

// Close closes the Redis connection
func (r *Redis) Close() error {
	if r.Client != nil {
		return r.Client.Close()
	}
	return nil
}

// HealthCheck performs a health check on Redis
func (r *Redis) HealthCheck() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return r.Client.Ping(ctx).Err()
}

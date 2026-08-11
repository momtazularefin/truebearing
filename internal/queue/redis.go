// Package queue provides Redis connection utilities for job queuing.
package queue

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// Connect parses the given Redis URL, creates a client, verifies connectivity
// with a ping, and returns the client. The caller is responsible for calling
// client.Close.
func Connect(ctx context.Context, redisURL string) (*redis.Client, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("queue: parse redis url: %w", err)
	}

	client := redis.NewClient(opts)

	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("queue: unable to ping redis: %w", err)
	}

	return client, nil
}

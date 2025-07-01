package redis

import (
	"context"
	"github.com/go-redis/redis/v8"
)

// Client wraps Redis operations
type Client struct {
	rdb *redis.Client
}

// NewClient creates a new Redis client
func NewClient(addr string) (*Client, error) {
	opt, err := redis.ParseURL(addr)
	if err != nil {
		return nil, err
	}

	rdb := redis.NewClient(opt)

	// Test connection
	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return &Client{rdb: rdb}, nil
}

// Close closes the Redis connection
func (c *Client) Close() error {
	return c.rdb.Close()
}
package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type Config struct {
	Host     string
	Port     int
	Password string
	DB       int
	PoolSize int
}

type Client struct {
	rdb *redis.Client
}

func NewClient(cfg Config) (*Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
		PoolSize: cfg.PoolSize,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return &Client{rdb: rdb}, nil
}

func (c *Client) Get(ctx context.Context, key string) (string, error) {
	return c.rdb.Get(ctx, key).Result()
}

func (c *Client) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return c.rdb.Set(ctx, key, value, expiration).Err()
}

func (c *Client) Incr(ctx context.Context, key string) (int64, error) {
	return c.rdb.Incr(ctx, key).Result()
}

func (c *Client) Expire(ctx context.Context, key string, expiration time.Duration) error {
	return c.rdb.Expire(ctx, key, expiration).Err()
}

func (c *Client) TTL(ctx context.Context, key string) (time.Duration, error) {
	return c.rdb.TTL(ctx, key).Result()
}

func (c *Client) ZAdd(ctx context.Context, key string, score float64, member string) error {
	return c.rdb.ZAdd(ctx, key, redis.Z{Score: score, Member: member}).Err()
}

func (c *Client) ZRemRangeByScore(ctx context.Context, key string, min, max string) error {
	return c.rdb.ZRemRangeByScore(ctx, key, min, max).Err()
}

func (c *Client) ZCard(ctx context.Context, key string) (int64, error) {
	return c.rdb.ZCard(ctx, key).Result()
}

func (c *Client) Eval(ctx context.Context, script string, keys []string, args ...interface{}) (interface{}, error) {
	return c.rdb.Eval(ctx, script, keys, args...).Result()
}

func (c *Client) Close() error {
	return c.rdb.Close()
}

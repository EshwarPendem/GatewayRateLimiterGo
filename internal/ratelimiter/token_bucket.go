package ratelimiter

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/example/go-rate-limiter-gateway/internal/redis"
)

type TokenBucketConfig struct {
	Rate     int
	Capacity int
}

type TokenBucket struct {
	redis    *redis.Client
	rate     int
	capacity int
}

func NewTokenBucket(redisClient *redis.Client, cfg TokenBucketConfig) *TokenBucket {
	return &TokenBucket{
		redis:    redisClient,
		rate:     cfg.Rate,
		capacity: cfg.Capacity,
	}
}

var tokenBucketScript = `
local key = KEYS[1]
local rate = tonumber(ARGV[1])
local capacity = tonumber(ARGV[2])
local now = tonumber(ARGV[3])
local requested = tonumber(ARGV[4])

local bucket = redis.call("HMGET", key, "tokens", "last_refill")
local tokens = tonumber(bucket[1])
local last_refill = tonumber(bucket[2])

if tokens == nil then
    tokens = capacity
    last_refill = now
end

local elapsed = now - last_refill
local refill = math.floor(elapsed * rate / 1000)
tokens = math.min(capacity, tokens + refill)

if refill > 0 then
    last_refill = now
end

local allowed = 0
if tokens >= requested then
    tokens = tokens - requested
    allowed = 1
end

redis.call("HMSET", key, "tokens", tokens, "last_refill", last_refill)
redis.call("EXPIRE", key, math.ceil(capacity / rate) + 1)

return allowed
`

func (tb *TokenBucket) Allow(ctx context.Context, key string) (bool, error) {
	now := time.Now().UnixMilli()
	bucketKey := fmt.Sprintf("ratelimit:tb:%s", key)

	result, err := tb.redis.Eval(ctx, tokenBucketScript, []string{bucketKey},
		tb.rate, tb.capacity, now, 1)
	if err != nil {
		return false, fmt.Errorf("token bucket eval failed: %w", err)
	}

	allowed, err := strconv.ParseInt(fmt.Sprintf("%v", result), 10, 64)
	if err != nil {
		return false, fmt.Errorf("failed to parse result: %w", err)
	}

	return allowed == 1, nil
}

func (tb *TokenBucket) Reset(ctx context.Context, key string) error {
	bucketKey := fmt.Sprintf("ratelimit:tb:%s", key)
	return tb.redis.Set(ctx, bucketKey, "", 0)
}

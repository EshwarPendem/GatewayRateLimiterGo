package ratelimiter

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/example/go-rate-limiter-gateway/internal/redis"
)

type SlidingWindowConfig struct {
	WindowSize  time.Duration
	MaxRequests int
}

type SlidingWindow struct {
	redis       *redis.Client
	windowSize  time.Duration
	maxRequests int
}

func NewSlidingWindow(redisClient *redis.Client, cfg SlidingWindowConfig) *SlidingWindow {
	return &SlidingWindow{
		redis:       redisClient,
		windowSize:  cfg.WindowSize,
		maxRequests: cfg.MaxRequests,
	}
}

var slidingWindowScript = `
local key = KEYS[1]
local window_size = tonumber(ARGV[1])
local max_requests = tonumber(ARGV[2])
local now = tonumber(ARGV[3])
local window_start = now - window_size

redis.call("ZREMRANGEBYSCORE", key, "-inf", window_start)

local count = redis.call("ZCARD", key)

if count < max_requests then
    redis.call("ZADD", key, now, now .. "-" .. math.random())
    redis.call("EXPIRE", key, math.ceil(window_size / 1000) + 1)
    return 1
end

return 0
`

func (sw *SlidingWindow) Allow(ctx context.Context, key string) (bool, error) {
	now := time.Now().UnixMilli()
	windowKey := fmt.Sprintf("ratelimit:sw:%s", key)
	windowSizeMs := sw.windowSize.Milliseconds()

	result, err := sw.redis.Eval(ctx, slidingWindowScript, []string{windowKey},
		windowSizeMs, sw.maxRequests, now)
	if err != nil {
		return false, fmt.Errorf("sliding window eval failed: %w", err)
	}

	allowed, err := strconv.ParseInt(fmt.Sprintf("%v", result), 10, 64)
	if err != nil {
		return false, fmt.Errorf("failed to parse result: %w", err)
	}

	return allowed == 1, nil
}

func (sw *SlidingWindow) Reset(ctx context.Context, key string) error {
	windowKey := fmt.Sprintf("ratelimit:sw:%s", key)
	return sw.redis.Set(ctx, windowKey, "", 0)
}

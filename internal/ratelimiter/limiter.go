package ratelimiter

import (
	"context"
)

type RateLimiter interface {
	Allow(ctx context.Context, key string) (bool, error)
	Reset(ctx context.Context, key string) error
}

type Algorithm string

const (
	TokenBucketAlgorithm   Algorithm = "token_bucket"
	SlidingWindowAlgorithm Algorithm = "sliding_window"
)

package ratelimiter

import (
	"context"
	"rate-limiter/types"
	"time"

	"github.com/redis/go-redis/v9"
)

// Absoutely simplest case: No limiting.
type PermissiveRateLimiter struct {
	redisClient *redis.Client
}

func NewPermissiveRateLimiter(redisClient *redis.Client) RateLimiter {
	//  Initialize the limiter - we don't actually use the client
	return &PermissiveRateLimiter{
		redisClient: redisClient, // So this can be NIL
	}
}

// CheckLimit implements the RateLimiter interface
func (c *PermissiveRateLimiter) CheckLimit(ctx context.Context, accountID int64, path string) (*types.RateLimitResult, error) {
	result := &types.RateLimitResult{
		ResetTime:  time.Now(),
		RetryAfter: -1,
		Allowed:    true,
		Limit:      -1,
		Remaining:  -1,
	}
	return result, nil
}

// Close gracefully shuts down the rate limiter
func (c *PermissiveRateLimiter) Close() error {
	// This stores nothing local
	// So No-Op.
	return nil
}

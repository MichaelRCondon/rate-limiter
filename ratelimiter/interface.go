package ratelimiter

import (
	"context"
	"rate-limiter/types"
)

type RateLimiter interface {
	// Interface all rate-limiter algorithms should conform to
	//	AccountID: Associate limit w. acctid, unique
	//  Path: Path targeted on back-end by request
	//  returns LimitResult: decision, or err if unable to process
	CheckLimit(ctx context.Context, accountId int64, path string) (*types.RateLimitResult, error)

	// Graceful shutdown handler
	// This needs to close DB connection handles, flush pending reqs, etc
	Close() error
}

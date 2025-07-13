package ratelimiter

import (
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type Algorithm string

// Current, defined and implemented typesafe list of rate-limiting algorithms
const ( // Play a sad kazoo 'my heart will go on' for the enums here
	Permissive Algorithm = "allow_all"
	// ContinuousSlidingWindow Algorithm = "continuous_sliding_window" // True continuous sliding window - no bucketing (Higher memory pressure)
	BucketedSlidingWindow Algorithm = "bucketed_sliding_window" // Less memory pressure: 1-minute buckets (No less than 1-minute fidelity though)
)

type Constructor func(client *redis.Client, windowSize time.Duration, defaultLimit int64) RateLimiter

var algorithmConstructors = map[Algorithm]Constructor{
	Permissive:            NewPermissiveRateLimiter,
	BucketedSlidingWindow: NewBucketedSlidingWindowLimiter,
	// TODO - MOAR.
}

func NewRateLimiter(alg Algorithm, client *redis.Client, windowSize time.Duration, defaultLimit int64) (RateLimiter, error) {
	constructor, exists := algorithmConstructors[alg]
	if !exists {
		return nil, fmt.Errorf("unknown rate-limiting algorithm %s", alg)
	}

	return constructor(client, windowSize, defaultLimit), nil
}

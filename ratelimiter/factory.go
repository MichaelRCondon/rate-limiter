package ratelimiter

import (
	"fmt"
	"strings"

	"github.com/redis/go-redis/v9"
)

type Algorithm string

// Current, defined and implemented typesafe list of rate-limiting algorithms
const ( // Play a sad kazoo 'my heart will go on' for the enums here
	Permissive              Algorithm = "allow_all"
	ContinuousSlidingWindow Algorithm = "continuous_sliding_window"
)

type Constructor func(*redis.Client) RateLimiter

var algorithmConstructors = map[Algorithm]Constructor{
	Permissive:              NewPermissiveRateLimiter,
	ContinuousSlidingWindow: NewContinuousSlidingWindowLimiter,
	// TODO - MOAR.
}

func NewRateLimiter(algName string, redisClient *redis.Client) (RateLimiter, error) {
	algo := Algorithm(strings.ToLower(strings.TrimSpace(algName)))
	if algo == "" {
		algo = Permissive
	}

	constructor, exists := algorithmConstructors[algo]
	if !exists {
		return nil, fmt.Errorf("Unknown rate-limiting algorithm %s", algName)
	}

	return constructor(redisClient), nil
}

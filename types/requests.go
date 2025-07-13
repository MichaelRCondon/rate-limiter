package types

import (
	"time"
)

// RateLimitResult represents the result of a rate limit check
type RateLimitResult struct {
	Allowed    bool          // Proceed or not
	Limit      int64         // configured cap
	remaining  int64         // remaining in-window for current user
	ResetTime  time.Time     // Window expiration time (not always useful - sliding window?)
	RetryAfter time.Duration // GO AWAY until...`
}

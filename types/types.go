// types/types.go
package types

import (
	"time"
)

// RateLimitEntry represents a custom rate limit stored in MongoDB
type RateLimitEntry struct {
	AccountID   int64         `json:"account_id"`
	Path        string        `json:"path"`
	LimitCount  int64         `json:"limit_count"`
	TimePeriod  time.Duration `json:"time_period"`
	LastUpdated time.Time     `json:"last_updated"`
}

// EndpointConfig represents configuration for a specific endpoint
type EndpointConfig struct {
	Path        string        `json:"path"`
	LimitCount  int64         `json:"limit_count"`
	TimePeriod  time.Duration `json:"time_period"`
	IsBlacklist bool          `json:"is_blacklist"`
	IsWhitelist bool          `json:"is_whitelist"`
}

// RateLimitRequest represents a request to check rate limiting
type RateLimitRequest struct {
	AccountID   int64         `json:"account_id"`
	RequestPath string        `json:"request_path"`
	Limit       int64         `json:"limit"`
	Period      time.Duration `json:"period"`
}

// RateLimitResult represents the result of a rate limit check
type RateLimitResult struct {
	Allowed    bool          // Proceed or not
	Limit      int64         // configured cap
	Remaining  int64         // remaining in-window for current user
	ResetTime  time.Time     // Window expiration time (not always useful - sliding window?)
	RetryAfter time.Duration // GO AWAY until...`
}

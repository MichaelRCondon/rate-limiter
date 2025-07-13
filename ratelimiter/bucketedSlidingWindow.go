package ratelimiter

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"os"
	"rate-limiter/types"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	InfoLogger  = log.New(os.Stdout, "[BucketedWindowRateLimiter] INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	ErrorLogger = log.New(os.Stderr, "[BucketedWindowRateLimiter] ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
)

const key_delimiter string = ":"
const key_prototype string = "%s:%s:%d:%s:%d" // Key structure for consistency
const (
	idx_prefix          int = 0
	idx_algorithm       int = 1
	idx_accountId       int = 2
	idx_path            int = 3
	idx_bucketTimestamp int = 4
)

type BucketedSlidingWindowRateLimiter struct {
	client            *redis.Client
	windowSize        time.Duration
	bucketWidth       time.Duration
	bucketCount       int
	DefaultlimitCount int64
	algorithm         string
	keyPrefix         string
}

func NewBucketedSlidingWindowLimiter(redClient *redis.Client, windowSize time.Duration, defaultLimit int64) RateLimiter {
	// TODO - allow passing of better configs
	bucketCount := 30
	keyPrefix := "rlbuk" //'rate limiting bucket'

	if bucketCount <= 0 {
		ErrorLogger.Panic("Invalid bucketing configuration supplied - Window Size: %v, Bucket Count: %v", windowSize, bucketCount)
	}
	bucketWidth := windowSize / time.Duration(bucketCount)
	if bucketWidth <= 0 {
		ErrorLogger.Panic("Invalid bucketing configuration supplied - Window Size: %v, Bucket Count: %v, Bucket Width: %v", windowSize, bucketCount, bucketWidth)
	}

	return &BucketedSlidingWindowRateLimiter{
		client:            redClient,
		windowSize:        windowSize,
		bucketWidth:       bucketWidth,
		bucketCount:       bucketCount,
		DefaultlimitCount: defaultLimit,
		algorithm:         "bucketed", // Used in key construction
		keyPrefix:         keyPrefix,
	}

}

func (rateLimiter *BucketedSlidingWindowRateLimiter) getBucketKey(accountId int64, path string, bucketId int64) string {
	return fmt.Sprintf(key_prototype, rateLimiter.keyPrefix, rateLimiter.algorithm, accountId, path, bucketId)
}

// CheckLimit implements the RateLimiter interface
func (rateLimiter *BucketedSlidingWindowRateLimiter) CheckLimit(ctx context.Context, accountID int64, path string) (*types.RateLimitResult, error) {
	// Find the bucket - integer division
	now := time.Now() // Request incoming time
	currentBucketId := now.Unix() / int64(rateLimiter.bucketWidth.Seconds())

	// get window boundaries
	windowStart := now.Add(-1 * rateLimiter.windowSize) //  subtract window from now
	windowStartId := windowStart.Unix() / int64(rateLimiter.bucketWidth.Seconds())

	currentBucketKey := rateLimiter.getBucketKey(accountID, path, currentBucketId)

	bucketKeys := rateLimiter.getBucketsInWindow(accountID, path, windowStartId, currentBucketId)

	incrPipe := rateLimiter.client.Pipeline() // Make this atomic

	incrCmd := incrPipe.Incr(ctx, currentBucketKey) // increment the current bucket - do this *first*, then separately re-load the count to catch racing requests

	_, err := incrPipe.Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("Unable to check rate limits for account %d, key %s: %v", accountID, currentBucketKey, err)
	}

	if incrCmd.Val() == 1 {
		// This is a new bucket - set expiry
		// There's a very small chance for racing expirys here, but it doesn't make a functional difference in outcome

		expiryTime := rateLimiter.windowSize + rateLimiter.bucketWidth // At least one bucket wider than window width
		err := rateLimiter.client.Expire(ctx, currentBucketKey, expiryTime).Err()
		if err != nil {
			return nil, fmt.Errorf("Unable to set bucket expiration time for bucket %s: %v", currentBucketKey, err)
		}
	}

	// Now get the sliding window count
	countPipe := rateLimiter.client.Pipeline() // Make this atomic

	var getCountCmds []*redis.StringCmd
	for _, bucketKey := range bucketKeys {
		cmd := countPipe.Get(ctx, bucketKey)
		getCountCmds = append(getCountCmds, cmd)
	}

	// Execute the commands to load bucketCmds with results
	_, err = countPipe.Exec(ctx)
	if err != nil && !errors.Is(err, redis.Nil) {
		return nil, fmt.Errorf("Unable to read buckets for AccountID: %d, Path: %s - %w", accountID, path, err)
	}

	totalCount := rateLimiter.calculateSlidingWindowCount(bucketKeys, getCountCmds, windowStart)
	resetTime := time.Unix(currentBucketId, 0).Add(rateLimiter.bucketWidth) // End of current bucket
	remainingInWindowCount := rateLimiter.DefaultlimitCount - totalCount    // This *can* be negative, since checking increments the counter. This punishes spammers who don't back off
	retryAfter := resetTime.Sub(now)
	allowed := true
	if totalCount >= rateLimiter.DefaultlimitCount { // TODO - need to load account-specific limit from Redis
		InfoLogger.Printf("Limited request for AccountID: %d, Path: %s", accountID, path)
		allowed = false
	}
	return &types.RateLimitResult{
		Allowed:    allowed,
		Limit:      rateLimiter.DefaultlimitCount,
		Remaining:  remainingInWindowCount,
		ResetTime:  resetTime,
		RetryAfter: retryAfter,
	}, nil

}

// Iterate all the buckets between start and now, build keys for each, return []string
func (rateLimiter *BucketedSlidingWindowRateLimiter) getBucketsInWindow(accountId int64, path string, windowStartId, windowEndId int64) []string {
	var bucketKeys []string

	for bucketId := windowStartId; bucketId <= windowEndId; bucketId++ {
		bucketKey := rateLimiter.getBucketKey(accountId, path, bucketId)
		bucketKeys = append(bucketKeys, bucketKey)
	}
	return bucketKeys
}

// Close gracefully shuts down the rate limiter
func (c *BucketedSlidingWindowRateLimiter) Close() error {
	// This stores nothing local
	// So No-Op.
	return nil
}

func (rateLimiter *BucketedSlidingWindowRateLimiter) calculateSlidingWindowCount(
	bucketKeys []string,
	bucketCmds []*redis.StringCmd,
	windowStart time.Time,
) int64 {
	// We're assuming that requests are distributed evenly across a single bucket.
	// If, eg, there are 5 buckets covering 5 minutes, each bucket holds a 1-minute slice
	// If we're 30 seconds into the current minute the border bucket - the oldest bucket - may cover outside the current window:
	// 	Calculate percentage of oldest bucket which is in-window, and weight it's count by that much - all other buckets contribute 100%

	totalCount := 0.0

	for i, bucketKey := range bucketKeys {
		var bucketCount int64 = 0
		if cmdBucketCountStr, err := bucketCmds[i].Result(); err != nil {
			if errors.Is(err, redis.Nil) {
				bucketCount = 0 // non-existant bucket
			} else {
				ErrorLogger.Printf("Invalid value in Bucket %v: %v", bucketKey, err)
				continue
			}
		} else {
			cmdBucketCount, err := strconv.ParseInt(cmdBucketCountStr, 10, 64)
			if err != nil {
				ErrorLogger.Printf("Non-parsable value in Bucket %v - %v: %v", bucketKey, cmdBucketCountStr, err)
				bucketCount = 0
			} else {
				bucketCount = cmdBucketCount
			}
		}

		keyParts := strings.Split(bucketKey, key_delimiter)
		bucketIdKey := keyParts[idx_bucketTimestamp] // Time is the last field in the key
		bucketIdInt, err := strconv.ParseInt(bucketIdKey, 10, 64)
		if err != nil {
			// Log an error and move next - don't get stuck on invalid data
			ErrorLogger.Printf("Invalid timestamp key in Bucket %v: %v", bucketKey, err)
			continue
		}
		bucketStartTime := time.Unix(bucketIdInt*int64(rateLimiter.bucketWidth.Seconds()), 0)
		bucketLatestTime := bucketStartTime.Add(rateLimiter.bucketWidth)

		if bucketLatestTime.Before(windowStart.Add(5 * time.Millisecond)) {
			// This means we somehow pulled a bucket that's outside our window entirely - ignore it, but log it.
			// There's a <narrow> window where this can happen benignly, but it's like 1ms. 5ms buffer should be enough we
			// never see this - otherwise it's unexpected / indicates an error in
			// the parsing
			ErrorLogger.Printf("Bucket key %s produced timestamp %s earlier than earliest in-window %v", bucketKey, bucketStartTime, windowStart)
			continue
		}

		overlapFactor := 1.0

		if bucketStartTime.Before(windowStart) { // we're glossing over the latest bucket - it should always count 100%, even if it's not completed
			// This bucket partially overlaps our window
			overlapDuration := bucketLatestTime.Sub(windowStart)                                            // Difference between most recent bucket edge, and earliest window time
			overlapFactor = float64(overlapDuration.Seconds()) / float64(rateLimiter.bucketWidth.Seconds()) // Get a pct as seconds to adjust the bucket counter by
		}

		totalCount += float64(bucketCount) * overlapFactor
	}

	return int64(math.Ceil(totalCount)) // Round up - over-estimate rate-limiting, rather than under
}

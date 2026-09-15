package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type Limiter struct {
	client *redis.Client
}

func New(client *redis.Client) *Limiter {
	return &Limiter{client: client}
}

// Allow checks whether a key is within the rate limit (maxAttempts per window).
// In case Redis is unreachable, it fails open to prevent outage cascades.
func (l *Limiter) Allow(ctx context.Context, key string, maxAttempts int64, window time.Duration) (bool, error) {
	if l.client == nil {
		return true, nil
	}

	redisKey := fmt.Sprintf("ratelimit:%s", key)
	pipe := l.client.Pipeline()
	incr := pipe.Incr(ctx, redisKey)
	pipe.ExpireNX(ctx, redisKey, window)

	_, err := pipe.Exec(ctx)
	if err != nil {
		// Fail open on Redis errors as per system design
		return true, fmt.Errorf("rate limit check failed: %w", err)
	}

	return incr.Val() <= maxAttempts, nil
}

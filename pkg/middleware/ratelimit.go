package middleware

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// rateLimitEntry tracks request count within a time window.
type rateLimitEntry struct {
	count    int
	windowAt time.Time
}

// RateLimiter implements a fixed-window rate limiter keyed by client IP.
type RateLimiter struct {
	mu      sync.Mutex
	entries map[string]*rateLimitEntry
	maxReqs int
	window  time.Duration
	cancel  context.CancelFunc
}

// NewRateLimiter creates a rate limiter that allows maxReqs requests per window per key.
// Call Stop() to release the background cleanup goroutine.
func NewRateLimiter(maxReqs int, window time.Duration) *RateLimiter {
	ctx, cancel := context.WithCancel(context.Background())
	rl := &RateLimiter{
		entries: make(map[string]*rateLimitEntry),
		maxReqs: maxReqs,
		window:  window,
		cancel:  cancel,
	}
	// Start background cleanup to prevent unbounded memory growth.
	go rl.cleanup(ctx)
	return rl
}

// Stop terminates the background cleanup goroutine.
// After Stop is called, the RateLimiter should not be used further.
func (rl *RateLimiter) Stop() {
	if rl.cancel != nil {
		rl.cancel()
	}
}

// Allow checks whether the key is within the rate limit.
// Returns true if the request is allowed, false otherwise.
func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	entry, ok := rl.entries[key]
	if !ok || now.Sub(entry.windowAt) >= rl.window {
		rl.entries[key] = &rateLimitEntry{
			count:    1,
			windowAt: now,
		}
		return true
	}
	if entry.count >= rl.maxReqs {
		return false
	}
	entry.count++
	return true
}

// cleanup periodically removes stale entries until the context is canceled.
func (rl *RateLimiter) cleanup(ctx context.Context) {
	ticker := time.NewTicker(rl.window * 2)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			rl.mu.Lock()
			now := time.Now()
			for key, entry := range rl.entries {
				if now.Sub(entry.windowAt) >= rl.window*2 {
					delete(rl.entries, key)
				}
			}
			rl.mu.Unlock()
		}
	}
}

// MidRateLimit returns a Gin middleware that rate-limits requests by client IP.
func MidRateLimit(rl *RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		if !rl.Allow(ip) {
			c.String(http.StatusTooManyRequests, "rate limit exceeded, please try again later")
			c.Abort()
			return
		}
		c.Next()
	}
}

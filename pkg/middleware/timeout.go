package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// TimeoutConfig holds configuration for the timeout middleware.
type TimeoutConfig struct {
	// Timeout is the maximum duration for request processing.
	Timeout time.Duration
	// StatusCode is the HTTP status code returned when a timeout occurs.
	StatusCode int
}

// DefaultTimeoutConfig returns sensible defaults for timeout middleware.
func DefaultTimeoutConfig() TimeoutConfig {
	return TimeoutConfig{
		Timeout:    30 * time.Second,
		StatusCode: http.StatusGatewayTimeout,
	}
}

// MidTimeout creates a timeout middleware that cancels the request context
// if processing exceeds the configured timeout. This prevents long-running
// requests from blocking resources indefinitely.
//
// The middleware uses context.WithTimeout to enforce the deadline, which
// allows downstream handlers (especially database queries) to check
// ctx.Err() and abort early when the timeout is exceeded.
func MidTimeout(cfg TimeoutConfig) gin.HandlerFunc {
	if cfg.Timeout <= 0 {
		cfg.Timeout = DefaultTimeoutConfig().Timeout
	}
	if cfg.StatusCode <= 0 {
		cfg.StatusCode = DefaultTimeoutConfig().StatusCode
	}

	return func(c *gin.Context) {
		// Create a timeout context derived from the request context
		ctx, cancel := context.WithTimeout(c.Request.Context(), cfg.Timeout)
		defer cancel()

		// Replace the request's context with the timeout context
		c.Request = c.Request.WithContext(ctx)

		// Process the request
		c.Next()

		// Check if the context was cancelled due to timeout
		if ctx.Err() == context.DeadlineExceeded {
			c.AbortWithStatusJSON(cfg.StatusCode, gin.H{
				"error":   "request timeout",
				"message": "The request took too long to process",
			})
		}
	}
}

// MidTimeoutWithDefault applies timeout middleware with default configuration.
func MidTimeoutWithDefault() gin.HandlerFunc {
	return MidTimeout(DefaultTimeoutConfig())
}

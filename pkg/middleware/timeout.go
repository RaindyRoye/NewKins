package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// MidRequestTimeout returns a Gin middleware that sets a deadline on each
// request's context. If the request takes longer than the specified timeout,
// the context is canceled and downstream handlers can check ctx.Done() to
// abort early. This prevents slow or stalled requests from exhausting
// server resources (goroutines, database connections, etc.).
//
// The middleware returns 504 Gateway Timeout when the deadline expires before
// the handler finishes writing a response.
func MidRequestTimeout(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()

		// Replace the request with the timeout-scoped context.
		c.Request = c.Request.WithContext(ctx)

		// Use a channel to detect when the handler has finished.
		done := make(chan struct{})
		go func() {
			c.Next()
			close(done)
		}()

		select {
		case <-done:
			// Handler completed normally.
		case <-ctx.Done():
			// Context deadline exceeded. If no status has been written yet,
			// return a 504 Gateway Timeout. If the handler already started
			// writing, we can't override the response.
			if !c.Writer.Written() {
				c.AbortWithStatusJSON(http.StatusGatewayTimeout, gin.H{
					"error": "request timeout",
				})
			}
		}
	}
}

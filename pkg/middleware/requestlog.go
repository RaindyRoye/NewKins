package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// MidRequestLog returns a Gin middleware that logs request duration and status.
// It uses the request ID from MidRequestID for correlation.
// Slow requests (> 1 second) are logged at Warn level.
func MidRequestLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method

		// Process request
		c.Next()

		// Calculate duration
		duration := time.Since(start)
		status := c.Writer.Status()
		reqID := GetRequestID(c)

		// Build log entry
		entry := logrus.WithFields(logrus.Fields{
			"method":   method,
			"path":     path,
			"status":   status,
			"duration": duration.String(),
			"client":   c.ClientIP(),
		})
		if reqID != "" {
			entry = entry.WithField("request_id", reqID)
		}

		// Log at appropriate level
		if status >= 500 {
			entry.Error("request failed")
		} else if duration > time.Second {
			entry.Warn("slow request")
		} else if status >= 400 {
			entry.Info("client error")
		} else {
			entry.Debug("request completed")
		}
	}
}

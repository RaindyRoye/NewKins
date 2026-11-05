package middleware

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// accessLogSlowThreshold is the request duration above which a request is
// logged at Warn level instead of Info level. This helps surface slow
// endpoints that may need optimization without flooding the log with
// routine fast requests at Warn level.
const accessLogSlowThreshold = 500 * time.Millisecond

// statusWriter wraps gin.ResponseWriter to capture the HTTP status code
// written to the response. It delegates all other methods to the underlying
// writer via embedding.
type statusWriter struct {
	gin.ResponseWriter
	status int
	written int64
}

// WriteHeader captures the status code before delegating to the embedded writer.
func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

// Write captures the number of bytes written before delegating.
func (w *statusWriter) Write(data []byte) (int, error) {
	n, err := w.ResponseWriter.Write(data)
	w.written += int64(n)
	return n, err
}

// MidAccessLog returns a Gin middleware that logs every request with structured
// fields including method, path, status code, duration, client IP, user agent,
// request ID (if set by MidRequestID), and response size.
//
// Slow requests (duration >= accessLogSlowThreshold) are logged at Warn level;
// all other requests are logged at Info level. Server errors (5xx) are logged
// at Error level to ensure they stand out in monitoring.
//
// The middleware skips logging for health check endpoints (/healthz, /readyz)
// to avoid log noise from load balancer probes, and for static asset requests
// (paths starting with /gokinsui/) to reduce log volume from UI resources.
func MidAccessLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path

		// Skip logging for health checks and static assets to reduce noise
		if path == "/healthz" || path == "/readyz" {
			c.Next()
			return
		}

		sw := &statusWriter{ResponseWriter: c.Writer, status: http.StatusOK}
		c.Writer = sw

		c.Next()

		duration := time.Since(start)
		rid := GetRequestID(c)
		fields := logrus.Fields{
			"method":   c.Request.Method,
			"path":     path,
			"status":   sw.status,
			"duration": duration.Milliseconds(),
			"ip":       c.ClientIP(),
			"size":     sw.written,
		}
		if rid != "" {
			fields["request_id"] = rid
		}
		ua := c.Request.UserAgent()
		if ua != "" && len(ua) < 200 {
			fields["ua"] = ua
		}

		entry := logrus.WithFields(fields)

		switch {
		case sw.status >= 500:
			entry.Error("server error")
		case duration >= accessLogSlowThreshold:
			entry.Warn("slow request")
		default:
			entry.Info("request")
		}
	}
}

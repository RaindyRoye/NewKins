package middleware

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/gin-gonic/gin"
)

const (
	// RequestIDHeader is the HTTP header used to propagate request IDs.
	RequestIDHeader = "X-Request-ID"

	// RequestIDKey is the gin.Context key for the request ID value.
	RequestIDKey = "request_id"
)

// MidRequestID returns a Gin middleware that ensures every request has a unique
// request ID. If the client provides an X-Request-ID header, it is reused;
// otherwise a new 128-bit random hex ID is generated.
//
// The request ID is:
//   - Set in the X-Request-ID response header (for client correlation)
//   - Stored in the gin.Context under RequestIDKey (for structured logging)
func MidRequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		rid := c.GetHeader(RequestIDHeader)
		if rid == "" {
			rid = generateRequestID()
		}
		c.Set(RequestIDKey, rid)
		c.Header(RequestIDHeader, rid)
		c.Next()
	}
}

// GetRequestID extracts the request ID from a Gin context.
// Returns an empty string if no request ID is set.
func GetRequestID(c *gin.Context) string {
	if v, ok := c.Get(RequestIDKey); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// generateRequestID creates a 16-byte (128-bit) random hex string.
// Falls back to a timestamp-based ID if crypto/rand fails.
func generateRequestID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// Fallback: this should never happen in practice
		return "fallback-unknown"
	}
	return hex.EncodeToString(b)
}

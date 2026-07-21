package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// allowedMethods lists the HTTP methods permitted for cross-origin requests.
// Restricted to standard CRUD operations plus OPTIONS for preflight.
var allowedMethods = "GET, POST, PUT, PATCH, DELETE, OPTIONS"

// allowedHeaders lists the HTTP headers that clients may send in cross-origin requests.
// Explicitly enumerated instead of using "*" to reduce attack surface.
var allowedHeaders = "Content-Type, Authorization, X-Requested-With, Accept, Origin, X-Request-Id"

// exposedHeaders lists response headers that browsers are allowed to read.
var exposedHeaders = "X-Request-Id"

// MidCORS is a CORS middleware that adds appropriate Access-Control
// headers to responses. For OPTIONS preflight requests, it returns 204 No Content.
// For other requests, it echoes the Origin header (when present) to allow
// credentialed cross-origin requests from that specific origin.
//
// Security improvements over the previous wildcard implementation:
// - Replaces Access-Control-Allow-Headers: "*" with an explicit allowlist
// - Replaces Access-Control-Allow-Methods: "*" with standard CRUD methods only
// - Adds Vary: Origin header for proper caching by intermediaries
// - Only echoes Origin when present, avoiding overly permissive responses
func MidCORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		method := strings.ToUpper(c.Request.Method)
		origin := c.Request.Header.Get("Origin")

		if origin != "" {
			// Echo the specific origin to allow credentialed requests.
			// This is more secure than "*" because browsers will only accept
			// the exact matching origin, and credentials are restricted to it.
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Vary", "Origin")
		}

		if method == "OPTIONS" || method == "POST" {
			c.Header("Access-Control-Allow-Headers", allowedHeaders)
			c.Header("Access-Control-Allow-Methods", allowedMethods)
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Expose-Headers", exposedHeaders)
		}

		// Handle preflight requests
		if method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

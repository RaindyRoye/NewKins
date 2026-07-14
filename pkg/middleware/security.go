package middleware

import (
	"github.com/gin-gonic/gin"
)

// MidSecurityHeaders returns a Gin middleware that sets common HTTP security headers
// to protect against XSS, clickjacking, MIME sniffing, and other attacks.
func MidSecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Prevent browsers from MIME-sniffing a response away from the declared Content-Type.
		c.Header("X-Content-Type-Options", "nosniff")
		// Protect against clickjacking by disallowing the page from being embedded in frames.
		c.Header("X-Frame-Options", "DENY")
		// Enable the browser's built-in XSS filter (legacy but still useful).
		c.Header("X-XSS-Protection", "1; mode=block")
		// Control which resources the browser is allowed to load.
		c.Header("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self' data:")
		// Prevent leaking referrer information to external sites.
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		// Ensure the connection is upgraded to HTTPS when available.
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")

		c.Next()
	}
}

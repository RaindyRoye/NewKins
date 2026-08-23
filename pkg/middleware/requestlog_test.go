package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestMidRequestLog(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		method         string
		path           string
		status         int
		expectLogLevel string // "debug", "info", "warn", "error"
	}{
		{
			name:           "successful GET request",
			method:         "GET",
			path:           "/api/test",
			status:         200,
			expectLogLevel: "debug",
		},
		{
			name:           "client error 404",
			method:         "GET",
			path:           "/api/missing",
			status:         404,
			expectLogLevel: "info",
		},
		{
			name:           "server error 500",
			method:         "POST",
			path:           "/api/fail",
			status:         500,
			expectLogLevel: "error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(MidRequestID())
			router.Use(MidRequestLog())

			router.Handle(tt.method, tt.path, func(c *gin.Context) {
				c.Status(tt.status)
			})

			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.status {
				t.Errorf("expected status %d, got %d", tt.status, w.Code)
			}

			// Verify request ID was set
			reqID := w.Header().Get(RequestIDHeader)
			if reqID == "" {
				t.Error("expected request ID header to be set")
			}
		})
	}
}

func TestMidRequestLog_WithExistingRequestID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(MidRequestID())
	router.Use(MidRequestLog())

	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set(RequestIDHeader, "existing-id-123")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should preserve the existing request ID
	if w.Header().Get(RequestIDHeader) != "existing-id-123" {
		t.Errorf("expected existing request ID to be preserved")
	}
}

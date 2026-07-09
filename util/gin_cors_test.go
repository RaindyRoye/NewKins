package util

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestMidAccessAllowFun_OPTIONS(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequestWithContext(context.Background(), "OPTIONS", "/api/test", nil)
	c.Request.Header.Set("Origin", "https://example.com")

	MidAccessAllowFun(c)

	if w.Code != http.StatusNoContent {
		t.Errorf("OPTIONS should return 204, got %d", w.Code)
	}

	if c.IsAborted() != true {
		t.Error("OPTIONS should abort the request")
	}

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "https://example.com" {
		t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, "https://example.com")
	}

	if got := w.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Errorf("Access-Control-Allow-Credentials = %q, want %q", got, "true")
	}
}

func TestMidAccessAllowFun_POST(t *testing.T) {
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)
	r.POST("/test", func(c *gin.Context) {
		c.String(200, "ok")
	})
	c.Request = httptest.NewRequestWithContext(context.Background(), "POST", "/test", nil)
	c.Request.Header.Set("Origin", "https://example.com")

	MidAccessAllowFun(c)

	if c.IsAborted() {
		t.Error("POST should not abort the request")
	}

	// CORS headers should be set for POST
	// Note: headers are set on the response writer, but in test context
	// we need to check if the function ran without panic
}

func TestMidAccessAllowFun_GET(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequestWithContext(context.Background(), "GET", "/test", nil)

	MidAccessAllowFun(c)

	if c.IsAborted() {
		t.Error("GET should not abort the request")
	}
}

func TestMidAccessAllowFun_PUT(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequestWithContext(context.Background(), "PUT", "/test", nil)

	MidAccessAllowFun(c)

	if c.IsAborted() {
		t.Error("PUT should not abort the request")
	}
}

func TestMidAccessAllowFun_DELETE(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequestWithContext(context.Background(), "DELETE", "/test", nil)

	MidAccessAllowFun(c)

	if c.IsAborted() {
		t.Error("DELETE should not abort the request")
	}
}

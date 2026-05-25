package util

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestRateLimiter_Allow(t *testing.T) {
	rl := NewRateLimiter(3, time.Minute)

	key := "192.168.1.1"
	// First 3 requests should be allowed
	for i := 0; i < 3; i++ {
		if !rl.Allow(key) {
			t.Errorf("request %d should be allowed", i+1)
		}
	}
	// 4th request should be denied
	if rl.Allow(key) {
		t.Error("4th request should be denied")
	}
}

func TestRateLimiter_WindowReset(t *testing.T) {
	rl := NewRateLimiter(2, 50*time.Millisecond)

	key := "10.0.0.1"
	if !rl.Allow(key) {
		t.Error("first request should be allowed")
	}
	if !rl.Allow(key) {
		t.Error("second request should be allowed")
	}
	if rl.Allow(key) {
		t.Error("third request should be denied")
	}

	// Wait for window to expire
	time.Sleep(60 * time.Millisecond)

	// Should be allowed again
	if !rl.Allow(key) {
		t.Error("request after window reset should be allowed")
	}
}

func TestRateLimiter_DifferentKeys(t *testing.T) {
	rl := NewRateLimiter(1, time.Minute)

	if !rl.Allow("ip1") {
		t.Error("first request from ip1 should be allowed")
	}
	if !rl.Allow("ip2") {
		t.Error("first request from ip2 should be allowed")
	}
	// ip1 should be denied
	if rl.Allow("ip1") {
		t.Error("second request from ip1 should be denied")
	}
	// ip2 should also be denied
	if rl.Allow("ip2") {
		t.Error("second request from ip2 should be denied")
	}
}

func TestMidRateLimit_GinMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rl := NewRateLimiter(2, time.Minute)

	g := gin.New()
	g.Use(MidRateLimit(rl))
	g.GET("/test", func(c *gin.Context) {
		c.String(200, "ok")
	})

	// First 2 requests should succeed
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "192.168.1.100:1234"
		w := httptest.NewRecorder()
		g.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("request %d: expected 200, got %d", i+1, w.Code)
		}
	}

	// 3rd request should be rate-limited
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.168.1.100:1234"
	w := httptest.NewRecorder()
	g.ServeHTTP(w, req)
	if w.Code != http.StatusTooManyRequests {
		t.Errorf("3rd request: expected 429, got %d", w.Code)
	}
}

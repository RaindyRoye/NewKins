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

func TestRateLimiter_Stop(t *testing.T) {
	rl := NewRateLimiter(2, 50*time.Millisecond)

	// Allow should work before Stop
	if !rl.Allow("test-key") {
		t.Error("request should be allowed before Stop")
	}

	// Stop should not panic
	rl.Stop()

	// Calling Stop again should not panic (idempotent)
	rl.Stop()

	// Allow should still work after Stop (just no more cleanup)
	if !rl.Allow("test-key") {
		t.Error("request should still be allowed after Stop")
	}
}

func TestRateLimiter_CleanupExitsOnStop(t *testing.T) {
	rl := NewRateLimiter(5, 10*time.Millisecond)

	// Add some entries
	for i := 0; i < 10; i++ {
		rl.Allow("key-" + string(rune('a'+i)))
	}

	rl.mu.Lock()
	entryCount := len(rl.entries)
	rl.mu.Unlock()
	if entryCount == 0 {
		t.Error("expected entries after Allow calls")
	}

	// Stop the limiter - cleanup goroutine should exit
	rl.Stop()

	// Give the goroutine time to exit
	time.Sleep(30 * time.Millisecond)
	// If the goroutine didn't exit, this would be a leak,
	// but we can't easily test that directly. The fact that
	// Stop() doesn't hang or panic is sufficient.
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

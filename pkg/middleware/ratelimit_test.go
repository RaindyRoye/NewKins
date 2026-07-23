package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestNewRateLimiter(t *testing.T) {
	rl := NewRateLimiter(5, time.Minute)
	defer rl.Stop()

	if rl == nil {
		t.Fatal("expected non-nil RateLimiter")
	}
	if rl.maxReqs != 5 {
		t.Errorf("expected maxReqs=5, got %d", rl.maxReqs)
	}
}

func TestRateLimiter_Allow(t *testing.T) {
	rl := NewRateLimiter(3, time.Second)
	defer rl.Stop()

	key := "192.168.1.1"

	// First 3 requests should be allowed
	for i := 0; i < 3; i++ {
		if !rl.Allow(key) {
			t.Errorf("request %d should be allowed", i+1)
		}
	}

	// 4th request should be denied
	if rl.Allow(key) {
		t.Error("4th request should be denied (rate limited)")
	}

	// Different key should still be allowed
	if !rl.Allow("192.168.1.2") {
		t.Error("different key should be allowed")
	}
}

func TestRateLimiter_WindowExpiry(t *testing.T) {
	rl := NewRateLimiter(2, 100*time.Millisecond)
	defer rl.Stop()

	key := "10.0.0.1"

	// Exhaust the limit
	rl.Allow(key)
	rl.Allow(key)
	if rl.Allow(key) {
		t.Error("should be rate limited")
	}

	// Wait for window to expire
	time.Sleep(150 * time.Millisecond)

	// Should be allowed again
	if !rl.Allow(key) {
		t.Error("should be allowed after window expires")
	}
}

func TestMidRateLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rl := NewRateLimiter(2, time.Minute)
	defer rl.Stop()

	router := gin.New()
	router.Use(MidRateLimit(rl))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	// First 2 requests should succeed
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/test", nil) //nolint:noctx // httptest.NewRequest without context is acceptable in tests
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("request %d: expected status 200, got %d", i+1, w.Code)
		}
	}

	// 3rd request should be rate limited
	req := httptest.NewRequest("GET", "/test", nil) //nolint:noctx // httptest.NewRequest without context is acceptable in tests
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("expected status 429, got %d", w.Code)
	}
}

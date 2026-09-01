package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestMidTimeout_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(MidTimeout(TimeoutConfig{
		Timeout:    1 * time.Second,
		StatusCode: http.StatusGatewayTimeout,
	}))

	router.GET("/fast", func(c *gin.Context) {
		time.Sleep(100 * time.Millisecond)
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/fast", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestMidTimeout_Exceeded(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(MidTimeout(TimeoutConfig{
		Timeout:    100 * time.Millisecond,
		StatusCode: http.StatusGatewayTimeout,
	}))

	router.GET("/slow", func(c *gin.Context) {
		// Sleep longer than the timeout
		select {
		case <-time.After(500 * time.Millisecond):
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		case <-c.Request.Context().Done():
			// Context was cancelled due to timeout
			return
		}
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/slow", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusGatewayTimeout {
		t.Errorf("Expected status %d, got %d", http.StatusGatewayTimeout, w.Code)
	}
}

func TestMidTimeout_ContextCancelled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(MidTimeout(TimeoutConfig{
		Timeout:    50 * time.Millisecond,
		StatusCode: http.StatusGatewayTimeout,
	}))

	handlerCalled := make(chan bool, 1)
	router.GET("/check", func(c *gin.Context) {
		handlerCalled <- true
		// Check if context gets cancelled
		select {
		case <-time.After(200 * time.Millisecond):
			t.Error("Handler should have been cancelled")
		case <-c.Request.Context().Done():
			// Expected: context cancelled
		}
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/check", nil)
	router.ServeHTTP(w, req)

	select {
	case <-handlerCalled:
		// Handler was called
	case <-time.After(100 * time.Millisecond):
		t.Error("Handler was not called")
	}
}

func TestMidTimeoutWithDefault(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(MidTimeoutWithDefault())

	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestMidTimeout_InvalidConfig(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	// Pass invalid config (zero timeout), should use defaults
	router.Use(MidTimeout(TimeoutConfig{
		Timeout:    0,
		StatusCode: 0,
	}))

	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
}

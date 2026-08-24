package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestMidRequestTimeout_NormalCompletion(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(MidRequestTimeout(100 * time.Millisecond))
	router.GET("/fast", func(c *gin.Context) {
		// Fast handler that completes before timeout
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/fast", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	if w.Body.String() != "ok" {
		t.Errorf("expected body 'ok', got %q", w.Body.String())
	}
}

func TestMidRequestTimeout_TimeoutExceeded(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(MidRequestTimeout(50 * time.Millisecond))
	router.GET("/slow", func(c *gin.Context) {
		// Slow handler that exceeds timeout
		select {
		case <-time.After(200 * time.Millisecond):
			c.String(http.StatusOK, "done")
		case <-c.Request.Context().Done():
			// Context canceled, but we haven't written yet
			// The middleware will handle this
			time.Sleep(10 * time.Millisecond)
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/slow", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusGatewayTimeout {
		t.Errorf("expected status 504, got %d", w.Code)
	}
}

func TestMidRequestTimeout_ContextCancellation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(MidRequestTimeout(50 * time.Millisecond))

	contextCanceled := make(chan bool, 1)
	router.GET("/check", func(c *gin.Context) {
		// Wait for context to be canceled
		<-c.Request.Context().Done()
		contextCanceled <- true
	})

	req := httptest.NewRequest(http.MethodGet, "/check", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Verify that the context was actually canceled
	select {
	case canceled := <-contextCanceled:
		if !canceled {
			t.Error("context should have been canceled")
		}
	case <-time.After(200 * time.Millisecond):
		t.Error("timeout waiting for context cancellation")
	}
}

func TestMidRequestTimeout_ZeroTimeout(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	// Zero timeout means immediate cancellation
	router.Use(MidRequestTimeout(0))
	router.GET("/zero", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/zero", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// With zero timeout, should get 504
	if w.Code != http.StatusGatewayTimeout {
		t.Errorf("expected status 504 with zero timeout, got %d", w.Code)
	}
}

func TestMidRequestTimeout_LargeTimeout(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(MidRequestTimeout(10 * time.Second))
	router.GET("/large", func(c *gin.Context) {
		// Fast handler with large timeout
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/large", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestMidRequestTimeout_ContextPropagation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(MidRequestTimeout(100 * time.Millisecond))

	var deadlineSet bool
	router.GET("/deadline", func(c *gin.Context) {
		_, ok := c.Request.Context().Deadline()
		deadlineSet = ok
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/deadline", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if !deadlineSet {
		t.Error("context should have a deadline set by middleware")
	}
}

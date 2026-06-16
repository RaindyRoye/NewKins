package util

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestMidRequestID_GeneratesNew(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(MidRequestID())
	r.GET("/test", func(c *gin.Context) {
		rid := GetRequestID(c)
		if rid == "" {
			t.Error("expected request ID to be set in context")
		}
		c.String(200, rid)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// Response should have X-Request-ID header
	hdr := w.Header().Get(RequestIDHeader)
	if hdr == "" {
		t.Error("expected X-Request-ID in response header")
	}
	if len(hdr) != 32 { // 16 bytes hex = 32 chars
		t.Errorf("expected 32-char hex ID, got %d chars: %q", len(hdr), hdr)
	}
}

func TestMidRequestID_ReusesExisting(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(MidRequestID())
	r.GET("/test", func(c *gin.Context) {
		c.String(200, "ok")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/test", nil)
	req.Header.Set(RequestIDHeader, "my-custom-id-12345")
	r.ServeHTTP(w, req)

	hdr := w.Header().Get(RequestIDHeader)
	if hdr != "my-custom-id-12345" {
		t.Errorf("expected reused ID %q, got %q", "my-custom-id-12345", hdr)
	}
}

func TestMidRequestID_UniquePerRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(MidRequestID())
	r.GET("/test", func(c *gin.Context) {
		c.String(200, "ok")
	})

	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequestWithContext(context.Background(), "GET", "/test", nil)
		r.ServeHTTP(w, req)

		hdr := w.Header().Get(RequestIDHeader)
		if ids[hdr] {
			t.Fatalf("duplicate request ID after %d requests: %q", i, hdr)
		}
		ids[hdr] = true
	}
}

func TestGetRequestID_Missing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	rid := GetRequestID(c)
	if rid != "" {
		t.Errorf("expected empty string, got %q", rid)
	}
}

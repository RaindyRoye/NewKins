package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestMidCORS_PreflightRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(MidCORS())
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	req := httptest.NewRequest("OPTIONS", "/test", nil) //nolint:noctx // httptest.NewRequest without context is acceptable in tests
	req.Header.Set("Origin", "https://example.com")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected status 204, got %d", w.Code)
	}

	// Check CORS headers
	headers := map[string]string{
		"Access-Control-Allow-Origin":      "https://example.com",
		"Access-Control-Allow-Methods":     "GET, POST, PUT, PATCH, DELETE, OPTIONS",
		"Access-Control-Allow-Headers":     "Content-Type, Authorization, X-Requested-With, Accept, Origin, X-Request-Id",
		"Access-Control-Allow-Credentials": "true",
		"Vary":                             "Origin",
	}

	for header, expected := range headers {
		actual := w.Header().Get(header)
		if actual != expected {
			t.Errorf("header %s: expected %q, got %q", header, expected, actual)
		}
	}
}

func TestMidCORS_SimpleRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(MidCORS())
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	req := httptest.NewRequest("GET", "/test", nil) //nolint:noctx // httptest.NewRequest without context is acceptable in tests
	req.Header.Set("Origin", "https://example.com")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// Check that origin is echoed
	origin := w.Header().Get("Access-Control-Allow-Origin")
	if origin != "https://example.com" {
		t.Errorf("expected origin https://example.com, got %q", origin)
	}

	// Check Vary header
	vary := w.Header().Get("Vary")
	if vary != "Origin" {
		t.Errorf("expected Vary: Origin, got %q", vary)
	}
}

func TestMidCORS_NoOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(MidCORS())
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	req := httptest.NewRequest("GET", "/test", nil) //nolint:noctx // httptest.NewRequest without context is acceptable in tests
	// No Origin header
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// Check that no CORS origin is set when Origin is missing
	origin := w.Header().Get("Access-Control-Allow-Origin")
	if origin != "" {
		t.Errorf("expected no Access-Control-Allow-Origin header, got %q", origin)
	}
}

package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestMidRequestID_GeneratesNew(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(MidRequestID())
	router.GET("/test", func(c *gin.Context) {
		rid := GetRequestID(c)
		c.String(http.StatusOK, rid)
	})

	req := httptest.NewRequest("GET", "/test", nil) //nolint:noctx // httptest.NewRequest without context is acceptable in tests
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// Check that a request ID was generated
	rid := w.Body.String()
	if rid == "" || rid == "fallback-unknown" {
		t.Errorf("expected valid request ID, got %q", rid)
	}

	// Check that the header is set
	responseRid := w.Header().Get(RequestIDHeader)
	if responseRid != rid {
		t.Errorf("response header %s: expected %q, got %q", RequestIDHeader, rid, responseRid)
	}
}

func TestMidRequestID_ReusesProvided(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(MidRequestID())
	router.GET("/test", func(c *gin.Context) {
		rid := GetRequestID(c)
		c.String(http.StatusOK, rid)
	})

	providedID := "custom-request-id-12345"
	req := httptest.NewRequest("GET", "/test", nil) //nolint:noctx // httptest.NewRequest without context is acceptable in tests
	req.Header.Set(RequestIDHeader, providedID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	rid := w.Body.String()
	if rid != providedID {
		t.Errorf("expected request ID %q, got %q", providedID, rid)
	}

	responseRid := w.Header().Get(RequestIDHeader)
	if responseRid != providedID {
		t.Errorf("response header: expected %q, got %q", providedID, responseRid)
	}
}

func TestGetRequestID_Missing(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.GET("/test", func(c *gin.Context) {
		rid := GetRequestID(c)
		c.String(http.StatusOK, rid)
	})

	req := httptest.NewRequest("GET", "/test", nil) //nolint:noctx // httptest.NewRequest without context is acceptable in tests
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	rid := w.Body.String()
	if rid != "" {
		t.Errorf("expected empty request ID, got %q", rid)
	}
}

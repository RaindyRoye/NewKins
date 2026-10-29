package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gokins/gokins/comm"
)

// TestMidUiHandle_NotInstalled_404Redirects tests that 404 responses are redirected to /install
func TestMidUiHandle_NotInstalled_404Redirects(t *testing.T) {
	gin.SetMode(gin.TestMode)

	origInstalled := comm.Installed
	comm.Installed = false
	defer func() { comm.Installed = origInstalled }()

	router := gin.New()
	router.Use(midUiHandle)
	router.GET("/api/test", func(c *gin.Context) {
		c.String(http.StatusNotFound, "not found")
	})

	req := httptest.NewRequest("GET", "/api/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should redirect to /install with 302
	if w.Code != http.StatusFound {
		t.Errorf("expected status 302, got %d", w.Code)
	}
	if location := w.Header().Get("Location"); location != "/install" {
		t.Errorf("expected redirect to /install, got %q", location)
	}
}

// TestMidUiHandle_NotInstalled_200NoRedirect tests that 200 responses are not redirected
func TestMidUiHandle_NotInstalled_200NoRedirect(t *testing.T) {
	gin.SetMode(gin.TestMode)

	origInstalled := comm.Installed
	comm.Installed = false
	defer func() { comm.Installed = origInstalled }()

	router := gin.New()
	router.Use(midUiHandle)
	router.GET("/api/test", func(c *gin.Context) {
		c.String(http.StatusOK, "success")
	})

	req := httptest.NewRequest("GET", "/api/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should not redirect, should return 200
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	if w.Body.String() != "success" {
		t.Errorf("expected 'success', got %q", w.Body.String())
	}
}

// TestMidUiHandle_Installed_404NoRedirect tests that 404 responses are not redirected when installed
func TestMidUiHandle_Installed_404NoRedirect(t *testing.T) {
	gin.SetMode(gin.TestMode)

	origInstalled := comm.Installed
	comm.Installed = true
	defer func() { comm.Installed = origInstalled }()

	router := gin.New()
	router.Use(midUiHandle)
	router.GET("/api/test", func(c *gin.Context) {
		c.String(http.StatusNotFound, "not found")
	})

	req := httptest.NewRequest("GET", "/api/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should not redirect, should return 404
	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
	if w.Body.String() != "not found" {
		t.Errorf("expected 'not found', got %q", w.Body.String())
	}
}

// TestMidUiHandle_Installed_200NoRedirect tests that 200 responses pass through when installed
func TestMidUiHandle_Installed_200NoRedirect(t *testing.T) {
	gin.SetMode(gin.TestMode)

	origInstalled := comm.Installed
	comm.Installed = true
	defer func() { comm.Installed = origInstalled }()

	router := gin.New()
	router.Use(midUiHandle)
	router.GET("/api/test", func(c *gin.Context) {
		c.String(http.StatusOK, "success")
	})

	req := httptest.NewRequest("GET", "/api/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should not redirect, should return 200
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	if w.Body.String() != "success" {
		t.Errorf("expected 'success', got %q", w.Body.String())
	}
}

// TestMidUiHandle_NotInstalled_OtherStatusNoRedirect tests that non-404 errors are not redirected
func TestMidUiHandle_NotInstalled_OtherStatusNoRedirect(t *testing.T) {
	gin.SetMode(gin.TestMode)

	origInstalled := comm.Installed
	comm.Installed = false
	defer func() { comm.Installed = origInstalled }()

	testCases := []struct {
		status int
		body   string
	}{
		{http.StatusBadRequest, "bad request"},
		{http.StatusInternalServerError, "server error"},
		{http.StatusUnauthorized, "unauthorized"},
		{http.StatusForbidden, "forbidden"},
	}

	for _, tc := range testCases {
		router := gin.New()
		router.Use(midUiHandle)
		router.GET("/api/test", func(c *gin.Context) {
			c.String(tc.status, tc.body)
		})

		req := httptest.NewRequest("GET", "/api/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should not redirect, should return original status
		if w.Code != tc.status {
			t.Errorf("status %d: expected status %d, got %d", tc.status, tc.status, w.Code)
		}
		if w.Body.String() != tc.body {
			t.Errorf("status %d: expected %q, got %q", tc.status, tc.body, w.Body.String())
		}
	}
}

// TestMidUiHandle_InstallRouteNoRedirect tests that /install route itself is not redirected
func TestMidUiHandle_InstallRouteNoRedirect(t *testing.T) {
	gin.SetMode(gin.TestMode)

	origInstalled := comm.Installed
	comm.Installed = false
	defer func() { comm.Installed = origInstalled }()

	router := gin.New()
	router.Use(midUiHandle)
	router.GET("/install", func(c *gin.Context) {
		c.String(http.StatusOK, "install page")
	})

	req := httptest.NewRequest("GET", "/install", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should not redirect even when not installed
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	if w.Body.String() != "install page" {
		t.Errorf("expected 'install page', got %q", w.Body.String())
	}
}

// TestMidUiHandle_Multiple404s tests multiple 404 requests in sequence
func TestMidUiHandle_Multiple404s(t *testing.T) {
	gin.SetMode(gin.TestMode)

	origInstalled := comm.Installed
	comm.Installed = false
	defer func() { comm.Installed = origInstalled }()

	router := gin.New()
	router.Use(midUiHandle)
	router.GET("/api/test", func(c *gin.Context) {
		c.String(http.StatusNotFound, "not found")
	})

	// Make multiple requests to ensure middleware state is clean
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("GET", "/api/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusFound {
			t.Errorf("request %d: expected status 302, got %d", i, w.Code)
		}
		if location := w.Header().Get("Location"); location != "/install" {
			t.Errorf("request %d: expected redirect to /install, got %q", i, location)
		}
	}
}

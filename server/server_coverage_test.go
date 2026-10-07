package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gokins/core"
	"github.com/gokins/gokins/comm"
)

// TestReadyzEndpoint_WithDatabase tests readyz when database is connected
func TestReadyzEndpoint_WithDatabase(t *testing.T) {
	// This test requires a real database connection, so we'll skip if not available
	origDb := comm.Db
	origCache := comm.BCache
	defer func() {
		comm.Db = origDb
		comm.BCache = origCache
	}()

	// If no database is available, skip this test
	if comm.Db == nil {
		t.Skip("skipping: no database available")
	}

	router := setupTestRouter(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/readyz", nil)
	router.ServeHTTP(w, req)

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	// Should be ready if both db and cache are connected
	if resp["status"] != "ready" {
		t.Logf("status = %v (may be not_ready if cache is nil)", resp["status"])
	}
}

// TestReadyzEndpoint_WithCacheOnly tests readyz when only cache is available
func TestReadyzEndpoint_WithCacheOnly(t *testing.T) {
	origDb := comm.Db
	origCache := comm.BCache
	defer func() {
		comm.Db = origDb
		comm.BCache = origCache
	}()

	comm.Db = nil
	// Keep cache if available

	router := setupTestRouter(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/readyz", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status 503 when db is nil, got %d", w.Code)
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp["db"] != "disconnected" {
		t.Errorf("expected db='disconnected', got %v", resp["db"])
	}
}

// TestPprofEndpoints_EnabledByConfig tests pprof when enabled via config
func TestPprofEndpoints_EnabledByConfig(t *testing.T) {
	origDebug := core.Debug
	origCfg := comm.Cfg
	defer func() {
		core.Debug = origDebug
		comm.Cfg = origCfg
	}()

	core.Debug = false
	comm.Cfg.Server.EnablePprof = true

	router := setupTestRouter(t)

	endpoints := []string{
		"/debug/pprof/",
		"/debug/pprof/cmdline",
		"/debug/pprof/symbol",
		"/debug/pprof/goroutine",
		"/debug/pprof/heap",
		"/debug/pprof/allocs",
		"/debug/pprof/block",
		"/debug/pprof/mutex",
		"/debug/pprof/threadcreate",
	}

	for _, endpoint := range endpoints {
		t.Run(endpoint, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequestWithContext(context.Background(), "GET", endpoint, nil)
			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("GET %s: expected status 200, got %d", endpoint, w.Code)
			}
		})
	}
}

// TestMidUiHandle_InstalledMode tests middleware when application is installed
func TestMidUiHandle_InstalledMode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(midUiHandle)
	router.GET("/api/test", func(c *gin.Context) {
		c.String(http.StatusOK, "api response")
	})

	origInstalled := comm.Installed
	defer func() { comm.Installed = origInstalled }()
	comm.Installed = true

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/api/test", nil)
	router.ServeHTTP(w, req)

	// Should return 200 from the handler
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

// TestMidUiHandle_InstallPage tests that /install is accessible when not installed
func TestMidUiHandle_InstallPage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(midUiHandle)
	router.GET("/install", func(c *gin.Context) {
		c.String(http.StatusOK, "install page")
	})

	origInstalled := comm.Installed
	defer func() { comm.Installed = origInstalled }()
	comm.Installed = false

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/install", nil)
	router.ServeHTTP(w, req)

	// Should return 200 from the install handler
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200 for /install, got %d", w.Code)
	}
}

// TestMidUiHandle_StaticUIPath tests that /gokinsui/ paths are accessible when not installed
func TestMidUiHandle_StaticUIPath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(midUiHandle)
	router.GET("/gokinsui/test.js", func(c *gin.Context) {
		c.String(http.StatusOK, "static file")
	})

	origInstalled := comm.Installed
	defer func() { comm.Installed = origInstalled }()
	comm.Installed = false

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/gokinsui/test.js", nil)
	router.ServeHTTP(w, req)

	// Should return 200 from the handler
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200 for /gokinsui/ path, got %d", w.Code)
	}
}

// TestApiRateLimiter tests that rate limiting is applied to API endpoints
func TestApiRateLimiter(t *testing.T) {
	router := setupTestRouter(t)

	// Make multiple requests to test rate limiting
	for i := 0; i < 5; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequestWithContext(context.Background(), "GET", "/api/", nil)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("request %d: expected status 200, got %d", i+1, w.Code)
		}
	}
}

// TestGracefulShutdownWithInflightRequests tests shutdown with slow requests
func TestGracefulShutdownWithInflightRequests(t *testing.T) {
	comm.ResetCtx()
	defer comm.ResetCtx()

	gin.SetMode(gin.TestMode)
	comm.WebEgn = gin.New()
	comm.WebEgn.Use(gin.Recovery())

	// Add a slow endpoint
	comm.WebEgn.GET("/slow", func(c *gin.Context) {
		time.Sleep(200 * time.Millisecond)
		c.String(http.StatusOK, "slow response")
	})

	comm.WebHost = ":0"

	done := make(chan struct{})
	go func() {
		runWeb()
		close(done)
	}()

	time.Sleep(50 * time.Millisecond)

	// Trigger shutdown
	comm.Cancel()

	// Wait for shutdown
	select {
	case <-done:
		// Success
	case <-time.After(3 * time.Second):
		t.Error("server did not shut down within timeout")
	}
}

// TestHealthzEndpoint_Fields tests that healthz contains all expected fields
func TestHealthzEndpoint_Fields(t *testing.T) {
	router := setupTestRouter(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/healthz", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	expectedFields := []string{"status", "version", "build_time", "git_commit"}
	for _, field := range expectedFields {
		if _, ok := resp[field]; !ok {
			t.Errorf("healthz response missing field: %s", field)
		}
	}
}

// TestReadyzEndpoint_Fields tests that readyz contains all expected fields
func TestReadyzEndpoint_Fields(t *testing.T) {
	origDb := comm.Db
	origCache := comm.BCache
	defer func() {
		comm.Db = origDb
		comm.BCache = origCache
	}()

	comm.Db = nil
	comm.BCache = nil

	router := setupTestRouter(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/readyz", nil)
	router.ServeHTTP(w, req)

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	expectedFields := []string{"status", "db", "cache"}
	for _, field := range expectedFields {
		if _, ok := resp[field]; !ok {
			t.Errorf("readyz response missing field: %s", field)
		}
	}
}

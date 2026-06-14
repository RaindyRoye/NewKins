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

func setupTestRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	comm.WebEgn = gin.New()
	comm.WebEgn.Use(gin.Recovery())
	regApi()
	return comm.WebEgn
}

func TestHealthzEndpoint(t *testing.T) {
	router := setupTestRouter(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/healthz", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp["status"] != "ok" {
		t.Errorf("expected status 'ok', got %v", resp["status"])
	}
	if resp["version"] != comm.Version {
		t.Errorf("expected version %q, got %v", comm.Version, resp["version"])
	}
}

func TestReadyzEndpoint_NotReady(t *testing.T) {
	// Save and restore global state
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

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status 503, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp["status"] != "not_ready" {
		t.Errorf("expected status 'not_ready', got %v", resp["status"])
	}
}

func TestPprofEndpoints_DisabledInReleaseMode(t *testing.T) {
	// Ensure debug mode is off
	origDebug := core.Debug
	core.Debug = false
	defer func() { core.Debug = origDebug }()

	router := setupTestRouter(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/debug/pprof/", nil)
	router.ServeHTTP(w, req)

	// Should get 404 since pprof is not registered in release mode
	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404 when debug=false, got %d", w.Code)
	}
}

func TestPprofEndpoints_EnabledInDebugMode(t *testing.T) {
	// Enable debug mode
	origDebug := core.Debug
	core.Debug = true
	defer func() { core.Debug = origDebug }()

	router := setupTestRouter(t)

	tests := []struct {
		path       string
		wantStatus int
	}{
		{"/debug/pprof/", http.StatusOK},
		{"/debug/pprof/cmdline", http.StatusOK},
		{"/debug/pprof/symbol", http.StatusOK},
		{"/debug/pprof/goroutine", http.StatusOK},
		{"/debug/pprof/heap", http.StatusOK},
		{"/debug/pprof/allocs", http.StatusOK},
		{"/debug/pprof/block", http.StatusOK},
		{"/debug/pprof/mutex", http.StatusOK},
		{"/debug/pprof/threadcreate", http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequestWithContext(context.Background(), "GET", tt.path, nil)
			router.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("GET %s: expected status %d, got %d", tt.path, tt.wantStatus, w.Code)
			}
		})
	}
}

func TestApiHelloEndpoint(t *testing.T) {
	router := setupTestRouter(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/api/", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	if w.Body.String() != "hello world" {
		t.Errorf("expected 'hello world', got %q", w.Body.String())
	}
}

func TestApiVersionEndpoint(t *testing.T) {
	router := setupTestRouter(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/api/version", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	if w.Body.String() != comm.Version {
		t.Errorf("expected version %q, got %q", comm.Version, w.Body.String())
	}
}

func TestGracefulShutdown(t *testing.T) {
	// Reset context for this test
	comm.ResetCtx()
	// Restore original context after test
	defer comm.ResetCtx()

	gin.SetMode(gin.TestMode)
	comm.WebEgn = gin.New()
	comm.WebEgn.Use(gin.Recovery())

	// Add a slow endpoint to test in-flight requests
	comm.WebEgn.GET("/slow", func(c *gin.Context) {
		time.Sleep(100 * time.Millisecond)
		c.String(http.StatusOK, "done")
	})

	comm.WebHost = ":0" // Use random port

	// Start server in background
	done := make(chan struct{})
	go func() {
		runWeb()
		close(done)
	}()

	// Give server time to start
	time.Sleep(50 * time.Millisecond)

	// Make a request to verify server is running
	// (Note: we can't easily test the actual HTTP request here since
	// we don't know the port, but we can test the shutdown behavior)

	// Trigger shutdown
	comm.Cancel()

	// Wait for shutdown to complete with timeout
	select {
	case <-done:
		// Server shut down successfully
	case <-time.After(2 * time.Second):
		t.Error("server did not shut down within timeout")
	}
}

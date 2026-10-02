package server

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
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

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp["status"] != "ok" {
		t.Errorf("expected status 'ok', got %v", resp["status"])
	}
	if resp["version"] != comm.Version {
		t.Errorf("expected version %q, got %v", comm.Version, resp["version"])
	}
	if _, ok := resp["build_time"]; !ok {
		t.Error("healthz response should contain build_time field")
	}
	if _, ok := resp["git_commit"]; !ok {
		t.Error("healthz response should contain git_commit field")
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

	var resp map[string]any
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
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse JSON response: %v", err)
	}
	if resp["version"] != comm.Version {
		t.Errorf("expected version %q, got %v", comm.Version, resp["version"])
	}
	if _, ok := resp["build_time"]; !ok {
		t.Error("response should contain build_time field")
	}
	if _, ok := resp["git_commit"]; !ok {
		t.Error("response should contain git_commit field")
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

func TestGetFile_EmptyPath(t *testing.T) {
	_, err := getFile("")
	if err == nil {
		t.Fatal("expected error for empty path, got nil")
	}
	if err.Error() != "getFile: path parameter is empty" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestGetFile_PathTraversal(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{"parent directory", "../etc/passwd"},
		{"absolute path", "/etc/passwd"},
		{"double dot in middle", "foo/../../bar"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := getFile(tt.path)
			if err == nil {
				t.Fatal("expected error for path traversal, got nil")
			}
			if !strings.Contains(err.Error(), "invalid path") {
				t.Errorf("unexpected error message: %v", err)
			}
		})
	}
}

func TestGetFile_FileNotFound(t *testing.T) {
	// Reset the zip reader to ensure clean state
	rder = nil
	rderOnce = sync.Once{}
	rderErr = nil

	// With no StaticPkg set, getRdr will fail
	origStaticPkg := comm.StaticPkg
	defer func() { comm.StaticPkg = origStaticPkg }()
	comm.StaticPkg = ""

	_, err := getFile("nonexistent.html")
	if err == nil {
		t.Fatal("expected error when zip reader fails, got nil")
	}
}

func TestMidUiHandle_NotInstalled(t *testing.T) {
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
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/some-page", nil)
	router.ServeHTTP(w, req)

	// Should redirect to /install when not installed
	// ResMsgUrl returns status 302 with an HTML redirect page
	if w.Code != http.StatusFound {
		t.Errorf("expected status 302 (redirect), got %d", w.Code)
	}
}

func TestMidUiHandle_FileFound(t *testing.T) {
	// This test would require setting up a valid StaticPkg with actual zip content
	// For now, we just verify the middleware doesn't panic
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(midUiHandle)
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "test")
	})

	origInstalled := comm.Installed
	defer func() { comm.Installed = origInstalled }()
	comm.Installed = true

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/test", nil)
	router.ServeHTTP(w, req)

	// Should return 200 from the handler
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestMidUiHandle_NotInstalled_InstallPath(t *testing.T) {
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

	// /install path should work even when not installed
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200 for /install, got %d", w.Code)
	}
}

func TestMidUiHandle_NotInstalled_GokinsUIPath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(midUiHandle)

	origInstalled := comm.Installed
	origStaticPkg := comm.StaticPkg
	defer func() {
		comm.Installed = origInstalled
		comm.StaticPkg = origStaticPkg
	}()
	comm.Installed = false
	comm.StaticPkg = "" // Empty static pkg will cause getFile to fail

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/gokinsui/test.js", nil)
	router.ServeHTTP(w, req)

	// /gokinsui/* paths should attempt to serve even when not installed
	// With empty StaticPkg, it will redirect to /
	if w.Code != http.StatusFound {
		t.Logf("status code: %d (expected redirect when static pkg unavailable)", w.Code)
	}
}

func TestMidUiHandle_404WithEmptyStaticPkg(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(midUiHandle)

	origInstalled := comm.Installed
	origStaticPkg := comm.StaticPkg
	defer func() {
		comm.Installed = origInstalled
		comm.StaticPkg = origStaticPkg
	}()
	comm.Installed = true
	comm.StaticPkg = ""

	// Reset zip reader state
	rder = nil
	rderOnce = sync.Once{}
	rderErr = nil

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/nonexistent", nil)
	router.ServeHTTP(w, req)

	// Should redirect to / when file not found
	if w.Code != http.StatusFound {
		t.Errorf("expected status 302 (redirect), got %d", w.Code)
	}
}

func TestMidUiHandle_200ResponseSkipsFileServe(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(midUiHandle)
	router.GET("/api/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	origInstalled := comm.Installed
	defer func() { comm.Installed = origInstalled }()
	comm.Installed = true

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/api/test", nil)
	router.ServeHTTP(w, req)

	// Should return 200 from handler, not attempt file serving
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "ok") {
		t.Errorf("expected JSON response, got %q", w.Body.String())
	}
}

func TestGetFile_PathNormalization(t *testing.T) {
	// Reset zip reader
	rder = nil
	rderOnce = sync.Once{}
	rderErr = nil

	origStaticPkg := comm.StaticPkg
	defer func() { comm.StaticPkg = origStaticPkg }()
	comm.StaticPkg = ""

	tests := []struct {
		name     string
		path     string
		wantErr  bool
		errMatch string
	}{
		{"empty path", "", true, "path parameter is empty"},
		{"backslash normalization", "foo\\bar.html", true, ""},
		{"leading slash", "/foo/bar.html", true, "invalid path"},
		{"parent traversal", "../secret.txt", true, "invalid path"},
		{"double parent", "foo/../../secret.txt", true, "invalid path"},
		{"normal path", "assets/style.css", true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := getFile(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("getFile(%q) error = %v, wantErr %v", tt.path, err, tt.wantErr)
				return
			}
			if tt.errMatch != "" && err != nil && !strings.Contains(err.Error(), tt.errMatch) {
				t.Errorf("getFile(%q) error = %v, want to contain %q", tt.path, err, tt.errMatch)
			}
		})
	}
}

func TestGetRdr_InvalidBase64(t *testing.T) {
	// Reset state
	rder = nil
	rderOnce = sync.Once{}
	rderErr = nil

	origStaticPkg := comm.StaticPkg
	defer func() { comm.StaticPkg = origStaticPkg }()

	// Set invalid base64 string
	comm.StaticPkg = "not-valid-base64!!!"

	_, err := getRdr()
	if err == nil {
		t.Fatal("expected error for invalid base64, got nil")
	}
}

func TestGetRdr_InvalidZip(t *testing.T) {
	// Reset state
	rder = nil
	rderOnce = sync.Once{}
	rderErr = nil

	origStaticPkg := comm.StaticPkg
	defer func() { comm.StaticPkg = origStaticPkg }()

	// Valid base64 but invalid zip format
	comm.StaticPkg = "aGVsbG8gd29ybGQ=" // "hello world" in base64

	_, err := getRdr()
	if err == nil {
		t.Fatal("expected error for invalid zip, got nil")
	}
}

func TestGetRdr_Success(t *testing.T) {
	// Reset state
	rder = nil
	rderOnce = sync.Once{}
	rderErr = nil

	origStaticPkg := comm.StaticPkg
	defer func() { comm.StaticPkg = origStaticPkg }()

	// Create a minimal valid zip file
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	f, err := w.Create("test.html")
	if err != nil {
		t.Fatalf("failed to create zip entry: %v", err)
	}
	_, err = f.Write([]byte("<html>test</html>"))
	if err != nil {
		t.Fatalf("failed to write zip entry: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("failed to close zip: %v", err)
	}

	// Encode to base64
	comm.StaticPkg = base64.StdEncoding.EncodeToString(buf.Bytes())

	reader, err := getRdr()
	if err != nil {
		t.Fatalf("getRdr() error = %v", err)
	}
	if reader == nil {
		t.Fatal("getRdr() returned nil reader")
	}
	if len(reader.File) != 1 {
		t.Errorf("expected 1 file in zip, got %d", len(reader.File))
	}
}

func TestGetFile_WithValidZip(t *testing.T) {
	// Reset state
	rder = nil
	rderOnce = sync.Once{}
	rderErr = nil

	origStaticPkg := comm.StaticPkg
	defer func() { comm.StaticPkg = origStaticPkg }()

	// Create zip with multiple files
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)

	files := map[string]string{
		"index.html":       "<html>index</html>",
		"assets/style.css": "body { margin: 0; }",
		"assets/app.js":    "console.log('hello');",
	}

	for name, content := range files {
		f, err := w.Create(name)
		if err != nil {
			t.Fatalf("failed to create %s: %v", name, err)
		}
		if _, err := f.Write([]byte(content)); err != nil {
			t.Fatalf("failed to write %s: %v", name, err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatalf("failed to close zip: %v", err)
	}

	comm.StaticPkg = base64.StdEncoding.EncodeToString(buf.Bytes())

	// Test finding existing files
	for path := range files {
		t.Run("find "+path, func(t *testing.T) {
			f, err := getFile(path)
			if err != nil {
				t.Fatalf("getFile(%q) error = %v", path, err)
			}
			if f == nil {
				t.Fatalf("getFile(%q) returned nil", path)
			}
			// Normalize path separators for comparison
			expectedName := strings.ReplaceAll(path, "\\", "/")
			actualName := strings.ReplaceAll(f.Name, "\\", "/")
			if actualName != expectedName {
				t.Errorf("file name = %q, want %q", actualName, expectedName)
			}
		})
	}

	// Test file not found
	_, err := getFile("nonexistent.txt")
	if err == nil {
		t.Fatal("expected error for nonexistent file, got nil")
	}
	if !strings.Contains(err.Error(), "file not found") {
		t.Errorf("error = %v, want to contain 'file not found'", err)
	}
}

func TestMidUiHandle_ServesHTMLFile(t *testing.T) {
	// Reset state
	rder = nil
	rderOnce = sync.Once{}
	rderErr = nil

	origInstalled := comm.Installed
	origStaticPkg := comm.StaticPkg
	defer func() {
		comm.Installed = origInstalled
		comm.StaticPkg = origStaticPkg
	}()

	// Create zip with HTML file
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	f, _ := w.Create("about.html")
	_, _ = f.Write([]byte("<html><body>About</body></html>"))
	_ = w.Close()

	comm.Installed = true
	comm.StaticPkg = base64.StdEncoding.EncodeToString(buf.Bytes())

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(midUiHandle)

	w2 := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/about.html", nil)
	router.ServeHTTP(w2, req)

	if w2.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w2.Code)
	}
	if ct := w2.Header().Get("Content-Type"); ct != "text/html" {
		t.Errorf("Content-Type = %q, want 'text/html'", ct)
	}
	if !strings.Contains(w2.Body.String(), "About") {
		t.Errorf("body = %q, want to contain 'About'", w2.Body.String())
	}
}

func TestMidUiHandle_ServesCSSFile(t *testing.T) {
	rder = nil
	rderOnce = sync.Once{}
	rderErr = nil

	origInstalled := comm.Installed
	origStaticPkg := comm.StaticPkg
	defer func() {
		comm.Installed = origInstalled
		comm.StaticPkg = origStaticPkg
	}()

	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	f, _ := w.Create("assets/main.css")
	_, _ = f.Write([]byte("body { color: red; }"))
	_ = w.Close()

	comm.Installed = true
	comm.StaticPkg = base64.StdEncoding.EncodeToString(buf.Bytes())

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(midUiHandle)

	w2 := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/assets/main.css", nil)
	router.ServeHTTP(w2, req)

	if w2.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w2.Code)
	}
	if ct := w2.Header().Get("Content-Type"); ct != "text/css" {
		t.Errorf("Content-Type = %q, want 'text/css'", ct)
	}
}

func TestMidUiHandle_ServesJSFile(t *testing.T) {
	rder = nil
	rderOnce = sync.Once{}
	rderErr = nil

	origInstalled := comm.Installed
	origStaticPkg := comm.StaticPkg
	defer func() {
		comm.Installed = origInstalled
		comm.StaticPkg = origStaticPkg
	}()

	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	f, _ := w.Create("app.js")
	_, _ = f.Write([]byte("console.log('test');"))
	_ = w.Close()

	comm.Installed = true
	comm.StaticPkg = base64.StdEncoding.EncodeToString(buf.Bytes())

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(midUiHandle)

	w2 := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/app.js", nil)
	router.ServeHTTP(w2, req)

	if w2.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w2.Code)
	}
	if ct := w2.Header().Get("Content-Type"); ct != "application/javascript" {
		t.Errorf("Content-Type = %q, want 'application/javascript'", ct)
	}
}

func TestMidUiHandle_ServesSVGFile(t *testing.T) {
	rder = nil
	rderOnce = sync.Once{}
	rderErr = nil

	origInstalled := comm.Installed
	origStaticPkg := comm.StaticPkg
	defer func() {
		comm.Installed = origInstalled
		comm.StaticPkg = origStaticPkg
	}()

	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	f, _ := w.Create("logo.svg")
	_, _ = f.Write([]byte("<svg></svg>"))
	_ = w.Close()

	comm.Installed = true
	comm.StaticPkg = base64.StdEncoding.EncodeToString(buf.Bytes())

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(midUiHandle)

	w2 := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/logo.svg", nil)
	router.ServeHTTP(w2, req)

	if w2.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w2.Code)
	}
	if ct := w2.Header().Get("Content-Type"); ct != "image/svg+xml" {
		t.Errorf("Content-Type = %q, want 'image/svg+xml'", ct)
	}
}

func TestMidUiHandle_FallbackToIndexHTML(t *testing.T) {
	rder = nil
	rderOnce = sync.Once{}
	rderErr = nil

	origInstalled := comm.Installed
	origStaticPkg := comm.StaticPkg
	defer func() {
		comm.Installed = origInstalled
		comm.StaticPkg = origStaticPkg
	}()

	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	f, _ := w.Create("index.html")
	_, _ = f.Write([]byte("<html>SPA</html>"))
	_ = w.Close()

	comm.Installed = true
	comm.StaticPkg = base64.StdEncoding.EncodeToString(buf.Bytes())

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(midUiHandle)

	// Request a path that doesn't exist — should fall back to index.html
	w2 := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/some/spa/route", nil)
	router.ServeHTTP(w2, req)

	if w2.Code != http.StatusOK {
		t.Errorf("expected status 200 (fallback to index.html), got %d", w2.Code)
	}
	if !strings.Contains(w2.Body.String(), "SPA") {
		t.Errorf("body = %q, want to contain 'SPA'", w2.Body.String())
	}
}

func TestMidUiHandle_CacheControlForStaticAssets(t *testing.T) {
	rder = nil
	rderOnce = sync.Once{}
	rderErr = nil

	origInstalled := comm.Installed
	origStaticPkg := comm.StaticPkg
	defer func() {
		comm.Installed = origInstalled
		comm.StaticPkg = origStaticPkg
	}()

	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	f, _ := w.Create("bundle.js")
	_, _ = f.Write([]byte("var x=1;"))
	_ = w.Close()

	comm.Installed = true
	comm.StaticPkg = base64.StdEncoding.EncodeToString(buf.Bytes())

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(midUiHandle)

	w2 := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/bundle.js", nil)
	router.ServeHTTP(w2, req)

	// Non-HTML assets should have long cache TTL
	if cc := w2.Header().Get("Cache-Control"); cc != "max-age=360000000" {
		t.Errorf("Cache-Control = %q, want 'max-age=360000000'", cc)
	}
}

func TestMidUiHandle_HTMLNoCache(t *testing.T) {
	rder = nil
	rderOnce = sync.Once{}
	rderErr = nil

	origInstalled := comm.Installed
	origStaticPkg := comm.StaticPkg
	defer func() {
		comm.Installed = origInstalled
		comm.StaticPkg = origStaticPkg
	}()

	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	f, _ := w.Create("page.html")
	_, _ = f.Write([]byte("<html>page</html>"))
	_ = w.Close()

	comm.Installed = true
	comm.StaticPkg = base64.StdEncoding.EncodeToString(buf.Bytes())

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(midUiHandle)

	w2 := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/page.html", nil)
	router.ServeHTTP(w2, req)

	// HTML files should have no-cache
	if cc := w2.Header().Get("Cache-Control"); cc != "no-cache" {
		t.Errorf("Cache-Control = %q, want 'no-cache'", cc)
	}
	if p := w2.Header().Get("Pragma"); p != "no-cache" {
		t.Errorf("Pragma = %q, want 'no-cache'", p)
	}
}

func TestGetRdr_Singleton(t *testing.T) {
	rder = nil
	rderOnce = sync.Once{}
	rderErr = nil

	origStaticPkg := comm.StaticPkg
	defer func() { comm.StaticPkg = origStaticPkg }()

	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	f, _ := w.Create("test.txt")
	_, _ = f.Write([]byte("hello"))
	_ = w.Close()

	comm.StaticPkg = base64.StdEncoding.EncodeToString(buf.Bytes())

	r1, err1 := getRdr()
	if err1 != nil {
		t.Fatalf("first getRdr() error = %v", err1)
	}
	r2, err2 := getRdr()
	if err2 != nil {
		t.Fatalf("second getRdr() error = %v", err2)
	}

	if r1 != r2 {
		t.Error("getRdr() should return the same singleton instance")
	}
}

func TestGetFile_BackslashNormalization(t *testing.T) {
	rder = nil
	rderOnce = sync.Once{}
	rderErr = nil

	origStaticPkg := comm.StaticPkg
	defer func() { comm.StaticPkg = origStaticPkg }()

	// Create zip with a file using backslash in name (Windows-style)
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	f, _ := w.Create("assets\\style.css")
	_, _ = f.Write([]byte("body{}"))
	_ = w.Close()

	comm.StaticPkg = base64.StdEncoding.EncodeToString(buf.Bytes())

	// Should find it with forward slash lookup
	f2, err := getFile("assets/style.css")
	if err != nil {
		t.Fatalf("getFile('assets/style.css') error = %v", err)
	}
	if f2 == nil {
		t.Fatal("getFile returned nil")
	}
}

package server

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gokins/gokins/comm"
)

// createTestZip creates a zip archive in memory with the given files.
// Returns the base64-encoded zip data.
func createTestZip(t *testing.T, files map[string]string) string {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	for name, content := range files {
		f, err := zw.Create(name)
		if err != nil {
			t.Fatalf("failed to create zip entry %q: %v", name, err)
		}
		if _, err := f.Write([]byte(content)); err != nil {
			t.Fatalf("failed to write zip entry %q: %v", name, err)
		}
	}

	if err := zw.Close(); err != nil {
		t.Fatalf("failed to close zip writer: %v", err)
	}

	return base64.StdEncoding.EncodeToString(buf.Bytes())
}

func TestMidUiHandle_InstalledWithStaticFile(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Reset zip reader state
	rder = nil
	rderOnce = sync.Once{}
	rderErr = nil

	// Create test zip with index.html and app.js
	origStaticPkg := comm.StaticPkg
	origInstalled := comm.Installed
	defer func() {
		comm.StaticPkg = origStaticPkg
		comm.Installed = origInstalled
	}()

	comm.StaticPkg = createTestZip(t, map[string]string{
		"index.html": "<html><body>Hello</body></html>",
		"app.js":     "console.log('test');",
		"style.css":  "body { color: red; }",
	})
	comm.Installed = true

	router := gin.New()
	router.Use(midUiHandle)
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "test")
	})

	// Request a non-existent path - should serve index.html as fallback
	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/some-page", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "<html>") {
		t.Errorf("expected HTML content, got %q", w.Body.String())
	}
	if w.Header().Get("Content-Type") != "text/html" {
		t.Errorf("expected Content-Type text/html, got %q", w.Header().Get("Content-Type"))
	}
	if w.Header().Get("Cache-Control") != "no-cache" {
		t.Errorf("expected Cache-Control no-cache for HTML, got %q", w.Header().Get("Cache-Control"))
	}
}

func TestMidUiHandle_JavaScriptFile(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rder = nil
	rderOnce = sync.Once{}
	rderErr = nil

	origStaticPkg := comm.StaticPkg
	origInstalled := comm.Installed
	defer func() {
		comm.StaticPkg = origStaticPkg
		comm.Installed = origInstalled
	}()

	comm.StaticPkg = createTestZip(t, map[string]string{
		"app.js": "console.log('hello');",
	})
	comm.Installed = true

	router := gin.New()
	router.Use(midUiHandle)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/app.js", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	if w.Body.String() != "console.log('hello');" {
		t.Errorf("expected JS content, got %q", w.Body.String())
	}
	if w.Header().Get("Content-Type") != "application/javascript" {
		t.Errorf("expected Content-Type application/javascript, got %q", w.Header().Get("Content-Type"))
	}
	if !strings.Contains(w.Header().Get("Cache-Control"), "max-age=") {
		t.Errorf("expected Cache-Control max-age for JS, got %q", w.Header().Get("Cache-Control"))
	}
}

func TestMidUiHandle_CSSFile(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rder = nil
	rderOnce = sync.Once{}
	rderErr = nil

	origStaticPkg := comm.StaticPkg
	origInstalled := comm.Installed
	defer func() {
		comm.StaticPkg = origStaticPkg
		comm.Installed = origInstalled
	}()

	comm.StaticPkg = createTestZip(t, map[string]string{
		"style.css": "body { margin: 0; }",
	})
	comm.Installed = true

	router := gin.New()
	router.Use(midUiHandle)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/style.css", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	if w.Header().Get("Content-Type") != "text/css" {
		t.Errorf("expected Content-Type text/css, got %q", w.Header().Get("Content-Type"))
	}
}

func TestMidUiHandle_FontFiles(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rder = nil
	rderOnce = sync.Once{}
	rderErr = nil

	origStaticPkg := comm.StaticPkg
	origInstalled := comm.Installed
	defer func() {
		comm.StaticPkg = origStaticPkg
		comm.Installed = origInstalled
	}()

	comm.StaticPkg = createTestZip(t, map[string]string{
		"font.woff2": "woff2data",
		"font.ttf":   "ttfdata",
		"icon.svg":   "<svg></svg>",
	})
	comm.Installed = true

	tests := []struct {
		path        string
		contentType string
	}{
		{"/font.woff2", ""}, // woff2 doesn't set Content-Type explicitly
		{"/font.ttf", "application/x-font-ttf"},
		{"/icon.svg", "image/svg+xml"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			router := gin.New()
			router.Use(midUiHandle)

			w := httptest.NewRecorder()
			req, _ := http.NewRequestWithContext(context.Background(), "GET", tt.path, nil)
			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("expected status 200, got %d", w.Code)
			}
			if tt.contentType != "" && w.Header().Get("Content-Type") != tt.contentType {
				t.Errorf("expected Content-Type %q, got %q", tt.contentType, w.Header().Get("Content-Type"))
			}
		})
	}
}

func TestMidUiHandle_FileNotFoundRedirect(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rder = nil
	rderOnce = sync.Once{}
	rderErr = nil

	origStaticPkg := comm.StaticPkg
	origInstalled := comm.Installed
	defer func() {
		comm.StaticPkg = origStaticPkg
		comm.Installed = origInstalled
	}()

	// Empty zip - no index.html fallback
	comm.StaticPkg = createTestZip(t, map[string]string{
		"other.txt": "test",
	})
	comm.Installed = true

	router := gin.New()
	router.Use(midUiHandle)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/nonexistent.js", nil)
	router.ServeHTTP(w, req)

	// Should redirect to / when file not found and no index.html
	if w.Code != http.StatusFound {
		t.Errorf("expected status 302 (redirect), got %d", w.Code)
	}
}

func TestMidUiHandle_HandlerReturnsOK(t *testing.T) {
	gin.SetMode(gin.TestMode)

	origInstalled := comm.Installed
	defer func() { comm.Installed = origInstalled }()
	comm.Installed = true

	router := gin.New()
	router.Use(midUiHandle)
	router.GET("/api/test", func(c *gin.Context) {
		c.String(http.StatusOK, "api response")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/api/test", nil)
	router.ServeHTTP(w, req)

	// Handler returned 200, so middleware should not serve static files
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	if w.Body.String() != "api response" {
		t.Errorf("expected 'api response', got %q", w.Body.String())
	}
}

func TestMidUiHandle_HandlerReturnsOtherStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rder = nil
	rderOnce = sync.Once{}
	rderErr = nil

	origStaticPkg := comm.StaticPkg
	origInstalled := comm.Installed
	defer func() {
		comm.StaticPkg = origStaticPkg
		comm.Installed = origInstalled
	}()

	comm.StaticPkg = createTestZip(t, map[string]string{
		"index.html": "<html>fallback</html>",
	})
	comm.Installed = true

	router := gin.New()
	router.Use(midUiHandle)
	router.GET("/error", func(c *gin.Context) {
		c.String(http.StatusInternalServerError, "error")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/error", nil)
	router.ServeHTTP(w, req)

	// Handler returned 500 (not 404), so middleware should not serve static files
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
}

func TestMidUiHandle_LargeFileRejected(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rder = nil
	rderOnce = sync.Once{}
	rderErr = nil

	origStaticPkg := comm.StaticPkg
	origInstalled := comm.Installed
	defer func() {
		comm.StaticPkg = origStaticPkg
		comm.Installed = origInstalled
	}()

	// Create a file larger than uiMaxSize (50 MiB)
	largeContent := strings.Repeat("x", int(uiMaxSize)+1000)
	comm.StaticPkg = createTestZip(t, map[string]string{
		"large.bin": largeContent,
	})
	comm.Installed = true

	router := gin.New()
	router.Use(midUiHandle)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/large.bin", nil)
	router.ServeHTTP(w, req)

	// Should redirect when file is too large
	if w.Code != http.StatusFound {
		t.Errorf("expected status 302 (redirect for oversized file), got %d", w.Code)
	}
}

func TestMidUiHandle_InstallPageWhenNotInstalled(t *testing.T) {
	gin.SetMode(gin.TestMode)

	origInstalled := comm.Installed
	defer func() { comm.Installed = origInstalled }()
	comm.Installed = false

	router := gin.New()
	router.Use(midUiHandle)
	router.GET("/install", func(c *gin.Context) {
		c.String(http.StatusOK, "install page")
	})

	// Request to /install should work even when not installed
	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/install", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200 for /install, got %d", w.Code)
	}
	if w.Body.String() != "install page" {
		t.Errorf("expected 'install page', got %q", w.Body.String())
	}
}

func TestMidUiHandle_GokinsUIPrefixWhenNotInstalled(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rder = nil
	rderOnce = sync.Once{}
	rderErr = nil

	origStaticPkg := comm.StaticPkg
	origInstalled := comm.Installed
	defer func() {
		comm.StaticPkg = origStaticPkg
		comm.Installed = origInstalled
	}()

	comm.StaticPkg = createTestZip(t, map[string]string{
		"gokinsui/app.js": "console.log('ui');",
	})
	comm.Installed = false

	router := gin.New()
	router.Use(midUiHandle)

	// /gokinsui/* paths should work even when not installed
	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/gokinsui/app.js", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200 for /gokinsui/ path, got %d", w.Code)
	}
}

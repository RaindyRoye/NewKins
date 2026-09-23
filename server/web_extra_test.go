package server

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gokins/gokins/comm"
)

// makeZipPkg creates an in-memory zip archive and returns its base64 encoding.
func makeZipPkg(t *testing.T, files map[string]string) string {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for name, content := range files {
		f, err := w.Create(name)
		if err != nil {
			t.Fatalf("zip create %s: %v", name, err)
		}
		if _, err := f.Write([]byte(content)); err != nil {
			t.Fatalf("zip write %s: %v", name, err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatalf("zip close: %v", err)
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes())
}

// resetZipReader resets the package-level zip reader state.
func resetZipReader() {
	rder = nil
	rderOnce = sync.Once{}
	rderErr = nil
}

func TestGetRdr_ValidZip(t *testing.T) {
	resetZipReader()
	origStaticPkg := comm.StaticPkg
	defer func() {
		comm.StaticPkg = origStaticPkg
		resetZipReader()
	}()

	comm.StaticPkg = makeZipPkg(t, map[string]string{
		"index.html": "<html>hello</html>",
		"style.css":  "body{}",
	})

	r, err := getRdr()
	if err != nil {
		t.Fatalf("getRdr error: %v", err)
	}
	if r == nil {
		t.Fatal("getRdr returned nil reader")
	}
	if len(r.File) != 2 {
		t.Errorf("expected 2 files in zip, got %d", len(r.File))
	}

	// Calling again should return the same cached reader
	r2, err := getRdr()
	if err != nil {
		t.Fatalf("second getRdr error: %v", err)
	}
	if r != r2 {
		t.Error("expected cached reader to be returned")
	}
}

func TestGetRdr_InvalidBase64(t *testing.T) {
	resetZipReader()
	origStaticPkg := comm.StaticPkg
	defer func() {
		comm.StaticPkg = origStaticPkg
		resetZipReader()
	}()

	comm.StaticPkg = "!!!not-valid-base64!!!"
	_, err := getRdr()
	if err == nil {
		t.Fatal("expected error for invalid base64")
	}
}

func TestGetRdr_InvalidZip(t *testing.T) {
	resetZipReader()
	origStaticPkg := comm.StaticPkg
	defer func() {
		comm.StaticPkg = origStaticPkg
		resetZipReader()
	}()

	comm.StaticPkg = base64.StdEncoding.EncodeToString([]byte("not a zip file"))
	_, err := getRdr()
	if err == nil {
		t.Fatal("expected error for invalid zip data")
	}
}

func TestGetFile_ValidFile(t *testing.T) {
	resetZipReader()
	origStaticPkg := comm.StaticPkg
	defer func() {
		comm.StaticPkg = origStaticPkg
		resetZipReader()
	}()

	comm.StaticPkg = makeZipPkg(t, map[string]string{
		"index.html":    "<html>test</html>",
		"app.js":        "console.log('hi')",
		"style.css":     "body { color: red; }",
		"icon.svg":      "<svg></svg>",
		"font.woff2":    "woff2data",
		"font.ttf":      "ttfdata",
		"sub/page.html": "<html>page</html>",
	})

	tests := []struct {
		path string
	}{
		{"index.html"},
		{"app.js"},
		{"style.css"},
		{"icon.svg"},
		{"font.woff2"},
		{"font.ttf"},
		{"sub/page.html"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			f, err := getFile(tt.path)
			if err != nil {
				t.Fatalf("getFile(%q) error: %v", tt.path, err)
			}
			if f == nil {
				t.Fatalf("getFile(%q) returned nil", tt.path)
			}
		})
	}
}

func TestGetFile_NotFoundInZip(t *testing.T) {
	resetZipReader()
	origStaticPkg := comm.StaticPkg
	defer func() {
		comm.StaticPkg = origStaticPkg
		resetZipReader()
	}()

	comm.StaticPkg = makeZipPkg(t, map[string]string{
		"index.html": "<html>test</html>",
	})

	_, err := getFile("nonexistent.html")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
	if err.Error() != "file not found" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestMidUiHandle_NotInstalledRedirect(t *testing.T) {
	resetZipReader()
	origInstalled := comm.Installed
	origStaticPkg := comm.StaticPkg
	defer func() {
		comm.Installed = origInstalled
		comm.StaticPkg = origStaticPkg
		resetZipReader()
	}()

	comm.Installed = false
	comm.StaticPkg = ""

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(midUiHandle)
	router.GET("/install", func(c *gin.Context) {
		c.String(http.StatusOK, "install page")
	})

	// Non-install, non-UI path should redirect
	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/some-page", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("expected 302 redirect, got %d", w.Code)
	}
}

func TestMidUiHandle_NotInstalled_AllowUIPath(t *testing.T) {
	resetZipReader()
	origInstalled := comm.Installed
	origStaticPkg := comm.StaticPkg
	defer func() {
		comm.Installed = origInstalled
		comm.StaticPkg = origStaticPkg
		resetZipReader()
	}()

	comm.Installed = false
	comm.StaticPkg = ""

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(midUiHandle)
	router.GET("/gokinsui/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ui content")
	})

	// /gokinsui/ paths should be allowed even when not installed
	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/gokinsui/test", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for UI path, got %d", w.Code)
	}
}

func TestMidUiHandle_NotInstalled_AllowInstallPath(t *testing.T) {
	resetZipReader()
	origInstalled := comm.Installed
	defer func() { comm.Installed = origInstalled }()

	comm.Installed = false

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(midUiHandle)
	router.GET("/install", func(c *gin.Context) {
		c.String(http.StatusOK, "install page")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/install", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for /install path, got %d", w.Code)
	}
}

func TestMidUiHandle_Installed_FileNotFound_Redirect(t *testing.T) {
	resetZipReader()
	origInstalled := comm.Installed
	origStaticPkg := comm.StaticPkg
	defer func() {
		comm.Installed = origInstalled
		comm.StaticPkg = origStaticPkg
		resetZipReader()
	}()

	comm.Installed = true
	comm.StaticPkg = "" // No static pkg → getFile fails → redirect

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(midUiHandle)
	router.GET("/", func(c *gin.Context) {
		// This won't be called since midUiHandle handles the response
		c.String(http.StatusOK, "root")
	})

	// Request a non-existent path — should redirect because getFile fails
	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/nonexistent", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("expected 302 redirect when file not found, got %d", w.Code)
	}
}

func TestMidUiHandle_Installed_HTMLFileServed(t *testing.T) {
	resetZipReader()
	origInstalled := comm.Installed
	origStaticPkg := comm.StaticPkg
	defer func() {
		comm.Installed = origInstalled
		comm.StaticPkg = origStaticPkg
		resetZipReader()
	}()

	comm.Installed = true
	comm.StaticPkg = makeZipPkg(t, map[string]string{
		"index.html": "<html>test page</html>",
	})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(midUiHandle)
	// The handler falls through (no matching route) → midUiHandle serves the file
	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/index.html", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "text/html" {
		t.Errorf("expected Content-Type text/html, got %q", ct)
	}
	if w.Header().Get("Cache-Control") != "no-cache" {
		t.Errorf("expected Cache-Control no-cache for HTML, got %q", w.Header().Get("Cache-Control"))
	}
}

func TestMidUiHandle_Installed_JSFileServed(t *testing.T) {
	resetZipReader()
	origInstalled := comm.Installed
	origStaticPkg := comm.StaticPkg
	defer func() {
		comm.Installed = origInstalled
		comm.StaticPkg = origStaticPkg
		resetZipReader()
	}()

	comm.Installed = true
	comm.StaticPkg = makeZipPkg(t, map[string]string{
		"app.js": "console.log('hello')",
	})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(midUiHandle)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/app.js", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/javascript" {
		t.Errorf("expected Content-Type application/javascript, got %q", ct)
	}
}

func TestMidUiHandle_Installed_CSSFileServed(t *testing.T) {
	resetZipReader()
	origInstalled := comm.Installed
	origStaticPkg := comm.StaticPkg
	defer func() {
		comm.Installed = origInstalled
		comm.StaticPkg = origStaticPkg
		resetZipReader()
	}()

	comm.Installed = true
	comm.StaticPkg = makeZipPkg(t, map[string]string{
		"style.css": "body { margin: 0; }",
	})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(midUiHandle)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/style.css", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "text/css" {
		t.Errorf("expected Content-Type text/css, got %q", ct)
	}
}

func TestMidUiHandle_Installed_SVGFileServed(t *testing.T) {
	resetZipReader()
	origInstalled := comm.Installed
	origStaticPkg := comm.StaticPkg
	defer func() {
		comm.Installed = origInstalled
		comm.StaticPkg = origStaticPkg
		resetZipReader()
	}()

	comm.Installed = true
	comm.StaticPkg = makeZipPkg(t, map[string]string{
		"icon.svg": "<svg></svg>",
	})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(midUiHandle)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/icon.svg", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "image/svg+xml" {
		t.Errorf("expected Content-Type image/svg+xml, got %q", ct)
	}
}

func TestMidUiHandle_Installed_TTFServed(t *testing.T) {
	resetZipReader()
	origInstalled := comm.Installed
	origStaticPkg := comm.StaticPkg
	defer func() {
		comm.Installed = origInstalled
		comm.StaticPkg = origStaticPkg
		resetZipReader()
	}()

	comm.Installed = true
	comm.StaticPkg = makeZipPkg(t, map[string]string{
		"font.ttf": "ttfbinary",
	})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(midUiHandle)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/font.ttf", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/x-font-ttf" {
		t.Errorf("expected Content-Type application/x-font-ttf, got %q", ct)
	}
}

func TestMidUiHandle_ExistingRoute_NotInterfered(t *testing.T) {
	resetZipReader()
	origInstalled := comm.Installed
	defer func() { comm.Installed = origInstalled }()

	comm.Installed = true

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(midUiHandle)
	router.GET("/api/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/api/test", nil)
	router.ServeHTTP(w, req)

	// Existing routes should return their normal status code (not 404)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 from existing route, got %d", w.Code)
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
}

func TestMidUiHandle_Installed_FallbackToIndex(t *testing.T) {
	resetZipReader()
	origInstalled := comm.Installed
	origStaticPkg := comm.StaticPkg
	defer func() {
		comm.Installed = origInstalled
		comm.StaticPkg = origStaticPkg
		resetZipReader()
	}()

	comm.Installed = true
	// Only index.html in the zip — request for "nonexistent" should fall back
	comm.StaticPkg = makeZipPkg(t, map[string]string{
		"index.html": "<html>SPA app</html>",
	})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(midUiHandle)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/spa-route", nil)
	router.ServeHTTP(w, req)

	// Should serve index.html as fallback
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 from SPA fallback, got %d", w.Code)
	}
	if w.Header().Get("Content-Type") != "text/html" {
		t.Errorf("expected text/html for SPA fallback, got %q", w.Header().Get("Content-Type"))
	}
}

func TestInitCache_Success(t *testing.T) {
	origWorkPath := comm.WorkPath
	origBCache := comm.BCache
	defer func() {
		comm.WorkPath = origWorkPath
		if comm.BCache != nil {
			_ = comm.BCache.Close()
		}
		comm.BCache = origBCache
	}()

	tmpDir := t.TempDir()
	comm.WorkPath = tmpDir

	err := initCache()
	if err != nil {
		t.Fatalf("initCache error: %v", err)
	}
	if comm.BCache == nil {
		t.Fatal("BCache should not be nil after initCache")
	}

	// Verify cache file was created
	cachePath := filepath.Join(tmpDir, "cache.dat")
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		t.Error("cache.dat was not created")
	}
}

func TestInitCache_InvalidPath(t *testing.T) {
	origWorkPath := comm.WorkPath
	origBCache := comm.BCache
	defer func() {
		comm.WorkPath = origWorkPath
		comm.BCache = origBCache
	}()

	// Use a non-existent directory that can't be created
	comm.WorkPath = "/dev/null/impossible/path"

	err := initCache()
	if err == nil {
		if comm.BCache != nil {
			_ = comm.BCache.Close()
		}
		t.Fatal("expected error for invalid cache path")
	}
}

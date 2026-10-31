package server

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gokins/gokins/comm"
)

// createTestZip creates a zip archive with the given files and returns it as base64.
func createTestZip(t *testing.T, files map[string]string) string {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, content := range files {
		fw, err := zw.Create(name)
		if err != nil {
			t.Fatalf("failed to create zip entry %s: %v", name, err)
		}
		if _, err := fw.Write([]byte(content)); err != nil {
			t.Fatalf("failed to write zip entry %s: %v", name, err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("failed to close zip: %v", err)
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes())
}

// resetZipReader clears the cached zip reader state.
func resetZipReader() {
	rder = nil
	rderOnce = sync.Once{}
	rderErr = nil
}

func TestMidUiHandle_Non404Status(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(midUiHandle)
	router.GET("/ok", func(c *gin.Context) {
		c.String(http.StatusOK, "success")
	})

	origInstalled := comm.Installed
	defer func() { comm.Installed = origInstalled }()
	comm.Installed = true

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(t.Context(), "GET", "/ok", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if w.Body.String() != "success" {
		t.Errorf("expected 'success', got %q", w.Body.String())
	}
}

func TestMidUiHandle_HasResponseContent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(midUiHandle)
	router.GET("/api/test", func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
	})

	origInstalled := comm.Installed
	defer func() { comm.Installed = origInstalled }()
	comm.Installed = true

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(t.Context(), "GET", "/api/test", nil)
	router.ServeHTTP(w, req)

	// Should return 404 with content (not trigger UI fallback)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "error") {
		t.Errorf("expected error in body, got %q", w.Body.String())
	}
}

func TestMidUiHandle_NotInstalled_RedirectToInstall(t *testing.T) {
	gin.SetMode(gin.TestMode)
	origInstalled := comm.Installed
	defer func() { comm.Installed = origInstalled }()
	comm.Installed = false

	router := gin.New()
	router.Use(midUiHandle)
	router.GET("/somepath", func(c *gin.Context) {
		c.Status(http.StatusNotFound)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(t.Context(), "GET", "/somepath", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("expected 302 redirect, got %d", w.Code)
	}
}

func TestMidUiHandle_NotInstalled_AllowGokinsUI(t *testing.T) {
	gin.SetMode(gin.TestMode)
	origInstalled := comm.Installed
	origStaticPkg := comm.StaticPkg
	defer func() {
		comm.Installed = origInstalled
		comm.StaticPkg = origStaticPkg
		resetZipReader()
	}()
	comm.Installed = false
	comm.StaticPkg = createTestZip(t, map[string]string{
		"gokinsui/app.js": "console.log('test');",
	})
	resetZipReader()

	router := gin.New()
	router.Use(midUiHandle)
	router.GET("/*path", func(c *gin.Context) {
		c.Status(http.StatusNotFound)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(t.Context(), "GET", "/gokinsui/app.js", nil)
	router.ServeHTTP(w, req)

	// Should serve the file even when not installed
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestMidUiHandle_NotInstalled_AllowInstallPath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	origInstalled := comm.Installed
	defer func() { comm.Installed = origInstalled }()
	comm.Installed = false

	router := gin.New()
	router.Use(midUiHandle)
	router.GET("/install", func(c *gin.Context) {
		c.String(http.StatusOK, "install page")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(t.Context(), "GET", "/install", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestMidUiHandle_FileNotFound_FallbackToIndex(t *testing.T) {
	gin.SetMode(gin.TestMode)
	origInstalled := comm.Installed
	origStaticPkg := comm.StaticPkg
	defer func() {
		comm.Installed = origInstalled
		comm.StaticPkg = origStaticPkg
		resetZipReader()
	}()
	comm.Installed = true
	comm.StaticPkg = createTestZip(t, map[string]string{
		"index.html": "<html><body>Home</body></html>",
	})
	resetZipReader()

	router := gin.New()
	router.Use(midUiHandle)
	router.GET("/*path", func(c *gin.Context) {
		c.Status(http.StatusNotFound)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(t.Context(), "GET", "/nonexistent.html", nil)
	router.ServeHTTP(w, req)

	// Should fallback to index.html
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Home") {
		t.Errorf("expected index.html content, got %q", w.Body.String())
	}
}

func TestMidUiHandle_FileNotFound_NoIndex(t *testing.T) {
	gin.SetMode(gin.TestMode)
	origInstalled := comm.Installed
	origStaticPkg := comm.StaticPkg
	defer func() {
		comm.Installed = origInstalled
		comm.StaticPkg = origStaticPkg
		resetZipReader()
	}()
	comm.Installed = true
	// Empty zip with no files
	comm.StaticPkg = createTestZip(t, map[string]string{})
	resetZipReader()

	router := gin.New()
	router.Use(midUiHandle)
	router.GET("/*path", func(c *gin.Context) {
		c.Status(http.StatusNotFound)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(t.Context(), "GET", "/notexist.html", nil)
	router.ServeHTTP(w, req)

	// Should redirect to / when neither file nor index.html found
	if w.Code != http.StatusFound {
		t.Errorf("expected 302 redirect to /, got %d", w.Code)
	}
}

func TestMidUiHandle_ValidHTMLFile(t *testing.T) {
	gin.SetMode(gin.TestMode)
	origInstalled := comm.Installed
	origStaticPkg := comm.StaticPkg
	defer func() {
		comm.Installed = origInstalled
		comm.StaticPkg = origStaticPkg
		resetZipReader()
	}()
	comm.Installed = true
	comm.StaticPkg = createTestZip(t, map[string]string{
		"index.html": "<html><body>Hello</body></html>",
	})
	resetZipReader()

	router := gin.New()
	router.Use(midUiHandle)
	router.GET("/*path", func(c *gin.Context) {
		c.Status(http.StatusNotFound)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(t.Context(), "GET", "/index.html", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if w.Header().Get("Content-Type") != "text/html" {
		t.Errorf("expected Content-Type text/html, got %s", w.Header().Get("Content-Type"))
	}
	if w.Header().Get("Cache-Control") != "no-cache" {
		t.Errorf("expected Cache-Control no-cache, got %s", w.Header().Get("Cache-Control"))
	}
}

func TestMidUiHandle_AssetTypes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	origInstalled := comm.Installed
	origStaticPkg := comm.StaticPkg
	defer func() {
		comm.Installed = origInstalled
		comm.StaticPkg = origStaticPkg
		resetZipReader()
	}()
	comm.Installed = true
	comm.StaticPkg = createTestZip(t, map[string]string{
		"style.css": "body { color: red; }",
		"app.js":    "console.log('hello');",
		"logo.svg":  "<svg></svg>",
	})
	resetZipReader()

	router := gin.New()
	router.Use(midUiHandle)
	router.GET("/*path", func(c *gin.Context) {
		c.Status(http.StatusNotFound)
	})

	tests := []struct {
		path, contentType string
	}{
		{"/style.css", "text/css"},
		{"/app.js", "application/javascript"},
		{"/logo.svg", "image/svg+xml"},
	}

	for _, tt := range tests {
		w := httptest.NewRecorder()
		req, _ := http.NewRequestWithContext(t.Context(), "GET", tt.path, nil)
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("%s: expected 200, got %d", tt.path, w.Code)
		}
		if w.Header().Get("Content-Type") != tt.contentType {
			t.Errorf("%s: expected %s, got %s", tt.path, tt.contentType, w.Header().Get("Content-Type"))
		}
	}
}

func TestMidUiHandle_FontFiles(t *testing.T) {
	gin.SetMode(gin.TestMode)
	origInstalled := comm.Installed
	origStaticPkg := comm.StaticPkg
	defer func() {
		comm.Installed = origInstalled
		comm.StaticPkg = origStaticPkg
		resetZipReader()
	}()
	comm.Installed = true
	comm.StaticPkg = createTestZip(t, map[string]string{
		"font.woff2": "woff2data",
		"font.ttf":   "ttfdata",
	})
	resetZipReader()

	router := gin.New()
	router.Use(midUiHandle)
	router.GET("/*path", func(c *gin.Context) {
		c.Status(http.StatusNotFound)
	})

	// Test woff2 font file
	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(t.Context(), "GET", "/font.woff2", nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("woff2: expected 200, got %d", w.Code)
	}

	// Test ttf font file with explicit Content-Type
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequestWithContext(t.Context(), "GET", "/font.ttf", nil)
	router.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Errorf("ttf: expected 200, got %d", w2.Code)
	}
	if w2.Header().Get("Content-Type") != "application/x-font-ttf" {
		t.Errorf("ttf: expected Content-Type application/x-font-ttf, got %s", w2.Header().Get("Content-Type"))
	}
}

func TestMidUiHandle_InvalidBase64(t *testing.T) {
	gin.SetMode(gin.TestMode)
	origInstalled := comm.Installed
	origStaticPkg := comm.StaticPkg
	defer func() {
		comm.Installed = origInstalled
		comm.StaticPkg = origStaticPkg
		resetZipReader()
	}()
	comm.Installed = true
	comm.StaticPkg = "invalid-base64-!!!"
	resetZipReader()

	router := gin.New()
	router.Use(midUiHandle)
	router.GET("/*path", func(c *gin.Context) {
		c.Status(http.StatusNotFound)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(t.Context(), "GET", "/test.html", nil)
	router.ServeHTTP(w, req)

	// Should redirect when zip cannot be decoded
	if w.Code != http.StatusFound {
		t.Errorf("expected 302 redirect, got %d", w.Code)
	}
}

func TestMidUiHandle_InvalidZip(t *testing.T) {
	gin.SetMode(gin.TestMode)
	origInstalled := comm.Installed
	origStaticPkg := comm.StaticPkg
	defer func() {
		comm.Installed = origInstalled
		comm.StaticPkg = origStaticPkg
		resetZipReader()
	}()
	comm.Installed = true
	// Valid base64 but invalid zip format
	comm.StaticPkg = base64.StdEncoding.EncodeToString([]byte("not a zip file"))
	resetZipReader()

	router := gin.New()
	router.Use(midUiHandle)
	router.GET("/*path", func(c *gin.Context) {
		c.Status(http.StatusNotFound)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(t.Context(), "GET", "/test.html", nil)
	router.ServeHTTP(w, req)

	// Should redirect when zip cannot be parsed
	if w.Code != http.StatusFound {
		t.Errorf("expected 302 redirect, got %d", w.Code)
	}
}

func TestGetRdr_InvalidBase64(t *testing.T) {
	origStaticPkg := comm.StaticPkg
	defer func() {
		comm.StaticPkg = origStaticPkg
		resetZipReader()
	}()
	comm.StaticPkg = "invalid-base64"
	resetZipReader()

	_, err := getRdr()
	if err == nil {
		t.Error("expected error for invalid base64, got nil")
	}
}

func TestGetRdr_InvalidZip(t *testing.T) {
	origStaticPkg := comm.StaticPkg
	defer func() {
		comm.StaticPkg = origStaticPkg
		resetZipReader()
	}()
	comm.StaticPkg = base64.StdEncoding.EncodeToString([]byte("not a zip"))
	resetZipReader()

	_, err := getRdr()
	if err == nil {
		t.Error("expected error for invalid zip, got nil")
	}
}

func TestGetRdr_CachesResult(t *testing.T) {
	origStaticPkg := comm.StaticPkg
	defer func() {
		comm.StaticPkg = origStaticPkg
		resetZipReader()
	}()
	comm.StaticPkg = createTestZip(t, map[string]string{"test.txt": "hello"})
	resetZipReader()

	r1, err := getRdr()
	if err != nil {
		t.Fatalf("first getRdr failed: %v", err)
	}
	r2, err := getRdr()
	if err != nil {
		t.Fatalf("second getRdr failed: %v", err)
	}
	if r1 != r2 {
		t.Error("getRdr should cache and return the same reader")
	}
}

func TestGetFile_ValidFile(t *testing.T) {
	origStaticPkg := comm.StaticPkg
	defer func() {
		comm.StaticPkg = origStaticPkg
		resetZipReader()
	}()
	comm.StaticPkg = createTestZip(t, map[string]string{"test.txt": "hello world"})
	resetZipReader()

	f, err := getFile("test.txt")
	if err != nil {
		t.Fatalf("getFile failed: %v", err)
	}
	if f.Name != "test.txt" {
		t.Errorf("expected file name test.txt, got %s", f.Name)
	}
}

func TestGetFile_BackslashNormalization(t *testing.T) {
	origStaticPkg := comm.StaticPkg
	defer func() {
		comm.StaticPkg = origStaticPkg
		resetZipReader()
	}()
	// Zip with backslash path separator (Windows-style)
	comm.StaticPkg = createTestZip(t, map[string]string{
		"subdir\\file.txt": "content",
	})
	resetZipReader()

	f, err := getFile("subdir/file.txt")
	if err != nil {
		t.Fatalf("getFile failed: %v", err)
	}
	if f == nil {
		t.Fatal("expected file, got nil")
	}
}

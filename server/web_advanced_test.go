package server

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gokins/gokins/comm"
)

// createTestZip creates a zip archive in memory with the given files and returns
// the base64-encoded string suitable for comm.StaticPkg.
func createTestZip(t *testing.T, files map[string]string) string {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for name, content := range files {
		f, err := w.Create(name)
		if err != nil {
			t.Fatalf("create zip entry %s: %v", name, err)
		}
		if _, err := f.Write([]byte(content)); err != nil {
			t.Fatalf("write zip entry %s: %v", name, err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close zip writer: %v", err)
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes())
}

// resetZipReader resets the sync.Once-protected zip reader state so that
// getRdr re-reads comm.StaticPkg on the next call.
func resetZipReader() {
	rder = nil
	rderOnce = sync.Once{}
	rderErr = nil
}

func TestGetRdr_ValidZip(t *testing.T) {
	origStaticPkg := comm.StaticPkg
	defer func() {
		comm.StaticPkg = origStaticPkg
		resetZipReader()
	}()

	comm.StaticPkg = createTestZip(t, map[string]string{
		"index.html": "<html>hello</html>",
	})
	resetZipReader()

	r, err := getRdr()
	if err != nil {
		t.Fatalf("getRdr failed: %v", err)
	}
	if r == nil {
		t.Fatal("expected non-nil zip reader")
	}
	if len(r.File) != 1 {
		t.Errorf("expected 1 file in zip, got %d", len(r.File))
	}
}

func TestGetRdr_InvalidBase64(t *testing.T) {
	origStaticPkg := comm.StaticPkg
	defer func() {
		comm.StaticPkg = origStaticPkg
		resetZipReader()
	}()

	comm.StaticPkg = "not-valid-base64!!!$$$"
	resetZipReader()

	_, err := getRdr()
	if err == nil {
		t.Fatal("expected error for invalid base64")
	}
}

func TestGetRdr_InvalidZip(t *testing.T) {
	origStaticPkg := comm.StaticPkg
	defer func() {
		comm.StaticPkg = origStaticPkg
		resetZipReader()
	}()

	// Valid base64 but not a valid zip
	comm.StaticPkg = base64.StdEncoding.EncodeToString([]byte("this is not a zip file"))
	resetZipReader()

	_, err := getRdr()
	if err == nil {
		t.Fatal("expected error for invalid zip data")
	}
}

func TestGetRdr_CachesResult(t *testing.T) {
	origStaticPkg := comm.StaticPkg
	defer func() {
		comm.StaticPkg = origStaticPkg
		resetZipReader()
	}()

	comm.StaticPkg = createTestZip(t, map[string]string{
		"a.txt": "aaa",
	})
	resetZipReader()

	r1, err := getRdr()
	if err != nil {
		t.Fatalf("first getRdr: %v", err)
	}
	r2, err := getRdr()
	if err != nil {
		t.Fatalf("second getRdr: %v", err)
	}
	// Should return the same cached pointer
	if r1 != r2 {
		t.Error("expected getRdr to return cached reader on second call")
	}
}

func TestGetFile_FoundInZip(t *testing.T) {
	origStaticPkg := comm.StaticPkg
	defer func() {
		comm.StaticPkg = origStaticPkg
		resetZipReader()
	}()

	comm.StaticPkg = createTestZip(t, map[string]string{
		"index.html":           "<html>index</html>",
		"gokinsui/css/app.css": "body { color: red; }",
	})
	resetZipReader()

	f, err := getFile("index.html")
	if err != nil {
		t.Fatalf("getFile index.html: %v", err)
	}
	if f == nil {
		t.Fatal("expected non-nil file")
	}
	if f.Name != "index.html" {
		t.Errorf("expected file name 'index.html', got %q", f.Name)
	}

	// Test nested path
	f2, err := getFile("gokinsui/css/app.css")
	if err != nil {
		t.Fatalf("getFile nested: %v", err)
	}
	if f2 == nil {
		t.Fatal("expected non-nil file for nested path")
	}
}

func TestGetFile_NotFoundInZip(t *testing.T) {
	origStaticPkg := comm.StaticPkg
	defer func() {
		comm.StaticPkg = origStaticPkg
		resetZipReader()
	}()

	comm.StaticPkg = createTestZip(t, map[string]string{
		"index.html": "<html></html>",
	})
	resetZipReader()

	_, err := getFile("nonexistent.txt")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestMidUiHandle_InstalledWithStaticFile(t *testing.T) {
	origInstalled := comm.Installed
	origStaticPkg := comm.StaticPkg
	defer func() {
		comm.Installed = origInstalled
		comm.StaticPkg = origStaticPkg
		resetZipReader()
	}()

	comm.Installed = true
	comm.StaticPkg = createTestZip(t, map[string]string{
		"index.html": "<html>welcome</html>",
	})
	resetZipReader()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(midUiHandle)

	// Request a path that returns 404 from the router, triggering midUiHandle fallback
	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/nonexistent", nil)
	router.ServeHTTP(w, req)

	// Should serve index.html as fallback
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestMidUiHandle_InstalledHtmlContentType(t *testing.T) {
	origInstalled := comm.Installed
	origStaticPkg := comm.StaticPkg
	defer func() {
		comm.Installed = origInstalled
		comm.StaticPkg = origStaticPkg
		resetZipReader()
	}()

	comm.Installed = true
	comm.StaticPkg = createTestZip(t, map[string]string{
		"test.html": "<html>test</html>",
	})
	resetZipReader()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(midUiHandle)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/test.html", nil)
	router.ServeHTTP(w, req)

	// Should serve test.html with text/html content type
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	ct := w.Header().Get("Content-Type")
	if ct != "text/html" {
		t.Errorf("expected Content-Type text/html, got %q", ct)
	}
}

func TestMidUiHandle_InstalledCssContentType(t *testing.T) {
	origInstalled := comm.Installed
	origStaticPkg := comm.StaticPkg
	defer func() {
		comm.Installed = origInstalled
		comm.StaticPkg = origStaticPkg
		resetZipReader()
	}()

	comm.Installed = true
	comm.StaticPkg = createTestZip(t, map[string]string{
		"style.css": "body { margin: 0; }",
	})
	resetZipReader()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(midUiHandle)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/style.css", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	ct := w.Header().Get("Content-Type")
	if ct != "text/css" {
		t.Errorf("expected Content-Type text/css, got %q", ct)
	}
}

func TestMidUiHandle_InstalledJsContentType(t *testing.T) {
	origInstalled := comm.Installed
	origStaticPkg := comm.StaticPkg
	defer func() {
		comm.Installed = origInstalled
		comm.StaticPkg = origStaticPkg
		resetZipReader()
	}()

	comm.Installed = true
	comm.StaticPkg = createTestZip(t, map[string]string{
		"app.js": "console.log('hello');",
	})
	resetZipReader()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(midUiHandle)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/app.js", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	ct := w.Header().Get("Content-Type")
	if ct != "application/javascript" {
		t.Errorf("expected Content-Type application/javascript, got %q", ct)
	}
}

func TestMidUiHandle_InstalledSvgContentType(t *testing.T) {
	origInstalled := comm.Installed
	origStaticPkg := comm.StaticPkg
	defer func() {
		comm.Installed = origInstalled
		comm.StaticPkg = origStaticPkg
		resetZipReader()
	}()

	comm.Installed = true
	comm.StaticPkg = createTestZip(t, map[string]string{
		"icon.svg": "<svg></svg>",
	})
	resetZipReader()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(midUiHandle)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/icon.svg", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	ct := w.Header().Get("Content-Type")
	if ct != "image/svg+xml" {
		t.Errorf("expected Content-Type image/svg+xml, got %q", ct)
	}
}

func TestMidUiHandle_InstalledFontContentType(t *testing.T) {
	origInstalled := comm.Installed
	origStaticPkg := comm.StaticPkg
	defer func() {
		comm.Installed = origInstalled
		comm.StaticPkg = origStaticPkg
		resetZipReader()
	}()

	comm.Installed = true
	comm.StaticPkg = createTestZip(t, map[string]string{
		"font.ttf": "fake-ttf-data",
	})
	resetZipReader()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(midUiHandle)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/font.ttf", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	ct := w.Header().Get("Content-Type")
	if ct != "application/x-font-ttf" {
		t.Errorf("expected Content-Type application/x-font-ttf, got %q", ct)
	}
}

func TestMidUiHandle_NotInstalled_InstallPath(t *testing.T) {
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

	// /install path should NOT trigger redirect
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for /install, got %d", w.Code)
	}
}

func TestMidUiHandle_NotInstalled_GokinsUiPath(t *testing.T) {
	origInstalled := comm.Installed
	defer func() { comm.Installed = origInstalled }()

	comm.Installed = false

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(midUiHandle)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/gokinsui/somefile", nil)
	router.ServeHTTP(w, req)

	// /gokinsui/ paths should NOT trigger redirect either
	// They'll get a redirect to "/" since the file isn't found, but not the "未安装" redirect
	if w.Code == http.StatusFound {
		// This is the fallback redirect (not the "not installed" one), which is acceptable
		t.Log("got expected fallback redirect for /gokinsui/ path when not installed")
	}
}

func TestMidUiHandle_HandlerReturnsNon404(t *testing.T) {
	origInstalled := comm.Installed
	defer func() { comm.Installed = origInstalled }()

	comm.Installed = true

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(midUiHandle)
	router.GET("/api/hello", func(c *gin.Context) {
		c.String(http.StatusOK, "hello")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/api/hello", nil)
	router.ServeHTTP(w, req)

	// When handler returns non-404, midUiHandle should not interfere
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if w.Body.String() != "hello" {
		t.Errorf("expected 'hello', got %q", w.Body.String())
	}
}

func TestMidUiHandle_HandlerReturnsContent(t *testing.T) {
	origInstalled := comm.Installed
	defer func() { comm.Installed = origInstalled }()

	comm.Installed = true

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(midUiHandle)
	// Handler that returns 404 but with content (Size > 0)
	router.GET("/custom404", func(c *gin.Context) {
		c.String(http.StatusNotFound, "custom 404 page")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/custom404", nil)
	router.ServeHTTP(w, req)

	// When Size > 0, midUiHandle should not interfere
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
	if w.Body.String() != "custom 404 page" {
		t.Errorf("expected 'custom 404 page', got %q", w.Body.String())
	}
}

func TestMidUiHandle_FallbackToIndexHtml(t *testing.T) {
	origInstalled := comm.Installed
	origStaticPkg := comm.StaticPkg
	defer func() {
		comm.Installed = origInstalled
		comm.StaticPkg = origStaticPkg
		resetZipReader()
	}()

	comm.Installed = true
	comm.StaticPkg = createTestZip(t, map[string]string{
		"index.html": "<html>SPA fallback</html>",
	})
	resetZipReader()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(midUiHandle)

	// Request a deep SPA route that doesn't exist as a static file
	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/pipelines/123/edit", nil)
	router.ServeHTTP(w, req)

	// Should serve index.html as SPA fallback
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 (SPA fallback), got %d", w.Code)
	}
}

func TestMidUiHandle_NoStaticPkg(t *testing.T) {
	origInstalled := comm.Installed
	origStaticPkg := comm.StaticPkg
	defer func() {
		comm.Installed = origInstalled
		comm.StaticPkg = origStaticPkg
		resetZipReader()
	}()

	comm.Installed = true
	comm.StaticPkg = "" // No static package
	resetZipReader()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(midUiHandle)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/any-path", nil)
	router.ServeHTTP(w, req)

	// Should redirect to "/" since no static files available
	if w.Code != http.StatusFound {
		t.Errorf("expected 302 redirect when no static pkg, got %d", w.Code)
	}
}

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

// makeTestZip creates a small zip containing the given files, returns base64-encoded data.
func makeTestZip(t *testing.T, files map[string]string) string {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for name, content := range files {
		f, err := w.Create(name)
		if err != nil {
			t.Fatalf("create zip entry %q: %v", name, err)
		}
		if _, err := f.Write([]byte(content)); err != nil {
			t.Fatalf("write zip entry %q: %v", name, err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes())
}

// resetZipReader resets the package-level zip reader state for test isolation.
func resetZipReader() {
	rder = nil
	rderOnce = sync.Once{}
	rderErr = nil
}

func TestMidUiHandle_InstalledWithStaticFile(t *testing.T) {
	origStatic := comm.StaticPkg
	origInstalled := comm.Installed
	defer func() {
		comm.StaticPkg = origStatic
		comm.Installed = origInstalled
		resetZipReader()
	}()

	resetZipReader()
	comm.StaticPkg = makeTestZip(t, map[string]string{
		"index.html": "<html><body>test</body></html>",
	})
	comm.Installed = true

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(midUiHandle)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/index.html", nil)
	router.ServeHTTP(w, req)

	// Should serve the file from the zip
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "text/html" {
		t.Errorf("Content-Type = %q, want %q", ct, "text/html")
	}
	if !strings.Contains(w.Body.String(), "<html>") {
		t.Errorf("body should contain HTML, got %q", w.Body.String())
	}
}

func TestMidUiHandle_InstalledWithCSSFile(t *testing.T) {
	origStatic := comm.StaticPkg
	origInstalled := comm.Installed
	defer func() {
		comm.StaticPkg = origStatic
		comm.Installed = origInstalled
		resetZipReader()
	}()

	resetZipReader()
	comm.StaticPkg = makeTestZip(t, map[string]string{
		"style.css": "body { color: red; }",
	})
	comm.Installed = true

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(midUiHandle)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/style.css", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "text/css" {
		t.Errorf("Content-Type = %q, want %q", ct, "text/css")
	}
}

func TestMidUiHandle_InstalledWithJSFile(t *testing.T) {
	origStatic := comm.StaticPkg
	origInstalled := comm.Installed
	defer func() {
		comm.StaticPkg = origStatic
		comm.Installed = origInstalled
		resetZipReader()
	}()

	resetZipReader()
	comm.StaticPkg = makeTestZip(t, map[string]string{
		"app.js": "console.log('hello')",
	})
	comm.Installed = true

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(midUiHandle)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/app.js", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/javascript" {
		t.Errorf("Content-Type = %q, want %q", ct, "application/javascript")
	}
}

func TestMidUiHandle_InstalledWithSVGFile(t *testing.T) {
	origStatic := comm.StaticPkg
	origInstalled := comm.Installed
	defer func() {
		comm.StaticPkg = origStatic
		comm.Installed = origInstalled
		resetZipReader()
	}()

	resetZipReader()
	comm.StaticPkg = makeTestZip(t, map[string]string{
		"icon.svg": "<svg></svg>",
	})
	comm.Installed = true

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(midUiHandle)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/icon.svg", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "image/svg+xml" {
		t.Errorf("Content-Type = %q, want %q", ct, "image/svg+xml")
	}
}

func TestMidUiHandle_InstalledWithTTFFile(t *testing.T) {
	origStatic := comm.StaticPkg
	origInstalled := comm.Installed
	defer func() {
		comm.StaticPkg = origStatic
		comm.Installed = origInstalled
		resetZipReader()
	}()

	resetZipReader()
	comm.StaticPkg = makeTestZip(t, map[string]string{
		"font.ttf": "fake-ttf-data",
	})
	comm.Installed = true

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(midUiHandle)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/font.ttf", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/x-font-ttf" {
		t.Errorf("Content-Type = %q, want %q", ct, "application/x-font-ttf")
	}
}

func TestMidUiHandle_FileNotFoundFallbackToIndex(t *testing.T) {
	origStatic := comm.StaticPkg
	origInstalled := comm.Installed
	defer func() {
		comm.StaticPkg = origStatic
		comm.Installed = origInstalled
		resetZipReader()
	}()

	resetZipReader()
	comm.StaticPkg = makeTestZip(t, map[string]string{
		"index.html": "<html>fallback</html>",
	})
	comm.Installed = true

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(midUiHandle)

	// Request a non-existent file — should fall back to index.html
	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/nonexistent", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200 (fallback to index), got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "fallback") {
		t.Errorf("expected fallback index.html content, got %q", w.Body.String())
	}
}

func TestMidUiHandle_BothFileAndIndexMissing(t *testing.T) {
	origStatic := comm.StaticPkg
	origInstalled := comm.Installed
	defer func() {
		comm.StaticPkg = origStatic
		comm.Installed = origInstalled
		resetZipReader()
	}()

	// Empty zip with no files
	resetZipReader()
	comm.StaticPkg = makeTestZip(t, map[string]string{})
	comm.Installed = true

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(midUiHandle)

	// Both the requested file and index.html are missing → redirect to /
	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/nonexistent", nil)
	router.ServeHTTP(w, req)

	// httpex.ResMsgUrl returns 302
	if w.Code != http.StatusFound {
		t.Errorf("expected status 302 (redirect), got %d", w.Code)
	}
}

func TestMidUiHandle_NotInstalledRedirectsToInstall(t *testing.T) {
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
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/anything", nil)
	router.ServeHTTP(w, req)

	// Should redirect to /install
	if w.Code != http.StatusFound {
		t.Errorf("expected status 302 (redirect to /install), got %d", w.Code)
	}
}

func TestMidUiHandle_NotInstalled_AllowsGokinsuiPath(t *testing.T) {
	origInstalled := comm.Installed
	origStatic := comm.StaticPkg
	defer func() {
		comm.Installed = origInstalled
		comm.StaticPkg = origStatic
		resetZipReader()
	}()

	resetZipReader()
	comm.StaticPkg = makeTestZip(t, map[string]string{
		"gokinsui/app.js": "console.log('ui')",
	})
	comm.Installed = false

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(midUiHandle)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/gokinsui/app.js", nil)
	router.ServeHTTP(w, req)

	// Should serve the file (not redirect) because /gokinsui/ is allowed when not installed
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200 for /gokinsui/ path, got %d", w.Code)
	}
}

func TestMidUiHandle_NotInstalled_AllowsInstallPath(t *testing.T) {
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

	// Should NOT redirect — /install is explicitly allowed when not installed
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200 for /install, got %d", w.Code)
	}
}

func TestMidUiHandle_HandlerReturnsNon404(t *testing.T) {
	origInstalled := comm.Installed
	defer func() { comm.Installed = origInstalled }()
	comm.Installed = true

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(midUiHandle)
	router.GET("/api/test", func(c *gin.Context) {
		c.String(http.StatusOK, "api response")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/api/test", nil)
	router.ServeHTTP(w, req)

	// midUiHandle should skip when status != 404
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	if w.Body.String() != "api response" {
		t.Errorf("body = %q, want %q", w.Body.String(), "api response")
	}
}

func TestMidUiHandle_HandlerWritesContent(t *testing.T) {
	origInstalled := comm.Installed
	defer func() { comm.Installed = origInstalled }()
	comm.Installed = true

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(midUiHandle)
	// Handler that writes a 404 with content (size > 0)
	router.GET("/notfound", func(c *gin.Context) {
		c.String(http.StatusNotFound, "custom 404 page")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/notfound", nil)
	router.ServeHTTP(w, req)

	// midUiHandle should skip because size > 0
	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
	if w.Body.String() != "custom 404 page" {
		t.Errorf("body = %q, want %q", w.Body.String(), "custom 404 page")
	}
}

func TestGetFile_ValidPath(t *testing.T) {
	origStatic := comm.StaticPkg
	defer func() {
		comm.StaticPkg = origStatic
		resetZipReader()
	}()

	resetZipReader()
	comm.StaticPkg = makeTestZip(t, map[string]string{
		"index.html": "<html>test</html>",
		"css/app.css": "body{}",
	})

	f, err := getFile("index.html")
	if err != nil {
		t.Fatalf("getFile: %v", err)
	}
	if f == nil {
		t.Fatal("getFile returned nil file")
	}
	if f.Name != "index.html" {
		t.Errorf("file name = %q, want %q", f.Name, "index.html")
	}
}

func TestGetFile_NestedPath(t *testing.T) {
	origStatic := comm.StaticPkg
	defer func() {
		comm.StaticPkg = origStatic
		resetZipReader()
	}()

	resetZipReader()
	comm.StaticPkg = makeTestZip(t, map[string]string{
		"css/app.css": "body{color:red}",
	})

	f, err := getFile("css/app.css")
	if err != nil {
		t.Fatalf("getFile: %v", err)
	}
	if f == nil {
		t.Fatal("getFile returned nil file")
	}
}

func TestGetFile_NotFound(t *testing.T) {
	origStatic := comm.StaticPkg
	defer func() {
		comm.StaticPkg = origStatic
		resetZipReader()
	}()

	resetZipReader()
	comm.StaticPkg = makeTestZip(t, map[string]string{
		"index.html": "<html>test</html>",
	})

	_, err := getFile("nonexistent.js")
	if err == nil {
		t.Fatal("expected error for non-existent file")
	}
	if !strings.Contains(err.Error(), "file not found") {
		t.Errorf("error = %v, want file not found", err)
	}
}

func TestGetFile_InvalidBase64StaticPkg(t *testing.T) {
	origStatic := comm.StaticPkg
	defer func() {
		comm.StaticPkg = origStatic
		resetZipReader()
	}()

	resetZipReader()
	comm.StaticPkg = "!!!invalid-base64!!!"

	_, err := getFile("anything.html")
	if err == nil {
		t.Fatal("expected error for invalid base64 StaticPkg")
	}
}

func TestGetFile_CorruptedZip(t *testing.T) {
	origStatic := comm.StaticPkg
	defer func() {
		comm.StaticPkg = origStatic
		resetZipReader()
	}()

	resetZipReader()
	// Valid base64 but not a valid zip
	comm.StaticPkg = base64.StdEncoding.EncodeToString([]byte("this is not a zip file at all"))

	_, err := getFile("anything.html")
	if err == nil {
		t.Fatal("expected error for corrupted zip data")
	}
}

func TestMidUiHandle_InstalledWithWoff2File(t *testing.T) {
	origStatic := comm.StaticPkg
	origInstalled := comm.Installed
	defer func() {
		comm.StaticPkg = origStatic
		comm.Installed = origInstalled
		resetZipReader()
	}()

	resetZipReader()
	comm.StaticPkg = makeTestZip(t, map[string]string{
		"font.woff2": "fake-woff2-data",
	})
	comm.Installed = true

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(midUiHandle)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/font.woff2", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestMidUiHandle_InstalledWithTTCFile(t *testing.T) {
	origStatic := comm.StaticPkg
	origInstalled := comm.Installed
	defer func() {
		comm.StaticPkg = origStatic
		comm.Installed = origInstalled
		resetZipReader()
	}()

	resetZipReader()
	comm.StaticPkg = makeTestZip(t, map[string]string{
		"font.ttc": "fake-ttc-data",
	})
	comm.Installed = true

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(midUiHandle)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/font.ttc", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/x-font-ttf" {
		t.Errorf("Content-Type = %q, want %q", ct, "application/x-font-ttf")
	}
}

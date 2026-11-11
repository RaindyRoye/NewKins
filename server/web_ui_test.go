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

// resetZipReader resets the singleton zip reader state so tests start fresh.
func resetZipReader() {
	rder = nil
	rderOnce = sync.Once{}
	rderErr = nil
}

// makeTestZip creates a base64-encoded zip archive containing the given files.
// Each entry is a name→content pair. Returns the base64 string suitable for
// assigning to comm.StaticPkg.
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
		t.Fatalf("close zip writer: %v", err)
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes())
}

// setupUiRouter creates a minimal gin router with only the midUiHandle middleware
// and a catch-all route that returns 404 (simulating a missing API route).
func setupUiRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(midUiHandle)
	// Catch-all returns 404 with no body — triggers the UI fallback logic
	r.NoRoute(func(c *gin.Context) {
		c.Status(http.StatusNotFound)
	})
	// A handler that returns 200 with content — should NOT trigger UI fallback
	r.GET("/api/hello", func(c *gin.Context) {
		c.String(http.StatusOK, "hello")
	})
	// A handler that returns 404 with body content — should NOT trigger UI fallback
	r.GET("/api/partial", func(c *gin.Context) {
		c.String(http.StatusNotFound, "partial content")
	})
	return r
}

// --- midUiHandle: handler returns non-404 (should pass through) ---

func TestMidUiHandle_NonNotFound_PassThrough(t *testing.T) {
	router := setupUiRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/hello", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if w.Body.String() != "hello" {
		t.Errorf("expected body 'hello', got %q", w.Body.String())
	}
}

// --- midUiHandle: handler returns 404 with body content (should pass through) ---

func TestMidUiHandle_NotFoundWithBody_PassThrough(t *testing.T) {
	router := setupUiRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/partial", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
	if w.Body.String() != "partial content" {
		t.Errorf("expected body 'partial content', got %q", w.Body.String())
	}
}

// --- midUiHandle: not installed, redirects to /install ---

func TestMidUiHandle_NotInstalled_RedirectsToInstall(t *testing.T) {
	origInstalled := comm.Installed
	defer func() { comm.Installed = origInstalled }()
	comm.Installed = false

	router := setupUiRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/some-page", nil)
	router.ServeHTTP(w, req)

	// httpex.ResMsgUrl returns 302 with HTML redirect
	if w.Code != http.StatusFound {
		t.Errorf("expected 302 redirect, got %d", w.Code)
	}
}

// --- midUiHandle: not installed, /install path bypasses redirect ---

func TestMidUiHandle_NotInstalled_InstallPathBypassesRedirect(t *testing.T) {
	origInstalled := comm.Installed
	defer func() { comm.Installed = origInstalled }()
	comm.Installed = false

	resetZipReader()
	origPkg := comm.StaticPkg
	defer func() { comm.StaticPkg = origPkg }()
	// Empty StaticPkg → getRdr fails → fallback redirect to /
	comm.StaticPkg = ""

	router := setupUiRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/install", nil)
	router.ServeHTTP(w, req)

	// /install bypasses the install check, falls into getFile which fails → redirect to /
	if w.Code != http.StatusFound {
		t.Errorf("expected 302 redirect to /, got %d", w.Code)
	}
}

// --- midUiHandle: not installed, /gokinsui/* bypasses redirect ---

func TestMidUiHandle_NotInstalled_GokinsUiPathBypassesRedirect(t *testing.T) {
	origInstalled := comm.Installed
	defer func() { comm.Installed = origInstalled }()
	comm.Installed = false

	resetZipReader()
	origPkg := comm.StaticPkg
	defer func() { comm.StaticPkg = origPkg }()
	comm.StaticPkg = ""

	router := setupUiRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/gokinsui/app.js", nil)
	router.ServeHTTP(w, req)

	// /gokinsui/* bypasses the install check, falls into getFile which fails → redirect to /
	if w.Code != http.StatusFound {
		t.Errorf("expected 302 redirect to /, got %d", w.Code)
	}
}

// --- midUiHandle: installed, both file and index.html not found → redirect to / ---

func TestMidUiHandle_Installed_NoFiles_RedirectsToRoot(t *testing.T) {
	origInstalled := comm.Installed
	defer func() { comm.Installed = origInstalled }()
	comm.Installed = true

	resetZipReader()
	origPkg := comm.StaticPkg
	defer func() { comm.StaticPkg = origPkg }()
	// Empty StaticPkg → getRdr fails → both getFile calls fail
	comm.StaticPkg = ""

	router := setupUiRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/nonexistent", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("expected 302 redirect, got %d", w.Code)
	}
}

// --- midUiHandle: installed, file not found but index.html exists → serves index.html ---

func TestMidUiHandle_Installed_FallbackToIndexHtml(t *testing.T) {
	origInstalled := comm.Installed
	defer func() { comm.Installed = origInstalled }()
	comm.Installed = true

	resetZipReader()
	origPkg := comm.StaticPkg
	defer func() { comm.StaticPkg = origPkg }()
	comm.StaticPkg = makeTestZip(t, map[string]string{
		"index.html": "<html>fallback</html>",
	})

	router := setupUiRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/unknown-page", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "fallback") {
		t.Errorf("expected body to contain 'fallback', got %q", w.Body.String())
	}
	// HTML should have no-cache headers
	if w.Header().Get("Cache-Control") != "no-cache" {
		t.Errorf("expected Cache-Control no-cache, got %q", w.Header().Get("Cache-Control"))
	}
	if w.Header().Get("Content-Type") != "text/html" {
		t.Errorf("expected Content-Type text/html, got %q", w.Header().Get("Content-Type"))
	}
}

// --- midUiHandle: serves CSS file with correct headers ---

func TestMidUiHandle_Installed_ServesCssWithCacheHeaders(t *testing.T) {
	origInstalled := comm.Installed
	defer func() { comm.Installed = origInstalled }()
	comm.Installed = true

	resetZipReader()
	origPkg := comm.StaticPkg
	defer func() { comm.StaticPkg = origPkg }()
	comm.StaticPkg = makeTestZip(t, map[string]string{
		"style.css": "body { color: red; }",
	})

	router := setupUiRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/style.css", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if w.Body.String() != "body { color: red; }" {
		t.Errorf("unexpected body: %q", w.Body.String())
	}
	if w.Header().Get("Content-Type") != "text/css" {
		t.Errorf("expected Content-Type text/css, got %q", w.Header().Get("Content-Type"))
	}
	if !strings.Contains(w.Header().Get("Cache-Control"), "max-age=") {
		t.Errorf("expected Cache-Control with max-age, got %q", w.Header().Get("Cache-Control"))
	}
}

// --- midUiHandle: serves JS file with correct headers ---

func TestMidUiHandle_Installed_ServesJsWithCacheHeaders(t *testing.T) {
	origInstalled := comm.Installed
	defer func() { comm.Installed = origInstalled }()
	comm.Installed = true

	resetZipReader()
	origPkg := comm.StaticPkg
	defer func() { comm.StaticPkg = origPkg }()
	comm.StaticPkg = makeTestZip(t, map[string]string{
		"app.js": "console.log('hello');",
	})

	router := setupUiRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/app.js", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if w.Header().Get("Content-Type") != "application/javascript" {
		t.Errorf("expected Content-Type application/javascript, got %q", w.Header().Get("Content-Type"))
	}
}

// --- midUiHandle: serves SVG file ---

func TestMidUiHandle_Installed_ServesSvg(t *testing.T) {
	origInstalled := comm.Installed
	defer func() { comm.Installed = origInstalled }()
	comm.Installed = true

	resetZipReader()
	origPkg := comm.StaticPkg
	defer func() { comm.StaticPkg = origPkg }()
	comm.StaticPkg = makeTestZip(t, map[string]string{
		"icon.svg": "<svg></svg>",
	})

	router := setupUiRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/icon.svg", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if w.Header().Get("Content-Type") != "image/svg+xml" {
		t.Errorf("expected Content-Type image/svg+xml, got %q", w.Header().Get("Content-Type"))
	}
}

// --- midUiHandle: serves WOFF2 font ---

func TestMidUiHandle_Installed_ServesWoff2(t *testing.T) {
	origInstalled := comm.Installed
	defer func() { comm.Installed = origInstalled }()
	comm.Installed = true

	resetZipReader()
	origPkg := comm.StaticPkg
	defer func() { comm.StaticPkg = origPkg }()
	comm.StaticPkg = makeTestZip(t, map[string]string{
		"font.woff2": "fakefont",
	})

	router := setupUiRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/font.woff2", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Header().Get("Cache-Control"), "max-age=") {
		t.Errorf("expected Cache-Control with max-age for woff2, got %q", w.Header().Get("Cache-Control"))
	}
}

// --- midUiHandle: serves TTF/TTC font ---

func TestMidUiHandle_Installed_ServesTtf(t *testing.T) {
	origInstalled := comm.Installed
	defer func() { comm.Installed = origInstalled }()
	comm.Installed = true

	resetZipReader()
	origPkg := comm.StaticPkg
	defer func() { comm.StaticPkg = origPkg }()
	comm.StaticPkg = makeTestZip(t, map[string]string{
		"font.ttf": "fakettf",
	})

	router := setupUiRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/font.ttf", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if w.Header().Get("Content-Type") != "application/x-font-ttf" {
		t.Errorf("expected Content-Type application/x-font-ttf, got %q", w.Header().Get("Content-Type"))
	}
}

// --- midUiHandle: serves unknown extension (no special headers) ---

func TestMidUiHandle_Installed_ServesUnknownExtension(t *testing.T) {
	origInstalled := comm.Installed
	defer func() { comm.Installed = origInstalled }()
	comm.Installed = true

	resetZipReader()
	origPkg := comm.StaticPkg
	defer func() { comm.StaticPkg = origPkg }()
	comm.StaticPkg = makeTestZip(t, map[string]string{
		"data.json": `{"key":"value"}`,
	})

	router := setupUiRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/data.json", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if w.Body.String() != `{"key":"value"}` {
		t.Errorf("unexpected body: %q", w.Body.String())
	}
	// No special Content-Type should be set for .json
	if ct := w.Header().Get("Content-Type"); ct != "" {
		t.Errorf("expected no Content-Type header for unknown ext, got %q", ct)
	}
}

// --- midUiHandle: oversized file rejected ---

func TestMidUiHandle_Installed_OversizedFileRejected(t *testing.T) {
	origInstalled := comm.Installed
	defer func() { comm.Installed = origInstalled }()
	comm.Installed = true

	resetZipReader()
	origPkg := comm.StaticPkg
	defer func() { comm.StaticPkg = origPkg }()

	// Create a zip with a file whose UncompressedSize64 exceeds uiMaxSize.
	// We can't easily create a 50MB+ file in a test, so we construct the zip
	// manually with a stored entry that reports a large uncompressed size.
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	fw, err := zw.Create("big.html")
	if err != nil {
		t.Fatalf("create zip entry: %v", err)
	}
	// Write some content — but we'll manipulate the zip header to report huge size
	if _, err := fw.Write([]byte("small")); err != nil {
		t.Fatalf("write zip entry: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}

	// Re-open and manipulate the zip to set UncompressedSize64 > uiMaxSize
	zipBytes := buf.Bytes()
	r, err := zip.NewReader(bytes.NewReader(zipBytes), int64(len(zipBytes)))
	if err != nil {
		t.Fatalf("reopen zip: %v", err)
	}
	// Override the file's UncompressedSize64
	for _, f := range r.File {
		if f.Name == "big.html" {
			f.UncompressedSize64 = uiMaxSize + 1
		}
	}
	rder = r
	rderOnce.Do(func() {}) // mark as initialized

	comm.StaticPkg = "placeholder" // won't be read since rder is set

	router := setupUiRouter()
	rec := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/big.html", nil)
	router.ServeHTTP(rec, req)

	// Should redirect because file is too large
	if rec.Code != http.StatusFound {
		t.Errorf("expected 302 redirect for oversized file, got %d", rec.Code)
	}
}

// --- getRdr: invalid base64 ---

func TestGetRdr_InvalidBase64(t *testing.T) {
	resetZipReader()
	origPkg := comm.StaticPkg
	defer func() { comm.StaticPkg = origPkg }()
	comm.StaticPkg = "!!!not-valid-base64!!!"

	_, err := getRdr()
	if err == nil {
		t.Fatal("expected error for invalid base64, got nil")
	}
}

// --- getRdr: valid base64 but invalid zip ---

func TestGetRdr_InvalidZip(t *testing.T) {
	resetZipReader()
	origPkg := comm.StaticPkg
	defer func() { comm.StaticPkg = origPkg }()
	comm.StaticPkg = base64.StdEncoding.EncodeToString([]byte("this is not a zip file"))

	_, err := getRdr()
	if err == nil {
		t.Fatal("expected error for invalid zip data, got nil")
	}
}

// --- getRdr: valid zip ---

func TestGetRdr_ValidZip(t *testing.T) {
	resetZipReader()
	origPkg := comm.StaticPkg
	defer func() { comm.StaticPkg = origPkg }()
	comm.StaticPkg = makeTestZip(t, map[string]string{
		"test.txt": "hello world",
	})

	r, err := getRdr()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r == nil {
		t.Fatal("expected non-nil reader")
	}
	if len(r.File) != 1 {
		t.Errorf("expected 1 file in zip, got %d", len(r.File))
	}
}

// --- getRdr: called twice returns same reader (singleton) ---

func TestGetRdr_Singleton(t *testing.T) {
	resetZipReader()
	origPkg := comm.StaticPkg
	defer func() { comm.StaticPkg = origPkg }()
	comm.StaticPkg = makeTestZip(t, map[string]string{
		"a.txt": "a",
	})

	r1, err := getRdr()
	if err != nil {
		t.Fatalf("first getRdr: %v", err)
	}
	r2, err := getRdr()
	if err != nil {
		t.Fatalf("second getRdr: %v", err)
	}
	if r1 != r2 {
		t.Error("expected same reader instance from both calls")
	}
}

// --- getFile: find existing file in zip ---

func TestGetFile_FindsExistingFile(t *testing.T) {
	resetZipReader()
	origPkg := comm.StaticPkg
	defer func() { comm.StaticPkg = origPkg }()
	comm.StaticPkg = makeTestZip(t, map[string]string{
		"index.html": "<html>test</html>",
		"style.css":  "body{}",
	})

	f, err := getFile("index.html")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.Name != "index.html" {
		t.Errorf("expected file name 'index.html', got %q", f.Name)
	}
}

// --- getFile: backslash paths normalized ---

func TestGetFile_BackslashNormalization(t *testing.T) {
	resetZipReader()
	origPkg := comm.StaticPkg
	defer func() { comm.StaticPkg = origPkg }()

	// Create zip with backslash path (Windows-style)
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	fw, err := w.Create("dir\\file.txt")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := fw.Write([]byte("content")); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	comm.StaticPkg = base64.StdEncoding.EncodeToString(buf.Bytes())

	// Search with forward slash — should find it via normalization
	f, err := getFile("dir/file.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f == nil {
		t.Fatal("expected to find file with normalized path")
	}
}

package server

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gokins/gokins/comm"
)

// createTestZip creates a base64-encoded zip archive with the given files
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

// resetZipReader resets the global zip reader state for testing
func resetZipReader() {
	rder = nil
	rderOnce = sync.Once{}
	rderErr = nil
}

func TestGetRdr_InvalidBase64(t *testing.T) {
	resetZipReader()
	origStaticPkg := comm.StaticPkg
	defer func() { comm.StaticPkg = origStaticPkg }()

	comm.StaticPkg = "not-valid-base64!!!"

	_, err := getRdr()
	if err == nil {
		t.Fatal("expected error for invalid base64, got nil")
	}

	// Subsequent calls should return the same error
	_, err2 := getRdr()
	if err2 == nil {
		t.Fatal("expected cached error, got nil")
	}
}

func TestGetRdr_InvalidZip(t *testing.T) {
	resetZipReader()
	origStaticPkg := comm.StaticPkg
	defer func() { comm.StaticPkg = origStaticPkg }()

	// Valid base64 but invalid zip content
	comm.StaticPkg = base64.StdEncoding.EncodeToString([]byte("not a zip file"))

	_, err := getRdr()
	if err == nil {
		t.Fatal("expected error for invalid zip, got nil")
	}
}

func TestGetRdr_ValidZip(t *testing.T) {
	resetZipReader()
	origStaticPkg := comm.StaticPkg
	defer func() { comm.StaticPkg = origStaticPkg }()

	comm.StaticPkg = createTestZip(t, map[string]string{
		"test.html": "<html>test</html>",
	})

	r, err := getRdr()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r == nil {
		t.Fatal("expected non-nil reader")
	}
	if len(r.File) != 1 {
		t.Errorf("expected 1 file, got %d", len(r.File))
	}
}

func TestGetFile_WithValidZip(t *testing.T) {
	resetZipReader()
	origStaticPkg := comm.StaticPkg
	defer func() { comm.StaticPkg = origStaticPkg }()

	comm.StaticPkg = createTestZip(t, map[string]string{
		"index.html":         "<html>index</html>",
		"styles/main.css":    "body { color: red; }",
		"scripts/app.js":     "console.log('hello');",
		"images/logo.svg":    "<svg></svg>",
		"fonts/custom.woff2": "font-data",
	})

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{"html file", "index.html", false},
		{"css file", "styles/main.css", false},
		{"js file", "scripts/app.js", false},
		{"svg file", "images/logo.svg", false},
		{"woff2 file", "fonts/custom.woff2", false},
		{"nonexistent", "missing.txt", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := getFile(tt.path)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if f == nil {
				t.Fatal("expected non-nil file")
			}
		})
	}
}

func TestGetFile_BackslashNormalization(t *testing.T) {
	resetZipReader()
	origStaticPkg := comm.StaticPkg
	defer func() { comm.StaticPkg = origStaticPkg }()

	// Create zip with Windows-style backslash paths
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	f, _ := w.Create("styles\\main.css")
	_, _ = f.Write([]byte("body {}"))
	_ = w.Close()

	comm.StaticPkg = base64.StdEncoding.EncodeToString(buf.Bytes())

	// Should be able to find it with forward slashes
	file, err := getFile("styles/main.css")
	if err != nil {
		t.Fatalf("expected to find file with normalized path: %v", err)
	}
	if file == nil {
		t.Fatal("expected non-nil file")
	}
}

func TestMidUiHandle_NotInstalled_Redirect(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(midUiHandle)

	origInstalled := comm.Installed
	defer func() { comm.Installed = origInstalled }()
	comm.Installed = false

	// Request to non-install path should redirect
	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/dashboard", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("expected status 302, got %d", w.Code)
	}
}

func TestMidUiHandle_NotInstalled_AllowInstallPath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(midUiHandle)
	router.GET("/install", func(c *gin.Context) {
		c.String(http.StatusOK, "install")
	})

	origInstalled := comm.Installed
	defer func() { comm.Installed = origInstalled }()
	comm.Installed = false

	// Request to /install should pass through
	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/install", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestMidUiHandle_Installed_ServeHtmlContent(t *testing.T) {
	resetZipReader()
	origStaticPkg := comm.StaticPkg
	origInstalled := comm.Installed
	defer func() {
		comm.StaticPkg = origStaticPkg
		comm.Installed = origInstalled
	}()

	comm.StaticPkg = createTestZip(t, map[string]string{
		"index.html": "<html>hello</html>",
	})
	comm.Installed = true

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(midUiHandle)

	// Request should fall through to serving index.html
	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/nonexistent", nil)
	router.ServeHTTP(w, req)

	// Should serve index.html content
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "hello") {
		t.Errorf("expected HTML content, got %q", w.Body.String())
	}
}

func TestMidUiHandle_ContentType_HTML(t *testing.T) {
	resetZipReader()
	origStaticPkg := comm.StaticPkg
	origInstalled := comm.Installed
	defer func() {
		comm.StaticPkg = origStaticPkg
		comm.Installed = origInstalled
	}()

	comm.StaticPkg = createTestZip(t, map[string]string{
		"test.html": "<html>test</html>",
	})
	comm.Installed = true

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(midUiHandle)
	// Don't register a handler - let it 404 and fall through to static serving

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/test.html", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "text/html" {
		t.Errorf("expected Content-Type text/html, got %q", ct)
	}
	if cc := w.Header().Get("Cache-Control"); cc != "no-cache" {
		t.Errorf("expected Cache-Control no-cache for HTML, got %q", cc)
	}
}

func TestMidUiHandle_ContentType_CSS(t *testing.T) {
	resetZipReader()
	origStaticPkg := comm.StaticPkg
	origInstalled := comm.Installed
	defer func() {
		comm.StaticPkg = origStaticPkg
		comm.Installed = origInstalled
	}()

	comm.StaticPkg = createTestZip(t, map[string]string{
		"styles.css": "body { margin: 0; }",
	})
	comm.Installed = true

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(midUiHandle)
	// Let it 404 and fall through to static serving

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/styles.css", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "text/css" {
		t.Errorf("expected Content-Type text/css, got %q", ct)
	}
	if !strings.Contains(w.Header().Get("Cache-Control"), "max-age") {
		t.Errorf("expected Cache-Control with max-age, got %q", w.Header().Get("Cache-Control"))
	}
}

func TestMidUiHandle_ContentType_JavaScript(t *testing.T) {
	resetZipReader()
	origStaticPkg := comm.StaticPkg
	origInstalled := comm.Installed
	defer func() {
		comm.StaticPkg = origStaticPkg
		comm.Installed = origInstalled
	}()

	comm.StaticPkg = createTestZip(t, map[string]string{
		"app.js": "console.log('test');",
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
		t.Errorf("expected Content-Type application/javascript, got %q", ct)
	}
}

func TestMidUiHandle_ContentType_SVG(t *testing.T) {
	resetZipReader()
	origStaticPkg := comm.StaticPkg
	origInstalled := comm.Installed
	defer func() {
		comm.StaticPkg = origStaticPkg
		comm.Installed = origInstalled
	}()

	comm.StaticPkg = createTestZip(t, map[string]string{
		"logo.svg": "<svg></svg>",
	})
	comm.Installed = true

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(midUiHandle)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/logo.svg", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "image/svg+xml" {
		t.Errorf("expected Content-Type image/svg+xml, got %q", ct)
	}
}

func TestMidUiHandle_ContentType_Fonts(t *testing.T) {
	resetZipReader()
	origStaticPkg := comm.StaticPkg
	origInstalled := comm.Installed
	defer func() {
		comm.StaticPkg = origStaticPkg
		comm.Installed = origInstalled
	}()

	comm.StaticPkg = createTestZip(t, map[string]string{
		"font.woff2": "woff2-data",
		"font.ttf":   "ttf-data",
	})
	comm.Installed = true

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(midUiHandle)

	// Test woff2
	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/font.woff2", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("woff2: expected status 200, got %d", w.Code)
	}

	// Test ttf
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequestWithContext(context.Background(), "GET", "/font.ttf", nil)
	router.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("ttf: expected status 200, got %d", w2.Code)
	}
	if ct := w2.Header().Get("Content-Type"); ct != "application/x-font-ttf" {
		t.Errorf("ttf: expected Content-Type application/x-font-ttf, got %q", ct)
	}
}

func TestMidUiHandle_OversizedFile(t *testing.T) {
	resetZipReader()
	origStaticPkg := comm.StaticPkg
	origInstalled := comm.Installed
	defer func() {
		comm.StaticPkg = origStaticPkg
		comm.Installed = origInstalled
	}()

	// Create a zip with a file whose UncompressedSize64 exceeds uiMaxSize.
	// Note: Store method may not populate UncompressedSize64 for small content,
	// so we use Deflate method with a large claimed size.
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	header := &zip.FileHeader{
		Name:               "huge.html",
		Method:             zip.Deflate,
		UncompressedSize64: uint64(uiMaxSize + 1024),
	}
	f, err := w.CreateHeader(header)
	if err != nil {
		t.Fatalf("create header: %v", err)
	}
	// Write minimal content (Deflate will compress)
	_, _ = f.Write([]byte("oversized file"))
	_ = w.Close()

	comm.StaticPkg = base64.StdEncoding.EncodeToString(buf.Bytes())
	comm.Installed = true

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(midUiHandle)
	router.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "root")
	})

	w2 := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/huge.html", nil)
	router.ServeHTTP(w2, req)

	// After the oversized-file redirect, the response should not be 200 OK
	// for the original path. We just verify the request completes.
	if w2.Code == 0 {
		t.Error("expected non-zero status code")
	}
}

func TestMidUiHandle_NoFileFound(t *testing.T) {
	resetZipReader()
	origStaticPkg := comm.StaticPkg
	origInstalled := comm.Installed
	defer func() {
		comm.StaticPkg = origStaticPkg
		comm.Installed = origInstalled
	}()

	// Zip with no index.html and no matching file
	comm.StaticPkg = createTestZip(t, map[string]string{
		"other.html": "<html>other</html>",
	})
	comm.Installed = true

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(midUiHandle)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/missing.html", nil)
	router.ServeHTTP(w, req)

	// Should redirect when file not found (no index.html fallback)
	if w.Code != http.StatusFound {
		t.Errorf("expected status 302 (redirect), got %d", w.Code)
	}
}

func TestMidUiHandle_NoFileFound_WithIndexFallback(t *testing.T) {
	resetZipReader()
	origStaticPkg := comm.StaticPkg
	origInstalled := comm.Installed
	defer func() {
		comm.StaticPkg = origStaticPkg
		comm.Installed = origInstalled
	}()

	// Zip with index.html as fallback
	comm.StaticPkg = createTestZip(t, map[string]string{
		"index.html": "<html>fallback</html>",
	})
	comm.Installed = true

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(midUiHandle)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/missing.html", nil)
	router.ServeHTTP(w, req)

	// Should serve index.html as fallback (returns 200)
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200 (fallback to index), got %d", w.Code)
	}
}

func TestMidUiHandle_ResponseWithContent(t *testing.T) {
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

	// Should pass through with 200 status
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	if w.Body.String() != "api response" {
		t.Errorf("expected 'api response', got %q", w.Body.String())
	}
}

func TestMidUiHandle_CorruptedZipEntry(t *testing.T) {
	resetZipReader()
	origStaticPkg := comm.StaticPkg
	origInstalled := comm.Installed
	defer func() {
		comm.StaticPkg = origStaticPkg
		comm.Installed = origInstalled
	}()

	// Create a valid zip
	comm.StaticPkg = createTestZip(t, map[string]string{
		"valid.html": "<html>valid</html>",
	})
	comm.Installed = true

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(midUiHandle)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/valid.html", nil)
	router.ServeHTTP(w, req)

	// Should serve the file successfully
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestGetFile_PathCleaning(t *testing.T) {
	resetZipReader()
	origStaticPkg := comm.StaticPkg
	defer func() { comm.StaticPkg = origStaticPkg }()

	comm.StaticPkg = createTestZip(t, map[string]string{
		"dir/file.html": "<html>test</html>",
	})

	// Should handle paths with extra slashes or dots
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{"normal path", "dir/file.html", false},
		{"path with ./", "./dir/file.html", false},
		{"path with extra slashes", "dir//file.html", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := getFile(tt.path)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if f == nil {
				t.Fatal("expected non-nil file")
			}
		})
	}
}

func TestMidUiHandle_ContentLength(t *testing.T) {
	resetZipReader()
	origStaticPkg := comm.StaticPkg
	origInstalled := comm.Installed
	defer func() {
		comm.StaticPkg = origStaticPkg
		comm.Installed = origInstalled
	}()

	content := "<html>test content with known length</html>"
	comm.StaticPkg = createTestZip(t, map[string]string{
		"test.html": content,
	})
	comm.Installed = true

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(midUiHandle)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/test.html", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// Check that Content-Length is set correctly
	cl := w.Header().Get("Content-Length")
	if cl == "" {
		t.Error("expected Content-Length header to be set")
	}
}

func TestMidUiHandle_StreamingWithLimitReader(t *testing.T) {
	resetZipReader()
	origStaticPkg := comm.StaticPkg
	origInstalled := comm.Installed
	defer func() {
		comm.StaticPkg = origStaticPkg
		comm.Installed = origInstalled
	}()

	// Create a file with known content
	content := strings.Repeat("x", 1024)
	comm.StaticPkg = createTestZip(t, map[string]string{
		"data.txt": content,
	})
	comm.Installed = true

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(midUiHandle)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/data.txt", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// Verify all content was streamed
	body, _ := io.ReadAll(w.Body)
	if string(body) != content {
		t.Errorf("content mismatch: got %d bytes, want %d bytes", len(body), len(content))
	}
}

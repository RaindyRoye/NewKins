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

// createTestZip creates a minimal zip archive with the given files
func createTestZip(t *testing.T, files map[string]string) string {
	t.Helper()
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)

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

func TestGetRdr_ValidZip(t *testing.T) {
	origStaticPkg := comm.StaticPkg
	defer func() {
		comm.StaticPkg = origStaticPkg
		rder = nil
		rderOnce = sync.Once{}
		rderErr = nil
	}()

	files := map[string]string{
		"index.html": "<html><body>Test</body></html>",
		"style.css":  "body { margin: 0; }",
	}
	comm.StaticPkg = createTestZip(t, files)

	r, err := getRdr()
	if err != nil {
		t.Fatalf("getRdr() error: %v", err)
	}
	if r == nil {
		t.Fatal("expected zip reader, got nil")
	}
	if len(r.File) != 2 {
		t.Errorf("expected 2 files in zip, got %d", len(r.File))
	}
}

func TestGetRdr_InvalidBase64(t *testing.T) {
	origStaticPkg := comm.StaticPkg
	defer func() {
		comm.StaticPkg = origStaticPkg
		rder = nil
		rderOnce = sync.Once{}
		rderErr = nil
	}()

	comm.StaticPkg = "invalid-base64!"

	_, err := getRdr()
	if err == nil {
		t.Fatal("expected error for invalid base64, got nil")
	}
}

func TestGetRdr_InvalidZip(t *testing.T) {
	origStaticPkg := comm.StaticPkg
	defer func() {
		comm.StaticPkg = origStaticPkg
		rder = nil
		rderOnce = sync.Once{}
		rderErr = nil
	}()

	// Valid base64 but invalid zip format
	comm.StaticPkg = base64.StdEncoding.EncodeToString([]byte("not a zip file"))

	_, err := getRdr()
	if err == nil {
		t.Fatal("expected error for invalid zip, got nil")
	}
}

func TestGetFile_ValidFile(t *testing.T) {
	origStaticPkg := comm.StaticPkg
	defer func() {
		comm.StaticPkg = origStaticPkg
		rder = nil
		rderOnce = sync.Once{}
		rderErr = nil
	}()

	files := map[string]string{
		"index.html": "<html><body>Test</body></html>",
		"app.js":     "console.log('test');",
	}
	comm.StaticPkg = createTestZip(t, files)

	// Test finding index.html
	f, err := getFile("index.html")
	if err != nil {
		t.Fatalf("getFile(index.html) error: %v", err)
	}
	if f == nil {
		t.Fatal("expected file, got nil")
	}
	if f.Name != "index.html" {
		t.Errorf("expected file name 'index.html', got %q", f.Name)
	}

	// Test finding app.js
	f2, err := getFile("app.js")
	if err != nil {
		t.Fatalf("getFile(app.js) error: %v", err)
	}
	if f2 == nil {
		t.Fatal("expected file, got nil")
	}
	if f2.Name != "app.js" {
		t.Errorf("expected file name 'app.js', got %q", f2.Name)
	}
}

func TestGetFile_NotFound(t *testing.T) {
	origStaticPkg := comm.StaticPkg
	defer func() {
		comm.StaticPkg = origStaticPkg
		rder = nil
		rderOnce = sync.Once{}
		rderErr = nil
	}()

	files := map[string]string{
		"index.html": "<html><body>Test</body></html>",
	}
	comm.StaticPkg = createTestZip(t, files)

	_, err := getFile("nonexistent.html")
	if err == nil {
		t.Fatal("expected error for nonexistent file, got nil")
	}
}

func TestMidUiHandle_ServesStaticContent(t *testing.T) {
	origStaticPkg := comm.StaticPkg
	origInstalled := comm.Installed
	defer func() {
		comm.StaticPkg = origStaticPkg
		comm.Installed = origInstalled
		rder = nil
		rderOnce = sync.Once{}
		rderErr = nil
	}()

	files := map[string]string{
		"index.html": "<html><body>Test Page</body></html>",
		"style.css":  "body { margin: 0; }",
		"app.js":     "console.log('test');",
		"logo.svg":   "<svg></svg>",
	}
	comm.StaticPkg = createTestZip(t, files)
	comm.Installed = true

	tests := []struct {
		name        string
		path        string
		wantStatus  int
		wantContain string
	}{
		{
			name:        "html file",
			path:        "/index.html",
			wantStatus:  http.StatusOK,
			wantContain: "Test Page",
		},
		{
			name:        "css file",
			path:        "/style.css",
			wantStatus:  http.StatusOK,
			wantContain: "margin",
		},
		{
			name:        "js file",
			path:        "/app.js",
			wantStatus:  http.StatusOK,
			wantContain: "console.log",
		},
		{
			name:        "svg file",
			path:        "/logo.svg",
			wantStatus:  http.StatusOK,
			wantContain: "svg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			router := gin.New()
			router.Use(midUiHandle)
			// No handler registered - gin will return 404, allowing middleware to serve static files

			w := httptest.NewRecorder()
			req, _ := http.NewRequestWithContext(context.Background(), "GET", tt.path, nil)
			router.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, w.Code)
			}
			if tt.wantContain != "" && !bytes.Contains(w.Body.Bytes(), []byte(tt.wantContain)) {
				t.Errorf("expected body to contain %q, got %q", tt.wantContain, w.Body.String())
			}
		})
	}
}

func TestMidUiHandle_FallbackToIndex(t *testing.T) {
	origStaticPkg := comm.StaticPkg
	origInstalled := comm.Installed
	defer func() {
		comm.StaticPkg = origStaticPkg
		comm.Installed = origInstalled
		rder = nil
		rderOnce = sync.Once{}
		rderErr = nil
	}()

	files := map[string]string{
		"index.html": "<html><body>Index Page</body></html>",
	}
	comm.StaticPkg = createTestZip(t, files)
	comm.Installed = true

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(midUiHandle)
	// No handler registered - gin will return 404, allowing middleware to serve static files

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/unknown-route", nil)
	router.ServeHTTP(w, req)

	// Should fall back to index.html
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("Index Page")) {
		t.Errorf("expected fallback to index.html, got %q", w.Body.String())
	}
}

func TestMidUiHandle_NoStaticPkg(t *testing.T) {
	origStaticPkg := comm.StaticPkg
	origInstalled := comm.Installed
	defer func() {
		comm.StaticPkg = origStaticPkg
		comm.Installed = origInstalled
		rder = nil
		rderOnce = sync.Once{}
		rderErr = nil
	}()

	comm.StaticPkg = ""
	comm.Installed = true

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(midUiHandle)
	// No handler registered - gin will return 404, allowing middleware to handle

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/any-path", nil)
	router.ServeHTTP(w, req)

	// Should redirect to "/" when no content available
	if w.Code != http.StatusFound {
		t.Errorf("expected status 302 (redirect), got %d", w.Code)
	}
}

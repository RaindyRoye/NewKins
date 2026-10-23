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

// TestMidUiHandle_WithStaticContent tests serving actual static files from zip
func TestMidUiHandle_WithStaticContent(t *testing.T) {
	// Create a test zip with various file types
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	files := map[string]string{
		"index.html":           "<html><body>Hello</body></html>",
		"style.css":            "body { color: red; }",
		"app.js":               "console.log('test');",
		"logo.svg":             "<svg></svg>",
		"icons/gokins.ttf":     "fake ttf data",
		"icons/gokins.woff2":   "fake woff2 data",
		"gokinsui/index.html":  "<html>UI</html>",
	}

	for name, content := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatalf("failed to create zip entry %s: %v", name, err)
		}
		if _, err := w.Write([]byte(content)); err != nil {
			t.Fatalf("failed to write to %s: %v", name, err)
		}
	}

	if err := zw.Close(); err != nil {
		t.Fatalf("failed to close zip writer: %v", err)
	}

	// Save and restore globals
	origStaticPkg := comm.StaticPkg
	origInstalled := comm.Installed
	defer func() {
		comm.StaticPkg = origStaticPkg
		comm.Installed = origInstalled
		rder = nil
		rderOnce = sync.Once{}
		rderErr = nil
	}()

	comm.StaticPkg = base64.StdEncoding.EncodeToString(buf.Bytes())
	comm.Installed = true

	// Reset zip reader to pick up new StaticPkg
	rder = nil
	rderOnce = sync.Once{}
	rderErr = nil

	tests := []struct {
		path            string
		wantStatus      int
		wantContentType string
		wantContains    string
	}{
		{"/index.html", http.StatusOK, "text/html", "Hello"},
		{"/style.css", http.StatusOK, "text/css", "color: red"},
		{"/app.js", http.StatusOK, "application/javascript", "console.log"},
		{"/logo.svg", http.StatusOK, "image/svg+xml", "<svg>"},
		{"/icons/gokins.ttf", http.StatusOK, "application/x-font-ttf", ""},
		{"/icons/gokins.woff2", http.StatusOK, "", ""},
		{"/gokinsui/index.html", http.StatusOK, "text/html", "UI"},
		{"/nonexistent.html", http.StatusOK, "text/html", "Hello"}, // falls back to index.html
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			// Reset zip reader for each test
			rder = nil
			rderOnce = sync.Once{}
			rderErr = nil

			gin.SetMode(gin.TestMode)
			router := gin.New()
			router.Use(midUiHandle)

			w := httptest.NewRecorder()
			req, _ := http.NewRequestWithContext(context.Background(), "GET", tt.path, nil)
			router.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, w.Code)
			}

			if tt.wantContentType != "" {
				ct := w.Header().Get("Content-Type")
				if ct != tt.wantContentType {
					t.Errorf("expected Content-Type %q, got %q", tt.wantContentType, ct)
				}
			}

			if tt.wantContains != "" && !bytes.Contains(w.Body.Bytes(), []byte(tt.wantContains)) {
				t.Errorf("expected body to contain %q, got %q", tt.wantContains, w.Body.String())
			}
		})
	}
}

// TestMidUiHandle_CacheHeaders tests cache control headers for different file types
func TestMidUiHandle_CacheHeaders(t *testing.T) {
	// Create a minimal zip with one HTML file
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	w, _ := zw.Create("index.html")
	_, _ = w.Write([]byte("<html>test</html>"))
	_ = zw.Close()

	origStaticPkg := comm.StaticPkg
	origInstalled := comm.Installed
	defer func() {
		comm.StaticPkg = origStaticPkg
		comm.Installed = origInstalled
		rder = nil
		rderOnce = sync.Once{}
		rderErr = nil
	}()

	comm.StaticPkg = base64.StdEncoding.EncodeToString(buf.Bytes())
	comm.Installed = true

	rder = nil
	rderOnce = sync.Once{}
	rderErr = nil

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(midUiHandle)

	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/index.html", nil)
	wr := httptest.NewRecorder()
	router.ServeHTTP(wr, req)

	// HTML files should have no-cache
	cc := wr.Header().Get("Cache-Control")
	if cc != "no-cache" {
		t.Errorf("expected Cache-Control 'no-cache' for HTML, got %q", cc)
	}

	pr := wr.Header().Get("Pragma")
	if pr != "no-cache" {
		t.Errorf("expected Pragma 'no-cache' for HTML, got %q", pr)
	}

	exp := wr.Header().Get("Expires")
	if exp != "0" {
		t.Errorf("expected Expires '0' for HTML, got %q", exp)
	}
}

// TestMidUiHandle_InstallPathWhenNotInstalled tests that /install works when not installed
func TestMidUiHandle_InstallPathWhenNotInstalled(t *testing.T) {
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

	// Should not redirect, should reach the handler
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200 for /install, got %d", w.Code)
	}
	if w.Body.String() != "install page" {
		t.Errorf("expected 'install page', got %q", w.Body.String())
	}
}

// TestMidUiHandle_GokinsuiPathWhenNotInstalled tests that /gokinsui/ paths work when not installed
func TestMidUiHandle_GokinsuiPathWhenNotInstalled(t *testing.T) {
	// Create a minimal zip with gokinsui content
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	w, _ := zw.Create("gokinsui/index.html")
	_, _ = w.Write([]byte("<html>UI</html>"))
	_ = zw.Close()

	origStaticPkg := comm.StaticPkg
	origInstalled := comm.Installed
	defer func() {
		comm.StaticPkg = origStaticPkg
		comm.Installed = origInstalled
		rder = nil
		rderOnce = sync.Once{}
		rderErr = nil
	}()

	comm.StaticPkg = base64.StdEncoding.EncodeToString(buf.Bytes())
	comm.Installed = false

	rder = nil
	rderOnce = sync.Once{}
	rderErr = nil

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(midUiHandle)

	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/gokinsui/index.html", nil)
	wr := httptest.NewRecorder()
	router.ServeHTTP(wr, req)

	// Should serve the file even when not installed
	if wr.Code != http.StatusOK {
		t.Errorf("expected status 200 for /gokinsui/, got %d", wr.Code)
	}
}

// TestRunHbtp_NoHbtpHost tests that runHbtp returns early when HbtpHost is empty
func TestRunHbtp_NoHbtpHost(t *testing.T) {
	origHbtpHost := comm.Cfg.Server.HbtpHost
	defer func() { comm.Cfg.Server.HbtpHost = origHbtpHost }()

	comm.Cfg.Server.HbtpHost = ""

	// Should return immediately without error
	runHbtp()
}

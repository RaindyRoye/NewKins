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

// TestMidUiHandle_Installed_NoFile tests that when installed and no file is found,
// the middleware returns 302 redirect to home.
func TestMidUiHandle_Installed_NoFile(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Reset zip reader state
	rder = nil
	rderOnce = sync.Once{}
	rderErr = nil

	origInstalled := comm.Installed
	origStaticPkg := comm.StaticPkg
	defer func() {
		comm.Installed = origInstalled
		comm.StaticPkg = origStaticPkg
	}()

	comm.Installed = true
	comm.StaticPkg = "" // No static package

	router := gin.New()
	router.Use(midUiHandle)
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "test")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/nonexistent", nil)
	router.ServeHTTP(w, req)

	// Should redirect to home when file not found
	if w.Code != http.StatusFound {
		t.Errorf("expected status 302 (redirect), got %d", w.Code)
	}
}

// TestMidUiHandle_WithValidStaticPkg tests serving a valid static file.
func TestMidUiHandle_WithValidStaticPkg(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Reset zip reader state
	rder = nil
	rderOnce = sync.Once{}
	rderErr = nil

	origInstalled := comm.Installed
	origStaticPkg := comm.StaticPkg
	defer func() {
		comm.Installed = origInstalled
		comm.StaticPkg = origStaticPkg
	}()

	// Create a minimal zip file with an HTML file
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	fw, err := zw.Create("index.html")
	if err != nil {
		t.Fatalf("create zip entry: %v", err)
	}
	_, _ = fw.Write([]byte("<html><body>test</body></html>"))
	_ = zw.Close()

	comm.Installed = true
	comm.StaticPkg = base64.StdEncoding.EncodeToString(buf.Bytes())

	router := gin.New()
	router.Use(midUiHandle)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/index.html", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	if w.Header().Get("Content-Type") != "text/html" {
		t.Errorf("expected Content-Type text/html, got %s", w.Header().Get("Content-Type"))
	}
}

// TestGetRdr_InvalidBase64 tests that getRdr returns an error for invalid base64.
func TestGetRdr_InvalidBase64(t *testing.T) {
	// Reset state
	rder = nil
	rderOnce = sync.Once{}
	rderErr = nil

	origStaticPkg := comm.StaticPkg
	defer func() { comm.StaticPkg = origStaticPkg }()

	comm.StaticPkg = "not-valid-base64!!!"

	_, err := getRdr()
	if err == nil {
		t.Fatal("expected error for invalid base64, got nil")
	}
}

// TestGetRdr_InvalidZip tests that getRdr returns an error for invalid zip data.
func TestGetRdr_InvalidZip(t *testing.T) {
	// Reset state
	rder = nil
	rderOnce = sync.Once{}
	rderErr = nil

	origStaticPkg := comm.StaticPkg
	defer func() { comm.StaticPkg = origStaticPkg }()

	// Valid base64 but invalid zip
	comm.StaticPkg = base64.StdEncoding.EncodeToString([]byte("not a zip file"))

	_, err := getRdr()
	if err == nil {
		t.Fatal("expected error for invalid zip data, got nil")
	}
}

// TestGetFile_WithValidZip tests getFile with a valid zip containing the requested file.
func TestGetFile_WithValidZip(t *testing.T) {
	// Reset state
	rder = nil
	rderOnce = sync.Once{}
	rderErr = nil

	origStaticPkg := comm.StaticPkg
	defer func() { comm.StaticPkg = origStaticPkg }()

	// Create a zip with a CSS file
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	fw, err := zw.Create("styles/main.css")
	if err != nil {
		t.Fatalf("create zip entry: %v", err)
	}
	_, _ = fw.Write([]byte("body { color: red; }"))
	_ = zw.Close()

	comm.StaticPkg = base64.StdEncoding.EncodeToString(buf.Bytes())

	f, err := getFile("styles/main.css")
	if err != nil {
		t.Fatalf("getFile failed: %v", err)
	}
	if f.Name != "styles/main.css" {
		t.Errorf("expected file name 'styles/main.css', got %s", f.Name)
	}
}

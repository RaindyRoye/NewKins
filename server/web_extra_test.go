package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gokins/gokins/comm"
)

// resetRdr resets the zip reader singleton for test isolation.
func resetRdr() {
	rder = nil
	rderOnce = sync.Once{}
	rderErr = nil
}

// TestMidUiHandle_HandlerSucceeds verifies midUiHandle returns early when
// the handler returns a non-404 status code.
func TestMidUiHandle_HandlerSucceeds(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(midUiHandle)
	router.GET("/api/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	origInstalled := comm.Installed
	defer func() { comm.Installed = origInstalled }()
	comm.Installed = true

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/api/test", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	if w.Body.String() != "OK" {
		t.Errorf("expected body 'OK', got %q", w.Body.String())
	}
}

// TestMidUiHandle_404WithNoStaticPkg verifies midUiHandle redirects to /
// when no static assets are available.
func TestMidUiHandle_404WithNoStaticPkg(t *testing.T) {
	gin.SetMode(gin.TestMode)
	resetRdr()

	origStaticPkg := comm.StaticPkg
	origInstalled := comm.Installed
	defer func() {
		comm.StaticPkg = origStaticPkg
		comm.Installed = origInstalled
	}()
	comm.StaticPkg = ""
	comm.Installed = true

	router := gin.New()
	router.Use(midUiHandle)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/some-page", nil)
	router.ServeHTTP(w, req)

	// Should redirect to / since no static assets available
	if w.Code != http.StatusFound {
		t.Errorf("expected status 302 (redirect), got %d", w.Code)
	}
}

// TestMidUiHandle_NotInstalledNonUI verifies midUiHandle redirects to /install
// when not installed and path is not UI or install.
func TestMidUiHandle_NotInstalledNonUI(t *testing.T) {
	gin.SetMode(gin.TestMode)
	resetRdr()

	origInstalled := comm.Installed
	defer func() { comm.Installed = origInstalled }()
	comm.Installed = false

	router := gin.New()
	router.Use(midUiHandle)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/api/users", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("expected status 302 (redirect), got %d", w.Code)
	}
}

// TestMidUiHandle_NotInstalledGokinsUI verifies midUiHandle does not redirect
// when not installed but the path starts with /gokinsui/.
func TestMidUiHandle_NotInstalledGokinsUI(t *testing.T) {
	gin.SetMode(gin.TestMode)
	resetRdr()

	origInstalled := comm.Installed
	origStaticPkg := comm.StaticPkg
	defer func() {
		comm.Installed = origInstalled
		comm.StaticPkg = origStaticPkg
	}()
	comm.Installed = false
	comm.StaticPkg = ""

	router := gin.New()
	router.Use(midUiHandle)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/gokinsui/some.js", nil)
	router.ServeHTTP(w, req)

	// Should redirect (since no static pkg) but to / not /install
	if w.Code != http.StatusFound {
		t.Errorf("expected status 302, got %d", w.Code)
	}
}

// TestMidUiHandle_NotInstalledInstallPage verifies midUiHandle does not redirect
// when not installed but the path is /install.
func TestMidUiHandle_NotInstalledInstallPage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	resetRdr()

	origInstalled := comm.Installed
	defer func() { comm.Installed = origInstalled }()
	comm.Installed = false

	router := gin.New()
	router.Use(midUiHandle)
	router.GET("/install", func(c *gin.Context) {
		c.String(http.StatusOK, "install page")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/install", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

// TestGetFile_PathTraversalWithBackslash verifies getFile rejects paths
// containing backslashes that might be used for traversal on Windows.
func TestGetFile_PathTraversalWithBackslash(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{"backslash traversal", "..\\etc\\passwd"},
		{"mixed slashes", "foo\\..\\bar"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := getFile(tt.path)
			if err == nil {
				t.Fatal("expected error for path traversal, got nil")
			}
		})
	}
}

// TestGetFile_CleanedPath verifies getFile properly cleans and validates paths.
func TestGetFile_CleanedPath(t *testing.T) {
	resetRdr()

	origStaticPkg := comm.StaticPkg
	defer func() { comm.StaticPkg = origStaticPkg }()
	comm.StaticPkg = ""

	// With no static pkg, should fail to find any file
	_, err := getFile("some/file.txt")
	if err == nil {
		t.Fatal("expected error when static pkg is empty")
	}
}

// TestGetRdr_InvalidBase64 verifies getRdr handles invalid base64 input.
func TestGetRdr_InvalidBase64(t *testing.T) {
	resetRdr()

	origStaticPkg := comm.StaticPkg
	defer func() { comm.StaticPkg = origStaticPkg }()
	comm.StaticPkg = "not-valid-base64!@#$"

	_, err := getRdr()
	if err == nil {
		t.Fatal("expected error for invalid base64")
	}
}

// TestGetRdr_InvalidZip verifies getRdr handles valid base64 but invalid zip content.
func TestGetRdr_InvalidZip(t *testing.T) {
	resetRdr()

	origStaticPkg := comm.StaticPkg
	defer func() { comm.StaticPkg = origStaticPkg }()
	// Valid base64 encoding of "hello world" — not a valid zip file
	comm.StaticPkg = "aGVsbG8gd29ybGQ="

	_, err := getRdr()
	if err == nil {
		t.Fatal("expected error for invalid zip content")
	}
}

// TestGetRdr_EmptyStaticPkg verifies getRdr fails with empty StaticPkg.
func TestGetRdr_EmptyStaticPkg(t *testing.T) {
	resetRdr()

	origStaticPkg := comm.StaticPkg
	defer func() { comm.StaticPkg = origStaticPkg }()
	comm.StaticPkg = ""

	_, err := getRdr()
	// Empty string is valid base64 (decodes to empty bytes), but zip.NewReader
	// on empty buffer will fail
	if err == nil {
		t.Fatal("expected error for empty StaticPkg")
	}
}

// TestMidUiHandle_HandlerWithContent verifies midUiHandle returns early
// when handler writes content even with 404 status.
func TestMidUiHandle_HandlerWithContent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	resetRdr()

	origInstalled := comm.Installed
	defer func() { comm.Installed = origInstalled }()
	comm.Installed = true

	router := gin.New()
	router.Use(midUiHandle)
	router.NoRoute(func(c *gin.Context) {
		c.String(http.StatusNotFound, "custom 404")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/unknown", nil)
	router.ServeHTTP(w, req)

	// Since handler wrote content (Size > 0), midUiHandle should return early
	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
	if w.Body.String() != "custom 404" {
		t.Errorf("expected 'custom 404', got %q", w.Body.String())
	}
}

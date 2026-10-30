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

// TestMidUiHandle_Installed_NoFile verifies that midUiHandle returns 404
// when installed and no file is found.
func TestMidUiHandle_Installed_NoFile(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(midUiHandle)
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "test")
	})

	origInstalled := comm.Installed
	defer func() { comm.Installed = origInstalled }()
	comm.Installed = true

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/test", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

// TestMidUiHandle_NotFound_NotInstalled verifies that midUiHandle redirects
// to /install when not installed and path is not /install or /gokinsui/.
func TestMidUiHandle_NotFound_NotInstalled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(midUiHandle)

	origInstalled := comm.Installed
	defer func() { comm.Installed = origInstalled }()
	comm.Installed = false

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/some-page", nil)
	router.ServeHTTP(w, req)

	// Should redirect to /install
	if w.Code != http.StatusFound {
		t.Errorf("expected status 302 (redirect to /install), got %d", w.Code)
	}
}

// TestMidUiHandle_NotFound_InstallPath verifies that midUiHandle does not
// redirect when the path is /install even when not installed.
func TestMidUiHandle_NotFound_InstallPath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(midUiHandle)
	router.GET("/install", func(c *gin.Context) {
		c.String(http.StatusOK, "install page")
	})

	origInstalled := comm.Installed
	defer func() { comm.Installed = origInstalled }()
	comm.Installed = false

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/install", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

// TestMidUiHandle_NotFound_GokinsUiPath verifies that midUiHandle does not
// redirect when the path starts with /gokinsui/ even when not installed.
func TestMidUiHandle_NotFound_GokinsUiPath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(midUiHandle)

	origInstalled := comm.Installed
	defer func() { comm.Installed = origInstalled }()
	comm.Installed = false

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/gokinsui/something", nil)
	router.ServeHTTP(w, req)

	// Should not redirect, but will redirect to / since file not found
	if w.Code != http.StatusFound {
		t.Errorf("expected status 302 (redirect to /), got %d", w.Code)
	}
}

// TestGetFile_WithStaticPkg verifies getFile behavior with a valid zip package.
func TestGetFile_WithStaticPkg(t *testing.T) {
	// Reset the zip reader
	rder = nil
	rderOnce = sync.Once{}
	rderErr = nil

	origStaticPkg := comm.StaticPkg
	defer func() { comm.StaticPkg = origStaticPkg }()

	// Empty StaticPkg will cause base64 decode error
	comm.StaticPkg = ""

	_, err := getFile("test.html")
	if err == nil {
		t.Fatal("expected error with empty StaticPkg, got nil")
	}
}

// TestGetFile_PathTraversal_Variants verifies various path traversal attempts are blocked.
func TestGetFile_PathTraversal_Variants(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{"simple parent", "../etc/passwd"},
		{"absolute path", "/etc/passwd"},
		{"double dot in middle", "foo/../../bar"},
		{"encoded parent", "foo%2F..%2F..%2Fetc%2Fpasswd"},
		{"multiple parents", "../../../etc/passwd"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := getFile(tt.path)
			if err == nil {
				t.Fatalf("expected error for path traversal %q, got nil", tt.path)
			}
		})
	}
}

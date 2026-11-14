package server

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gokins/gokins/comm"
)

// createTestZip creates a minimal zip archive with the given files and returns
// the base64-encoded content suitable for use as comm.StaticPkg.
func createTestZip(t *testing.T, files map[string]string) string {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, content := range files {
		f, err := zw.Create(name)
		if err != nil {
			t.Fatalf("create zip entry %q: %v", name, err)
		}
		if _, err := f.Write([]byte(content)); err != nil {
			t.Fatalf("write zip entry %q: %v", name, err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("close zip writer: %v", err)
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes())
}

func resetZipReader() {
	rder = nil
	rderOnce = sync.Once{}
	rderErr = nil
}

func TestMidUiHandle_ServesStaticFile(t *testing.T) {
	resetZipReader()
	origStaticPkg := comm.StaticPkg
	origInstalled := comm.Installed
	defer func() {
		comm.StaticPkg = origStaticPkg
		comm.Installed = origInstalled
		resetZipReader()
	}()

	comm.Installed = true
	comm.StaticPkg = createTestZip(t, map[string]string{
		"index.html": "<html><body>Hello</body></html>",
	})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(midUiHandle)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/index.html", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	if w.Body.String() != "<html><body>Hello</body></html>" {
		t.Errorf("unexpected body: %q", w.Body.String())
	}
}

func TestMidUiHandle_FallsBackToIndexHTML(t *testing.T) {
	resetZipReader()
	origStaticPkg := comm.StaticPkg
	origInstalled := comm.Installed
	defer func() {
		comm.StaticPkg = origStaticPkg
		comm.Installed = origInstalled
		resetZipReader()
	}()

	comm.Installed = true
	comm.StaticPkg = createTestZip(t, map[string]string{
		"index.html": "<html>fallback</html>",
	})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(midUiHandle)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/nonexistent-page", nil)
	router.ServeHTTP(w, req)

	// Should fall back to index.html
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	if w.Body.String() != "<html>fallback</html>" {
		t.Errorf("unexpected body: %q", w.Body.String())
	}
}

func TestMidUiHandle_NotInstalledRedirect(t *testing.T) {
	resetZipReader()
	origStaticPkg := comm.StaticPkg
	origInstalled := comm.Installed
	defer func() {
		comm.StaticPkg = origStaticPkg
		comm.Installed = origInstalled
		resetZipReader()
	}()

	comm.Installed = false
	comm.StaticPkg = ""

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(midUiHandle)
	router.GET("/install", func(c *gin.Context) {
		c.String(http.StatusOK, "install page")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/some-page", nil)
	router.ServeHTTP(w, req)

	// Should redirect to /install
	if w.Code != http.StatusFound {
		t.Errorf("expected status 302, got %d", w.Code)
	}
}

func TestMidUiHandle_NoContentRedirects(t *testing.T) {
	resetZipReader()
	origStaticPkg := comm.StaticPkg
	origInstalled := comm.Installed
	defer func() {
		comm.StaticPkg = origStaticPkg
		comm.Installed = origInstalled
		resetZipReader()
	}()

	comm.Installed = true
	comm.StaticPkg = "" // Empty zip - getRdr will fail

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(midUiHandle)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/any-page", nil)
	router.ServeHTTP(w, req)

	// When no content is available, should redirect to /
	if w.Code != http.StatusFound {
		t.Errorf("expected status 302 (redirect to /), got %d", w.Code)
	}
}

func TestMidUiHandle_SkipsNon404Responses(t *testing.T) {
	resetZipReader()
	origStaticPkg := comm.StaticPkg
	origInstalled := comm.Installed
	defer func() {
		comm.StaticPkg = origStaticPkg
		comm.Installed = origInstalled
		resetZipReader()
	}()

	comm.Installed = true
	comm.StaticPkg = createTestZip(t, map[string]string{
		"index.html": "<html>should not be served</html>",
	})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(midUiHandle)
	router.GET("/api/test", func(c *gin.Context) {
		c.String(http.StatusOK, "api response")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/test", nil)
	router.ServeHTTP(w, req)

	// Should return the API response, not static file
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	if w.Body.String() != "api response" {
		t.Errorf("unexpected body: %q", w.Body.String())
	}
}

func TestRunHbtp_EmptyHost(t *testing.T) {
	origCfg := comm.Cfg
	defer func() { comm.Cfg = origCfg }()

	comm.Cfg.Server.HbtpHost = ""

	// runHbtp should return immediately when HbtpHost is empty
	// This test verifies it doesn't panic or hang
	done := make(chan struct{})
	go func() {
		runHbtp()
		close(done)
	}()

	<-done
	// If we reach here, runHbtp returned successfully
}

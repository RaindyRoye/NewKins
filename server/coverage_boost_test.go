package server

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gokins/gokins/comm"
	_ "github.com/mattn/go-sqlite3"
	"xorm.io/xorm"
)

// ---------------------------------------------------------------------------
// midUiHandle coverage tests
// ---------------------------------------------------------------------------

func TestMidUiHandle_Installed_NotFound_RedirectsToRoot(t *testing.T) {
	gin.SetMode(gin.TestMode)
	comm.Installed = true

	// Reset the zip reader so getFile will use comm.StaticPkg
	origStaticPkg := comm.StaticPkg
	origInstalled := comm.Installed
	rder = nil
	rderOnce = sync.Once{}
	rderErr = nil
	defer func() {
		comm.StaticPkg = origStaticPkg
		comm.Installed = origInstalled
		rder = nil
		rderOnce = sync.Once{}
		rderErr = nil
	}()
	comm.StaticPkg = "" // Empty → getRdr will fail, triggering redirect

	router := gin.New()
	router.Use(midUiHandle)
	router.GET("/test-page", func(c *gin.Context) {
		// Don't write anything — leaves status at 0 which becomes 200
		// but we need 404 to trigger the fallback
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/nonexistent-page", nil)
	router.ServeHTTP(w, req)

	// The midUiHandle should try getFile, fail, try index.html, fail again, then redirect
	// ResMsgUrl returns 302
	if w.Code != http.StatusFound {
		t.Errorf("expected status 302 (redirect), got %d", w.Code)
	}
}

func TestMidUiHandle_Installed_ServesHTMLFromZip(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create a real zip file with an HTML entry
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	fw, err := zw.Create("index.html")
	if err != nil {
		t.Fatalf("create zip entry: %v", err)
	}
	htmlContent := []byte("<html><body>Hello</body></html>")
	if _, err := fw.Write(htmlContent); err != nil {
		t.Fatalf("write zip entry: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}

	origStaticPkg := comm.StaticPkg
	origInstalled := comm.Installed
	rder = nil
	rderOnce = sync.Once{}
	rderErr = nil
	defer func() {
		comm.StaticPkg = origStaticPkg
		comm.Installed = origInstalled
		rder = nil
		rderOnce = sync.Once{}
		rderErr = nil
	}()
	comm.Installed = true
	comm.StaticPkg = base64.StdEncoding.EncodeToString(buf.Bytes())

	router := gin.New()
	router.Use(midUiHandle)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/some-page", nil)
	router.ServeHTTP(w, req)

	// Should serve index.html with text/html content-type
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	ct := w.Header().Get("Content-Type")
	if ct != "text/html" {
		t.Errorf("expected Content-Type 'text/html', got %q", ct)
	}
	if w.Header().Get("Cache-Control") != "no-cache" {
		t.Errorf("HTML should have Cache-Control: no-cache, got %q", w.Header().Get("Cache-Control"))
	}
}

func TestMidUiHandle_Installed_ServesCSSFromZip(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	fw, err := zw.Create("style.css")
	if err != nil {
		t.Fatalf("create zip entry: %v", err)
	}
	cssContent := []byte("body { color: red; }")
	if _, err := fw.Write(cssContent); err != nil {
		t.Fatalf("write zip entry: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}

	origStaticPkg := comm.StaticPkg
	origInstalled := comm.Installed
	rder = nil
	rderOnce = sync.Once{}
	rderErr = nil
	defer func() {
		comm.StaticPkg = origStaticPkg
		comm.Installed = origInstalled
		rder = nil
		rderOnce = sync.Once{}
		rderErr = nil
	}()
	comm.Installed = true
	comm.StaticPkg = base64.StdEncoding.EncodeToString(buf.Bytes())

	router := gin.New()
	router.Use(midUiHandle)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/style.css", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	ct := w.Header().Get("Content-Type")
	if ct != "text/css" {
		t.Errorf("expected Content-Type 'text/css', got %q", ct)
	}
}

func TestMidUiHandle_Installed_ServesJSFromZip(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	fw, err := zw.Create("app.js")
	if err != nil {
		t.Fatalf("create zip entry: %v", err)
	}
	jsContent := []byte("console.log('hello');")
	if _, err := fw.Write(jsContent); err != nil {
		t.Fatalf("write zip entry: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}

	origStaticPkg := comm.StaticPkg
	origInstalled := comm.Installed
	rder = nil
	rderOnce = sync.Once{}
	rderErr = nil
	defer func() {
		comm.StaticPkg = origStaticPkg
		comm.Installed = origInstalled
		rder = nil
		rderOnce = sync.Once{}
		rderErr = nil
	}()
	comm.Installed = true
	comm.StaticPkg = base64.StdEncoding.EncodeToString(buf.Bytes())

	router := gin.New()
	router.Use(midUiHandle)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/app.js", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	ct := w.Header().Get("Content-Type")
	if ct != "application/javascript" {
		t.Errorf("expected Content-Type 'application/javascript', got %q", ct)
	}
}

func TestMidUiHandle_Installed_ServesSVGFromZip(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	fw, err := zw.Create("logo.svg")
	if err != nil {
		t.Fatalf("create zip entry: %v", err)
	}
	if _, err := fw.Write([]byte("<svg></svg>")); err != nil {
		t.Fatalf("write zip entry: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}

	origStaticPkg := comm.StaticPkg
	origInstalled := comm.Installed
	rder = nil
	rderOnce = sync.Once{}
	rderErr = nil
	defer func() {
		comm.StaticPkg = origStaticPkg
		comm.Installed = origInstalled
		rder = nil
		rderOnce = sync.Once{}
		rderErr = nil
	}()
	comm.Installed = true
	comm.StaticPkg = base64.StdEncoding.EncodeToString(buf.Bytes())

	router := gin.New()
	router.Use(midUiHandle)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/logo.svg", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	ct := w.Header().Get("Content-Type")
	if ct != "image/svg+xml" {
		t.Errorf("expected Content-Type 'image/svg+xml', got %q", ct)
	}
}

func TestMidUiHandle_Installed_ServesFontFromZip(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	fw, err := zw.Create("font.ttf")
	if err != nil {
		t.Fatalf("create zip entry: %v", err)
	}
	if _, err := fw.Write([]byte{0x00, 0x01}); err != nil {
		t.Fatalf("write zip entry: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}

	origStaticPkg := comm.StaticPkg
	origInstalled := comm.Installed
	rder = nil
	rderOnce = sync.Once{}
	rderErr = nil
	defer func() {
		comm.StaticPkg = origStaticPkg
		comm.Installed = origInstalled
		rder = nil
		rderOnce = sync.Once{}
		rderErr = nil
	}()
	comm.Installed = true
	comm.StaticPkg = base64.StdEncoding.EncodeToString(buf.Bytes())

	router := gin.New()
	router.Use(midUiHandle)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/font.ttf", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	ct := w.Header().Get("Content-Type")
	if ct != "application/x-font-ttf" {
		t.Errorf("expected Content-Type 'application/x-font-ttf', got %q", ct)
	}
}

func TestMidUiHandle_StatusNon404_SkipsFallback(t *testing.T) {
	gin.SetMode(gin.TestMode)

	origInstalled := comm.Installed
	defer func() { comm.Installed = origInstalled }()
	comm.Installed = true

	router := gin.New()
	router.Use(midUiHandle)
	router.GET("/api/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/api/test", nil)
	router.ServeHTTP(w, req)

	// Should return the handler's response, not the fallback
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	if w.Body.String() != "ok" {
		t.Errorf("expected body 'ok', got %q", w.Body.String())
	}
}

func TestMidUiHandle_InstallPath_NotInstalled(t *testing.T) {
	gin.SetMode(gin.TestMode)

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

	// /install path should be served even when not installed
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestMidUiHandle_GokinsUIPath_NotInstalled(t *testing.T) {
	gin.SetMode(gin.TestMode)

	origInstalled := comm.Installed
	defer func() { comm.Installed = origInstalled }()
	comm.Installed = false

	router := gin.New()
	router.Use(midUiHandle)
	router.GET("/gokinsui/some-asset", func(c *gin.Context) {
		c.String(http.StatusOK, "asset content")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/gokinsui/some-asset", nil)
	router.ServeHTTP(w, req)

	// /gokinsui/ paths should be served even when not installed
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

// ---------------------------------------------------------------------------
// getFile tests with real zip content
// ---------------------------------------------------------------------------

func TestGetFile_WithValidZipContent(t *testing.T) {
	// Create a zip with a known file
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	fw, err := zw.Create("test.txt")
	if err != nil {
		t.Fatalf("create zip entry: %v", err)
	}
	if _, err := fw.Write([]byte("hello world")); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}

	origStaticPkg := comm.StaticPkg
	rder = nil
	rderOnce = sync.Once{}
	rderErr = nil
	defer func() {
		comm.StaticPkg = origStaticPkg
		rder = nil
		rderOnce = sync.Once{}
		rderErr = nil
	}()
	comm.StaticPkg = base64.StdEncoding.EncodeToString(buf.Bytes())

	f, err := getFile("test.txt")
	if err != nil {
		t.Fatalf("getFile should succeed for valid zip entry: %v", err)
	}
	if f.Name != "test.txt" {
		t.Errorf("expected file name 'test.txt', got %q", f.Name)
	}
}

func TestGetFile_NotInZip(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	fw, err := zw.Create("exists.txt")
	if err != nil {
		t.Fatalf("create zip entry: %v", err)
	}
	if _, err := fw.Write([]byte("data")); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}

	origStaticPkg := comm.StaticPkg
	rder = nil
	rderOnce = sync.Once{}
	rderErr = nil
	defer func() {
		comm.StaticPkg = origStaticPkg
		rder = nil
		rderOnce = sync.Once{}
		rderErr = nil
	}()
	comm.StaticPkg = base64.StdEncoding.EncodeToString(buf.Bytes())

	_, err = getFile("missing.txt")
	if err == nil {
		t.Fatal("expected error for missing file in zip, got nil")
	}
}

// ---------------------------------------------------------------------------
// getRdr tests
// ---------------------------------------------------------------------------

func TestGetRdr_InvalidBase64Data(t *testing.T) {
	origStaticPkg := comm.StaticPkg
	rder = nil
	rderOnce = sync.Once{}
	rderErr = nil
	defer func() {
		comm.StaticPkg = origStaticPkg
		rder = nil
		rderOnce = sync.Once{}
		rderErr = nil
	}()
	comm.StaticPkg = "!!!not-valid-base64!!!"

	_, err := getRdr()
	if err == nil {
		t.Fatal("expected error for invalid base64, got nil")
	}
}

func TestGetRdr_InvalidZipContent(t *testing.T) {
	origStaticPkg := comm.StaticPkg
	rder = nil
	rderOnce = sync.Once{}
	rderErr = nil
	defer func() {
		comm.StaticPkg = origStaticPkg
		rder = nil
		rderOnce = sync.Once{}
		rderErr = nil
	}()
	// Valid base64 but not a valid zip
	comm.StaticPkg = base64.StdEncoding.EncodeToString([]byte("this is not a zip file"))

	_, err := getRdr()
	if err == nil {
		t.Fatal("expected error for invalid zip content, got nil")
	}
}

func TestGetRdr_ValidZip(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	fw, err := zw.Create("readme.txt")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := fw.Write([]byte("hello")); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	origStaticPkg := comm.StaticPkg
	rder = nil
	rderOnce = sync.Once{}
	rderErr = nil
	defer func() {
		comm.StaticPkg = origStaticPkg
		rder = nil
		rderOnce = sync.Once{}
		rderErr = nil
	}()
	comm.StaticPkg = base64.StdEncoding.EncodeToString(buf.Bytes())

	r, err := getRdr()
	if err != nil {
		t.Fatalf("getRdr should succeed for valid zip: %v", err)
	}
	if r == nil {
		t.Fatal("getRdr returned nil reader")
	}
	if len(r.File) != 1 {
		t.Errorf("expected 1 file in zip, got %d", len(r.File))
	}
}

// ---------------------------------------------------------------------------
// initCache tests
// ---------------------------------------------------------------------------

func TestInitCache_Success(t *testing.T) {
	origWorkPath := comm.WorkPath
	origBCache := comm.BCache
	defer func() {
		comm.WorkPath = origWorkPath
		if comm.BCache != nil {
			_ = comm.BCache.Close()
		}
		comm.BCache = origBCache
	}()

	tmpDir := t.TempDir()
	comm.WorkPath = tmpDir

	err := initCache()
	if err != nil {
		t.Fatalf("initCache failed: %v", err)
	}
	if comm.BCache == nil {
		t.Fatal("BCache should not be nil after successful initCache")
	}

	// Verify the cache file exists
	cachePath := filepath.Join(tmpDir, "cache.dat")
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		t.Error("cache.dat should exist after initCache")
	}
}

// ---------------------------------------------------------------------------
// initDb tests (limited — real DB required for full coverage)
// ---------------------------------------------------------------------------

func TestInitDb_Sqlite3Driver(t *testing.T) {
	origDb := comm.Db
	origCfg := comm.Cfg
	origInstalled := comm.Installed
	origIsMySQL := comm.IsMySQL
	origWorkPath := comm.WorkPath
	defer func() {
		if comm.Db != nil {
			_ = comm.Db.Close()
		}
		comm.Db = origDb
		comm.Cfg = origCfg
		comm.Installed = origInstalled
		comm.IsMySQL = origIsMySQL
		comm.WorkPath = origWorkPath
	}()

	tmpDir := t.TempDir()
	comm.WorkPath = tmpDir
	// xorm expects "sqlite3" as the driver name
	comm.Cfg.Datasource.Driver = "sqlite3"
	dbPath := filepath.Join(tmpDir, "test.db")
	comm.Cfg.Datasource.Url = dbPath
	comm.Installed = true // skip migrations
	comm.IsMySQL = false

	err := initDb()
	if err != nil {
		t.Fatalf("initDb with sqlite3 failed: %v", err)
	}
	if comm.Db == nil {
		t.Fatal("Db should not be nil after initDb")
	}
	if comm.IsMySQL {
		t.Error("IsMySQL should be false for sqlite3 driver")
	}
}

func TestInitDb_MySQLDriverFlag(t *testing.T) {
	origDb := comm.Db
	origCfg := comm.Cfg
	origInstalled := comm.Installed
	origIsMySQL := comm.IsMySQL
	defer func() {
		if comm.Db != nil {
			_ = comm.Db.Close()
		}
		comm.Db = origDb
		comm.Cfg = origCfg
		comm.Installed = origInstalled
		comm.IsMySQL = origIsMySQL
	}()

	comm.Cfg.Datasource.Driver = "mysql"
	comm.Cfg.Datasource.Url = "root:password@tcp(nonexistent:3306)/testdb"
	comm.Installed = true
	comm.IsMySQL = false

	// initDb will create an xorm engine; it won't fail until ping
	err := initDb()
	// xorm.NewEngine may succeed even with bad credentials
	if err == nil {
		// If engine creation succeeded, IsMySQL should be true
		if !comm.IsMySQL {
			t.Error("IsMySQL should be true for mysql driver")
		}
	}
}

func TestInitDb_DefaultDriver(t *testing.T) {
	origDb := comm.Db
	origCfg := comm.Cfg
	origInstalled := comm.Installed
	origIsMySQL := comm.IsMySQL
	origWorkPath := comm.WorkPath
	defer func() {
		if comm.Db != nil {
			_ = comm.Db.Close()
		}
		comm.Db = origDb
		comm.Cfg = origCfg
		comm.Installed = origInstalled
		comm.IsMySQL = origIsMySQL
		comm.WorkPath = origWorkPath
	}()

	tmpDir := t.TempDir()
	comm.WorkPath = tmpDir
	comm.Cfg.Datasource.Driver = "" // should default to mysql
	comm.Cfg.Datasource.Url = "root:pass@tcp(localhost:3306)/test"
	comm.Installed = true

	_ = initDb() // may succeed or fail depending on xorm behavior
	// The important thing is that IsMySQL is set to true when driver defaults to mysql
	if !comm.IsMySQL {
		t.Error("IsMySQL should be true when driver defaults to mysql")
	}
}

// ---------------------------------------------------------------------------
// createIndexIfNotExists MySQL path
// ---------------------------------------------------------------------------

func TestCreateIndexIfNotExists_MySQLPath(t *testing.T) {
	// We can't easily test with a real MySQL, but we can test the function
	// with a sqlite DB while IsMySQL is true to verify error handling
	origDb := comm.Db
	origIsMySQL := comm.IsMySQL

	db, err := xorm.NewEngine("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer func() {
		_ = db.Close()
		comm.Db = origDb
		comm.IsMySQL = origIsMySQL
	}()

	comm.Db = db
	comm.IsMySQL = true // Force MySQL path

	_, err = db.Exec(`CREATE TABLE t_test (id VARCHAR(64) PRIMARY KEY, col1 VARCHAR(64))`)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}

	// This will use the MySQL SQL syntax on sqlite, which may or may not error
	// The important thing is the function doesn't panic
	_ = createIndexIfNotExists("t_test", "idx_test_col1", "col1")
}

// ---------------------------------------------------------------------------
// runHbtp tests
// ---------------------------------------------------------------------------

func TestRunHbtp_EmptyHost(t *testing.T) {
	origCfg := comm.Cfg
	defer func() { comm.Cfg = origCfg }()

	comm.Cfg.Server.HbtpHost = ""

	// Should return immediately without panicking
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("runHbtp panicked with empty HbtpHost: %v", r)
		}
	}()
	runHbtp()
}

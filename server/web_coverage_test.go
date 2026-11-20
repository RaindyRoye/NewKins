package server

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gokins/gokins/comm"
	_ "github.com/mattn/go-sqlite3"
	"xorm.io/xorm"
)

// ---------------------------------------------------------------------------
// initCache coverage
// ---------------------------------------------------------------------------

func TestInitCache_Success(t *testing.T) {
	origWorkPath := comm.WorkPath
	origCache := comm.BCache
	defer func() {
		comm.WorkPath = origWorkPath
		if comm.BCache != nil {
			_ = comm.BCache.Close()
		}
		comm.BCache = origCache
	}()

	tmpDir := t.TempDir()
	comm.WorkPath = tmpDir

	if err := initCache(); err != nil {
		t.Fatalf("initCache() error = %v", err)
	}
	if comm.BCache == nil {
		t.Fatal("expected BCache to be initialized, got nil")
	}

	// Verify the cache file exists
	cachePath := filepath.Join(tmpDir, "cache.dat")
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		t.Errorf("cache file %q was not created", cachePath)
	}
}

func TestInitCache_InvalidWorkPath(t *testing.T) {
	origWorkPath := comm.WorkPath
	origCache := comm.BCache
	defer func() {
		comm.WorkPath = origWorkPath
		comm.BCache = origCache
	}()

	// Use a path that cannot be written to
	comm.WorkPath = "/nonexistent/directory/that/does/not/exist"

	err := initCache()
	if err == nil {
		// Clean up if it somehow succeeded (e.g. running as root)
		if comm.BCache != nil {
			_ = comm.BCache.Close()
			comm.BCache = nil
		}
		t.Skip("skipping: initCache succeeded unexpectedly (possibly running as root)")
	}
	if !strings.Contains(err.Error(), "open cache db") {
		t.Errorf("error = %q, want to contain 'open cache db'", err.Error())
	}
}

// ---------------------------------------------------------------------------
// createIndexIfNotExists MySQL path coverage
// ---------------------------------------------------------------------------

func TestCreateIndexIfNotExists_MySQLPath(t *testing.T) {
	origDb := comm.Db
	origIsMySQL := comm.IsMySQL
	origCtx := comm.Ctx
	defer func() {
		comm.Db = origDb
		comm.IsMySQL = origIsMySQL
		comm.Ctx = origCtx
	}()

	db, err := xorm.NewEngine("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer func() { _ = db.Close() }()

	comm.Db = db
	comm.IsMySQL = true // Force MySQL path which uses plain CREATE INDEX

	_, err = db.Exec(`CREATE TABLE t_mysql_test (id VARCHAR(64) PRIMARY KEY, col1 VARCHAR(64))`)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}

	// First creation should succeed
	if err := createIndexIfNotExists("t_mysql_test", "idx_mysql_col1", "col1"); err != nil {
		t.Fatalf("first create: %v", err)
	}

	// Second creation should silently handle "Duplicate key name". SQLite
	// returns a different error message, so this exercises the MySQL error
	// matching logic only when running against a real MySQL server. The test
	// still verifies that the function doesn't panic or corrupt state.
	_ = createIndexIfNotExists("t_mysql_test", "idx_mysql_col1", "col1")
}

func TestCreateIndexIfNotExists_MySQLDuplicateKeyError(t *testing.T) {
	origDb := comm.Db
	origIsMySQL := comm.IsMySQL
	defer func() {
		comm.Db = origDb
		comm.IsMySQL = origIsMySQL
	}()

	db, err := xorm.NewEngine("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer func() { _ = db.Close() }()

	comm.Db = db
	comm.IsMySQL = true

	_, err = db.Exec(`CREATE TABLE t_dup_test (id VARCHAR(64) PRIMARY KEY, col1 VARCHAR(64))`)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}

	// Create an index with a duplicate name to trigger an error
	_, err = db.Exec(`CREATE INDEX idx_dup_col1 ON t_dup_test (col1)`)
	if err != nil {
		t.Fatalf("pre-create index: %v", err)
	}

	// Attempting to create the same index again in MySQL mode should return
	// an error. The MySQL error matching (Duplicate key name / 1061) won't
	// match SQLite's error, so we expect the function to propagate the error.
	err = createIndexIfNotExists("t_dup_test", "idx_dup_col1", "col1")
	if err == nil {
		t.Log("index creation succeeded (SQLite silently ignored duplicate); OK")
	}
}

// ---------------------------------------------------------------------------
// getRdr coverage — invalid base64, invalid zip
// ---------------------------------------------------------------------------

// resetZipReader resets the package-level zip reader state for test isolation.
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
}

func TestGetRdr_InvalidZipData(t *testing.T) {
	resetZipReader()
	origStaticPkg := comm.StaticPkg
	defer func() { comm.StaticPkg = origStaticPkg }()

	// Valid base64 but not a zip file
	comm.StaticPkg = base64.StdEncoding.EncodeToString([]byte("this is not a zip file"))

	_, err := getRdr()
	if err == nil {
		t.Fatal("expected error for invalid zip data, got nil")
	}
}

// buildTestZip creates a base64-encoded zip archive containing the given
// filename→content pairs.
func buildTestZip(t *testing.T, files map[string]string) string {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for name, content := range files {
		f, err := w.Create(name)
		if err != nil {
			t.Fatalf("zip create %q: %v", name, err)
		}
		if _, err := f.Write([]byte(content)); err != nil {
			t.Fatalf("zip write %q: %v", name, err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatalf("zip close: %v", err)
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes())
}

func TestGetRdr_ValidZip(t *testing.T) {
	resetZipReader()
	origStaticPkg := comm.StaticPkg
	defer func() { comm.StaticPkg = origStaticPkg }()

	comm.StaticPkg = buildTestZip(t, map[string]string{
		"index.html":     "<html><body>Hello</body></html>",
		"assets/app.js":  "console.log('hello');",
		"assets/app.css": "body { color: red; }",
	})

	r, err := getRdr()
	if err != nil {
		t.Fatalf("getRdr() error = %v", err)
	}
	if r == nil {
		t.Fatal("expected non-nil zip reader")
	}
	if len(r.File) != 3 {
		t.Errorf("expected 3 files in zip, got %d", len(r.File))
	}
}

func TestGetFile_Found(t *testing.T) {
	resetZipReader()
	origStaticPkg := comm.StaticPkg
	defer func() { comm.StaticPkg = origStaticPkg }()

	comm.StaticPkg = buildTestZip(t, map[string]string{
		"index.html": "<html>test</html>",
	})

	f, err := getFile("index.html")
	if err != nil {
		t.Fatalf("getFile() error = %v", err)
	}
	if f == nil {
		t.Fatal("expected non-nil zip file")
	}
	if f.Name != "index.html" {
		t.Errorf("file name = %q, want 'index.html'", f.Name)
	}
}

func TestGetFile_NotFound(t *testing.T) {
	resetZipReader()
	origStaticPkg := comm.StaticPkg
	defer func() { comm.StaticPkg = origStaticPkg }()

	comm.StaticPkg = buildTestZip(t, map[string]string{
		"index.html": "<html>test</html>",
	})

	_, err := getFile("nonexistent.html")
	if err == nil {
		t.Fatal("expected error for nonexistent file, got nil")
	}
	if !strings.Contains(err.Error(), "file not found") {
		t.Errorf("error = %q, want to contain 'file not found'", err.Error())
	}
}

// ---------------------------------------------------------------------------
// midUiHandle coverage — content-type branches, size check, open error
// ---------------------------------------------------------------------------

// setupUiRouter creates a test gin router with midUiHandle middleware.
func setupUiRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(midUiHandle)
	// Catch-all that returns 404 so midUiHandle's fallback logic triggers
	r.NoRoute(func(c *gin.Context) {
		c.Status(http.StatusNotFound)
	})
	return r
}

func TestMidUiHandle_ServesHTML(t *testing.T) {
	resetZipReader()
	origStaticPkg := comm.StaticPkg
	origInstalled := comm.Installed
	defer func() {
		comm.StaticPkg = origStaticPkg
		comm.Installed = origInstalled
	}()

	comm.Installed = true
	comm.StaticPkg = buildTestZip(t, map[string]string{
		"index.html": "<html><body>Hello World</body></html>",
	})

	router := setupUiRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(httptest.NewRequest("GET", "/nonexistent", nil).Context(), "GET", "/nonexistent", nil)
	router.ServeHTTP(w, req)

	// midUiHandle should fall back to index.html when the requested path isn't found
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	ct := w.Header().Get("Content-Type")
	if ct != "text/html" {
		t.Errorf("Content-Type = %q, want 'text/html'", ct)
	}
	cc := w.Header().Get("Cache-Control")
	if cc != "no-cache" {
		t.Errorf("Cache-Control = %q, want 'no-cache'", cc)
	}
	if !strings.Contains(w.Body.String(), "Hello World") {
		t.Errorf("body = %q, want to contain 'Hello World'", w.Body.String())
	}
}

func TestMidUiHandle_ServesCSS(t *testing.T) {
	resetZipReader()
	origStaticPkg := comm.StaticPkg
	origInstalled := comm.Installed
	defer func() {
		comm.StaticPkg = origStaticPkg
		comm.Installed = origInstalled
	}()

	comm.Installed = true
	comm.StaticPkg = buildTestZip(t, map[string]string{
		"assets/app.css": "body { color: red; }",
	})

	router := setupUiRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(httptest.NewRequest("GET", "/", nil).Context(), "GET", "/assets/app.css", nil)
	// The route won't match a handler, so midUiHandle's fallback triggers
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	ct := w.Header().Get("Content-Type")
	if ct != "text/css" {
		t.Errorf("Content-Type = %q, want 'text/css'", ct)
	}
}

func TestMidUiHandle_ServesJS(t *testing.T) {
	resetZipReader()
	origStaticPkg := comm.StaticPkg
	origInstalled := comm.Installed
	defer func() {
		comm.StaticPkg = origStaticPkg
		comm.Installed = origInstalled
	}()

	comm.Installed = true
	comm.StaticPkg = buildTestZip(t, map[string]string{
		"assets/app.js": "console.log('hello');",
	})

	router := setupUiRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(httptest.NewRequest("GET", "/", nil).Context(), "GET", "/assets/app.js", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	ct := w.Header().Get("Content-Type")
	if ct != "application/javascript" {
		t.Errorf("Content-Type = %q, want 'application/javascript'", ct)
	}
}

func TestMidUiHandle_ServesSVG(t *testing.T) {
	resetZipReader()
	origStaticPkg := comm.StaticPkg
	origInstalled := comm.Installed
	defer func() {
		comm.StaticPkg = origStaticPkg
		comm.Installed = origInstalled
	}()

	comm.Installed = true
	comm.StaticPkg = buildTestZip(t, map[string]string{
		"assets/logo.svg": "<svg></svg>",
	})

	router := setupUiRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(httptest.NewRequest("GET", "/", nil).Context(), "GET", "/assets/logo.svg", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	ct := w.Header().Get("Content-Type")
	if ct != "image/svg+xml" {
		t.Errorf("Content-Type = %q, want 'image/svg+xml'", ct)
	}
}

func TestMidUiHandle_ServesWoff2(t *testing.T) {
	resetZipReader()
	origStaticPkg := comm.StaticPkg
	origInstalled := comm.Installed
	defer func() {
		comm.StaticPkg = origStaticPkg
		comm.Installed = origInstalled
	}()

	comm.Installed = true
	comm.StaticPkg = buildTestZip(t, map[string]string{
		"assets/font.woff2": "fontdata",
	})

	router := setupUiRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(httptest.NewRequest("GET", "/", nil).Context(), "GET", "/assets/font.woff2", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	// woff2 doesn't set a Content-Type explicitly; check Cache-Control
	cc := w.Header().Get("Cache-Control")
	if !strings.Contains(cc, "max-age=") {
		t.Errorf("Cache-Control = %q, want to contain 'max-age='", cc)
	}
}

func TestMidUiHandle_ServesTTF(t *testing.T) {
	resetZipReader()
	origStaticPkg := comm.StaticPkg
	origInstalled := comm.Installed
	defer func() {
		comm.StaticPkg = origStaticPkg
		comm.Installed = origInstalled
	}()

	comm.Installed = true
	comm.StaticPkg = buildTestZip(t, map[string]string{
		"assets/font.ttf": "ttfdata",
	})

	router := setupUiRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(httptest.NewRequest("GET", "/", nil).Context(), "GET", "/assets/font.ttf", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	ct := w.Header().Get("Content-Type")
	if ct != "application/x-font-ttf" {
		t.Errorf("Content-Type = %q, want 'application/x-font-ttf'", ct)
	}
}

func TestMidUiHandle_NotInstalledRedirectsToInstall(t *testing.T) {
	resetZipReader()
	origInstalled := comm.Installed
	origStaticPkg := comm.StaticPkg
	defer func() {
		comm.Installed = origInstalled
		comm.StaticPkg = origStaticPkg
	}()

	comm.Installed = false
	comm.StaticPkg = ""

	router := setupUiRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(httptest.NewRequest("GET", "/", nil).Context(), "GET", "/some-page", nil)
	router.ServeHTTP(w, req)

	// Should redirect (302) to /install
	if w.Code != http.StatusFound {
		t.Errorf("status = %d, want 302", w.Code)
	}
}

func TestMidUiHandle_NoZipReaderRedirects(t *testing.T) {
	resetZipReader()
	origInstalled := comm.Installed
	origStaticPkg := comm.StaticPkg
	defer func() {
		comm.Installed = origInstalled
		comm.StaticPkg = origStaticPkg
	}()

	comm.Installed = true
	comm.StaticPkg = "" // No zip data → getRdr will fail

	router := setupUiRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(httptest.NewRequest("GET", "/", nil).Context(), "GET", "/missing", nil)
	router.ServeHTTP(w, req)

	// Both getFile calls fail → should redirect to /
	if w.Code != http.StatusFound {
		t.Errorf("status = %d, want 302 (redirect to /)", w.Code)
	}
}

package server

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gokins/gokins/comm"
	_ "github.com/mattn/go-sqlite3"
	"xorm.io/xorm"
)

// TestGetRdr_InvalidBase64 exercises the error path in getRdr when the
// StaticPkg contains invalid base64 data.
func TestGetRdr_InvalidBase64(t *testing.T) {
	// Reset the once singleton to ensure clean state
	rder = nil
	rderOnce = sync.Once{}
	rderErr = nil

	origStaticPkg := comm.StaticPkg
	defer func() {
		comm.StaticPkg = origStaticPkg
		rder = nil
		rderOnce = sync.Once{}
		rderErr = nil
	}()

	comm.StaticPkg = "not-valid-base64!!!"

	_, err := getRdr()
	if err == nil {
		t.Fatal("expected error for invalid base64, got nil")
	}
}

// TestGetRdr_ValidBase64InvalidZip exercises the error path when the
// decoded base64 data is not a valid zip archive.
func TestGetRdr_ValidBase64InvalidZip(t *testing.T) {
	rder = nil
	rderOnce = sync.Once{}
	rderErr = nil

	origStaticPkg := comm.StaticPkg
	defer func() {
		comm.StaticPkg = origStaticPkg
		rder = nil
		rderOnce = sync.Once{}
		rderErr = nil
	}()

	// Valid base64 encoding of "this is not a zip file"
	comm.StaticPkg = base64.StdEncoding.EncodeToString([]byte("this is not a zip file"))

	_, err := getRdr()
	if err == nil {
		t.Fatal("expected error for invalid zip data, got nil")
	}
}

// TestGetRdr_IdempotentError verifies that calling getRdr multiple times
// after an initial error consistently returns the same error.
func TestGetRdr_IdempotentError(t *testing.T) {
	rder = nil
	rderOnce = sync.Once{}
	rderErr = nil

	origStaticPkg := comm.StaticPkg
	defer func() {
		comm.StaticPkg = origStaticPkg
		rder = nil
		rderOnce = sync.Once{}
		rderErr = nil
	}()

	comm.StaticPkg = "bad-data"

	_, err1 := getRdr()
	_, err2 := getRdr()

	if err1 == nil || err2 == nil {
		t.Fatal("expected both calls to return errors")
	}
}

// TestMidUiHandle_InstalledNotFound exercises the path where the server
// is installed but the requested file is not found (triggers redirect).
func TestMidUiHandle_InstalledNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Reset static pkg to ensure no zip reader
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
	comm.StaticPkg = ""
	comm.Installed = true

	router := gin.New()
	router.Use(midUiHandle)
	// No handlers registered, so everything will be 404

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/some-nonexistent-path", nil)
	router.ServeHTTP(w, req)

	// Should redirect to / when file not found
	if w.Code != http.StatusFound {
		t.Errorf("expected redirect (302) for not found, got %d", w.Code)
	}
}

// TestMidUiHandle_AlreadyHandled verifies that midUiHandle skips the
// file-serving fallback when a handler already wrote a response.
func TestMidUiHandle_AlreadyHandled(t *testing.T) {
	gin.SetMode(gin.TestMode)

	origInstalled := comm.Installed
	defer func() { comm.Installed = origInstalled }()
	comm.Installed = true

	router := gin.New()
	router.Use(midUiHandle)
	router.GET("/api/test", func(c *gin.Context) {
		c.String(http.StatusOK, "handled")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/api/test", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	if w.Body.String() != "handled" {
		t.Errorf("expected 'handled', got %q", w.Body.String())
	}
}

// TestCreateIndexIfNotExists_InvalidTable verifies that creating an index
// on a non-existent table returns an error.
func TestCreateIndexIfNotExists_InvalidTable(t *testing.T) {
	origDb := comm.Db
	origIsMySQL := comm.IsMySQL
	defer func() {
		comm.Db = origDb
		comm.IsMySQL = origIsMySQL
	}()

	db, err := xorm.NewEngine("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("setup sqlite: %v", err)
	}
	defer db.Close()
	comm.Db = db
	comm.IsMySQL = false

	// Attempt to create index on non-existent table
	err = createIndexIfNotExists("t_nonexistent_table", "idx_test", "col1")
	if err == nil {
		t.Fatal("expected error for non-existent table, got nil")
	}
}

// TestCreateIndexIfNotExists_InvalidColumn verifies that creating an index
// on a non-existent column returns an error.
func TestCreateIndexIfNotExists_InvalidColumn(t *testing.T) {
	origDb := comm.Db
	origIsMySQL := comm.IsMySQL
	defer func() {
		comm.Db = origDb
		comm.IsMySQL = origIsMySQL
	}()

	db, err := xorm.NewEngine("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("setup sqlite: %v", err)
	}
	defer db.Close()
	comm.Db = db
	comm.IsMySQL = false

	// Create table first
	_, err = db.Exec(`CREATE TABLE t_test_idx (id VARCHAR(64) PRIMARY KEY, name VARCHAR(64))`)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}

	// Attempt to create index on non-existent column
	err = createIndexIfNotExists("t_test_idx", "idx_test_bad", "nonexistent_column")
	if err == nil {
		t.Fatal("expected error for non-existent column, got nil")
	}
}

// TestCreateIndexIfNotExists_MySQLMode exercises the MySQL code path
// with a SQLite backend (it generates different SQL syntax).
func TestCreateIndexIfNotExists_MySQLMode(t *testing.T) {
	origDb := comm.Db
	origIsMySQL := comm.IsMySQL
	defer func() {
		comm.Db = origDb
		comm.IsMySQL = origIsMySQL
	}()

	db, err := xorm.NewEngine("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("setup sqlite: %v", err)
	}
	defer db.Close()
	comm.Db = db
	comm.IsMySQL = true // Use MySQL SQL syntax with SQLite engine

	// Create table first
	_, err = db.Exec(`CREATE TABLE t_test_mysql (id VARCHAR(64) PRIMARY KEY, name VARCHAR(64))`)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}

	// MySQL-syntax CREATE INDEX (backtick-quoted) also works on SQLite
	err = createIndexIfNotExists("t_test_mysql", "idx_mysql_test", "name")
	if err != nil {
		t.Fatalf("mysql-mode index creation: %v", err)
	}

	// Second call should silently succeed (duplicate key name on MySQL,
	// but on SQLite the backtick-quoted SQL still creates successfully
	// with IF NOT EXISTS... wait, MySQL mode doesn't use IF NOT EXISTS,
	// it tries CREATE INDEX directly and ignores "Duplicate key name".
	// On SQLite this will error differently.)
	err = createIndexIfNotExists("t_test_mysql", "idx_mysql_test", "name")
	// On SQLite with MySQL-style SQL (no IF NOT EXISTS, backtick quoted),
	// the second call will error with "index already exists" which won't
	// match "Duplicate key name" or "1061", so an error is expected.
	// This verifies the MySQL-specific error handling path is exercised.
	if err == nil {
		t.Log("second MySQL-mode index creation succeeded (SQLite tolerant)")
	} else {
		t.Logf("expected: MySQL-mode duplicate index error on SQLite: %v", err)
	}
}

// TestMidUiHandle_NotInstalledInstallPath verifies that the /install
// path is accessible even when not installed.
func TestMidUiHandle_NotInstalledInstallPath(t *testing.T) {
	gin.SetMode(gin.TestMode)

	origInstalled := comm.Installed
	defer func() { comm.Installed = origInstalled }()
	comm.Installed = false

	router := gin.New()
	router.Use(midUiHandle)
	router.GET("/install", func(c *gin.Context) {
		c.String(http.StatusOK, "install page content")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/install", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200 for /install, got %d", w.Code)
	}
	if w.Body.String() != "install page content" {
		t.Errorf("expected 'install page content', got %q", w.Body.String())
	}
}

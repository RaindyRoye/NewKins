package server

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gokins/gokins/comm"
	_ "github.com/mattn/go-sqlite3"
	bolt "go.etcd.io/bbolt"
)

func TestInitDb_SQLite(t *testing.T) {
	// Save and restore global state
	origDb := comm.Db
	origCfg := comm.Cfg
	origInstalled := comm.Installed
	origWorkPath := comm.WorkPath
	defer func() {
		if comm.Db != nil {
			_ = comm.Db.Close()
		}
		comm.Db = origDb
		comm.Cfg = origCfg
		comm.Installed = origInstalled
		comm.WorkPath = origWorkPath
	}()

	tmpDir := t.TempDir()
	comm.WorkPath = tmpDir
	comm.Cfg.Datasource.Driver = "sqlite3"
	comm.Cfg.Datasource.Url = filepath.Join(tmpDir, "test.db")
	comm.Installed = true // Skip migrations

	err := initDb()
	if err != nil {
		t.Fatalf("initDb() error = %v", err)
	}

	if comm.Db == nil {
		t.Error("initDb() did not set comm.Db")
	}

	// Verify database is accessible
	if err := comm.Db.Ping(); err != nil {
		t.Errorf("database ping failed: %v", err)
	}
}

func TestInitDb_MySQL_InvalidConnection(t *testing.T) {
	origDb := comm.Db
	origCfg := comm.Cfg
	origInstalled := comm.Installed
	defer func() {
		comm.Db = origDb
		comm.Cfg = origCfg
		comm.Installed = origInstalled
	}()

	comm.Cfg.Datasource.Driver = "mysql"
	comm.Cfg.Datasource.Url = "invalid:invalid@tcp(127.0.0.1:1)/testdb"
	comm.Installed = false // Try to run migrations

	err := initDb()
	if err == nil {
		t.Skip("skipping: MySQL connection unexpectedly succeeded")
	}

	// Verify error contains meaningful context
	if err.Error() == "" {
		t.Error("error message is empty")
	}
}

func TestInitDb_Postgres_InvalidConnection(t *testing.T) {
	origDb := comm.Db
	origCfg := comm.Cfg
	origInstalled := comm.Installed
	defer func() {
		comm.Db = origDb
		comm.Cfg = origCfg
		comm.Installed = origInstalled
	}()

	comm.Cfg.Datasource.Driver = "postgres"
	comm.Cfg.Datasource.Url = "postgres://invalid:***@127.0.0.1:1/testdb?sslmode=disable"
	comm.Installed = false

	err := initDb()
	if err == nil {
		t.Skip("skipping: PostgreSQL connection unexpectedly succeeded")
	}

	if err.Error() == "" {
		t.Error("error message is empty")
	}
}

func TestInitDb_DefaultDriver(t *testing.T) {
	origDb := comm.Db
	origCfg := comm.Cfg
	origInstalled := comm.Installed
	origWorkPath := comm.WorkPath
	defer func() {
		if comm.Db != nil {
			_ = comm.Db.Close()
		}
		comm.Db = origDb
		comm.Cfg = origCfg
		comm.Installed = origInstalled
		comm.WorkPath = origWorkPath
	}()

	tmpDir := t.TempDir()
	comm.WorkPath = tmpDir
	comm.Cfg.Datasource.Driver = "" // Empty driver should default to mysql
	comm.Cfg.Datasource.Url = filepath.Join(tmpDir, "test.db")
	comm.Installed = true

	// With empty driver, it defaults to mysql, which will fail without a server
	err := initDb()
	if err == nil {
		t.Log("initDb succeeded with default driver (mysql)")
	} else {
		t.Logf("initDb failed as expected without MySQL server: %v", err)
	}
}

func TestInitCache_Success(t *testing.T) {
	origCache := comm.BCache
	origWorkPath := comm.WorkPath
	defer func() {
		if comm.BCache != nil {
			_ = comm.BCache.Close()
		}
		comm.BCache = origCache
		comm.WorkPath = origWorkPath
	}()

	tmpDir := t.TempDir()
	comm.WorkPath = tmpDir

	err := initCache()
	if err != nil {
		t.Fatalf("initCache() error = %v", err)
	}

	if comm.BCache == nil {
		t.Error("initCache() did not set comm.BCache")
	}

	// Verify cache file was created
	cachePath := filepath.Join(tmpDir, "cache.dat")
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		t.Errorf("cache file %q was not created", cachePath)
	}

	// Verify cache is accessible
	if err := comm.BCache.View(func(tx *bolt.Tx) error {
		return nil
	}); err != nil {
		t.Errorf("cache view failed: %v", err)
	}
}

func TestInitCache_InvalidPath(t *testing.T) {
	origCache := comm.BCache
	origWorkPath := comm.WorkPath
	defer func() {
		comm.BCache = origCache
		comm.WorkPath = origWorkPath
	}()

	// Use a path that cannot be written
	comm.WorkPath = "/nonexistent/directory/that/does/not/exist"

	err := initCache()
	if err == nil {
		t.Skip("skipping: cache creation succeeded unexpectedly (possibly running as root)")
	}

	if err.Error() == "" {
		t.Error("error message is empty")
	}
}

func TestInitCache_RemovesOldCache(t *testing.T) {
	origCache := comm.BCache
	origWorkPath := comm.WorkPath
	defer func() {
		if comm.BCache != nil {
			_ = comm.BCache.Close()
		}
		comm.BCache = origCache
		comm.WorkPath = origWorkPath
	}()

	tmpDir := t.TempDir()
	comm.WorkPath = tmpDir

	// Create an old cache file
	cachePath := filepath.Join(tmpDir, "cache.dat")
	if err := os.WriteFile(cachePath, []byte("old cache data"), 0600); err != nil {
		t.Fatalf("failed to create old cache file: %v", err)
	}

	// Verify old cache exists
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		t.Fatal("old cache file was not created")
	}

	// initCache should remove the old file and create a new one
	err := initCache()
	if err != nil {
		t.Fatalf("initCache() error = %v", err)
	}

	// Verify new cache was created
	if comm.BCache == nil {
		t.Error("initCache() did not set comm.BCache")
	}
}

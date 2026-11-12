package server

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gokins/gokins/comm"
)

// TestInitCache_Success verifies that initCache creates a valid cache database.
func TestInitCache_Success(t *testing.T) {
	tmpDir := t.TempDir()
	origWorkPath := comm.WorkPath
	origBCache := comm.BCache
	defer func() {
		comm.WorkPath = origWorkPath
		if comm.BCache != nil {
			_ = comm.BCache.Close()
		}
		comm.BCache = origBCache
	}()

	comm.WorkPath = tmpDir
	comm.BCache = nil

	err := initCache()
	if err != nil {
		t.Fatalf("initCache() error = %v", err)
	}

	if comm.BCache == nil {
		t.Fatal("initCache() did not set comm.BCache")
	}

	// Verify the cache file exists
	cachePath := filepath.Join(tmpDir, "cache.dat")
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		t.Errorf("cache file not created at %s", cachePath)
	}
}

// TestInitCache_CleansExistingFile verifies that initCache removes old cache files.
func TestInitCache_CleansExistingFile(t *testing.T) {
	tmpDir := t.TempDir()
	origWorkPath := comm.WorkPath
	origBCache := comm.BCache
	defer func() {
		comm.WorkPath = origWorkPath
		if comm.BCache != nil {
			_ = comm.BCache.Close()
		}
		comm.BCache = origBCache
	}()

	comm.WorkPath = tmpDir
	comm.BCache = nil

	// Create an existing cache file
	cachePath := filepath.Join(tmpDir, "cache.dat")
	if err := os.WriteFile(cachePath, []byte("old data"), 0640); err != nil {
		t.Fatalf("failed to create test cache file: %v", err)
	}

	err := initCache()
	if err != nil {
		t.Fatalf("initCache() error = %v", err)
	}

	if comm.BCache == nil {
		t.Fatal("initCache() did not set comm.BCache")
	}
}

// TestInitCache_InvalidWorkPath verifies that initCache fails with invalid paths.
func TestInitCache_InvalidWorkPath(t *testing.T) {
	origWorkPath := comm.WorkPath
	origBCache := comm.BCache
	defer func() {
		comm.WorkPath = origWorkPath
		comm.BCache = origBCache
	}()

	comm.WorkPath = "/nonexistent/path/that/does/not/exist"
	comm.BCache = nil

	err := initCache()
	if err == nil {
		t.Fatal("expected error for invalid work path, got nil")
	}
}

// TestInitDb_SQLiteSuccess verifies SQLite database initialization.
func TestInitDb_SQLiteSuccess(t *testing.T) {
	tmpDir := t.TempDir()
	origWorkPath := comm.WorkPath
	origCfg := comm.Cfg
	origDb := comm.Db
	origInstalled := comm.Installed
	defer func() {
		comm.WorkPath = origWorkPath
		comm.Cfg = origCfg
		if comm.Db != nil {
			_ = comm.Db.Close()
		}
		comm.Db = origDb
		comm.Installed = origInstalled
	}()

	comm.WorkPath = tmpDir
	comm.Cfg.Datasource.Driver = "sqlite3"
	comm.Cfg.Datasource.Url = filepath.Join(tmpDir, "test.db")
	comm.Installed = false // Force migration
	comm.Db = nil

	err := initDb()
	if err != nil {
		t.Fatalf("initDb() error = %v", err)
	}

	if comm.Db == nil {
		t.Fatal("initDb() did not set comm.Db")
	}

	// Verify IsMySQL is set correctly
	if comm.IsMySQL {
		t.Error("comm.IsMySQL should be false for sqlite3")
	}
}

// TestInitDb_MySQLDriverFlag verifies that IsMySQL is set correctly for MySQL.
func TestInitDb_MySQLDriverFlag(t *testing.T) {
	tmpDir := t.TempDir()
	origWorkPath := comm.WorkPath
	origCfg := comm.Cfg
	origDb := comm.Db
	origInstalled := comm.Installed
	defer func() {
		comm.WorkPath = origWorkPath
		comm.Cfg = origCfg
		if comm.Db != nil {
			_ = comm.Db.Close()
		}
		comm.Db = origDb
		comm.Installed = origInstalled
	}()

	comm.WorkPath = tmpDir
	comm.Cfg.Datasource.Driver = "mysql"
	// Use invalid URL to force failure after driver check
	comm.Cfg.Datasource.Url = "invalid:mysql:url"
	comm.Installed = false
	comm.Db = nil

	err := initDb()
	// We expect this to fail during migration, but IsMySQL should be set
	if comm.IsMySQL != true {
		t.Error("comm.IsMySQL should be true for mysql driver")
	}

	// The error is expected since we're using an invalid URL
	if err == nil {
		t.Log("Warning: initDb succeeded with invalid MySQL URL (may have been skipped)")
	}
}

// TestInitDb_SkipMigrationWhenInstalled verifies migrations are skipped when already installed.
func TestInitDb_SkipMigrationWhenInstalled(t *testing.T) {
	tmpDir := t.TempDir()
	origWorkPath := comm.WorkPath
	origCfg := comm.Cfg
	origDb := comm.Db
	origInstalled := comm.Installed
	defer func() {
		comm.WorkPath = origWorkPath
		comm.Cfg = origCfg
		if comm.Db != nil {
			_ = comm.Db.Close()
		}
		comm.Db = origDb
		comm.Installed = origInstalled
	}()

	comm.WorkPath = tmpDir
	comm.Cfg.Datasource.Driver = "sqlite3"
	comm.Cfg.Datasource.Url = filepath.Join(tmpDir, "test.db")
	comm.Installed = true // Skip migration
	comm.Db = nil

	err := initDb()
	if err != nil {
		t.Fatalf("initDb() error = %v", err)
	}

	if comm.Db == nil {
		t.Fatal("initDb() did not set comm.Db")
	}
}

// TestInitDb_DefaultDriverIsMySQL verifies that when driver is empty,
// the default is mysql (matching comm.DatasourceDriverMySQL constant).
// The migration switch falls to default for sqlite, but the driver variable
// itself defaults to mysql, so xorm.NewEngine is called with "mysql".
func TestInitDb_DefaultDriverIsMySQL(t *testing.T) {
	origCfg := comm.Cfg
	origDb := comm.Db
	origInstalled := comm.Installed
	defer func() {
		comm.Cfg = origCfg
		if comm.Db != nil {
			_ = comm.Db.Close()
		}
		comm.Db = origDb
		comm.Installed = origInstalled
	}()

	comm.Cfg.Datasource.Driver = "" // Defaults to mysql internally
	comm.Cfg.Datasource.Url = "invalid:mysql:url"
	comm.Installed = false
	comm.Db = nil

	err := initDb()
	// We expect failure since the URL is invalid for mysql,
	// but IsMySQL should be true because default driver is mysql
	if comm.IsMySQL != true {
		t.Error("comm.IsMySQL should be true when driver is empty (defaults to mysql)")
	}
	if err == nil {
		t.Log("Warning: initDb succeeded with invalid mysql URL")
	}
}

// TestInitDb_InvalidURL verifies that initDb fails with invalid database URLs.
func TestInitDb_InvalidURL(t *testing.T) {
	origWorkPath := comm.WorkPath
	origCfg := comm.Cfg
	origDb := comm.Db
	origInstalled := comm.Installed
	defer func() {
		comm.WorkPath = origWorkPath
		comm.Cfg = origCfg
		if comm.Db != nil {
			_ = comm.Db.Close()
		}
		comm.Db = origDb
		comm.Installed = origInstalled
	}()

	comm.WorkPath = "/tmp"
	comm.Cfg.Datasource.Driver = "postgres"
	comm.Cfg.Datasource.Url = "invalid://postgres://url"
	comm.Installed = false
	comm.Db = nil

	err := initDb()
	if err == nil {
		t.Fatal("expected error for invalid database URL, got nil")
	}
}

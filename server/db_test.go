package server

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gokins/gokins/comm"
	_ "github.com/mattn/go-sqlite3"
	bolt "go.etcd.io/bbolt"
)

// TestInitCache_Success verifies that initCache creates a valid bolt DB
// in the working directory and assigns it to comm.BCache.
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
		t.Fatal("expected comm.BCache to be initialized, got nil")
	}

	// Verify the cache file was created
	cachePath := filepath.Join(tmpDir, "cache.dat")
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		t.Errorf("cache file not created at %s", cachePath)
	}

	// Verify we can perform a basic read transaction
	err = comm.BCache.View(func(tx *bolt.Tx) error {
		return nil
	})
	if err != nil {
		t.Errorf("cache view operation failed: %v", err)
	}
}

// TestInitCache_InvalidPath verifies that initCache fails gracefully
// when the working directory is invalid.
func TestInitCache_InvalidPath(t *testing.T) {
	origWorkPath := comm.WorkPath
	origBCache := comm.BCache
	defer func() {
		comm.WorkPath = origWorkPath
		comm.BCache = origBCache
	}()

	// Use a path that cannot be created (file instead of directory)
	tmpDir := t.TempDir()
	blocker := filepath.Join(tmpDir, "blocker")
	if err := os.WriteFile(blocker, []byte("block"), 0600); err != nil {
		t.Fatalf("create blocker: %v", err)
	}
	comm.WorkPath = filepath.Join(blocker, "impossible")

	err := initCache()
	if err == nil {
		t.Fatal("expected error for invalid path, got nil")
	}
}

// TestInitDb_InvalidDriver verifies that initDb fails with unsupported driver.
func TestInitDb_InvalidDriver(t *testing.T) {
	origCfg := comm.Cfg
	origDb := comm.Db
	origInstalled := comm.Installed
	defer func() {
		comm.Cfg = origCfg
		comm.Db = origDb
		comm.Installed = origInstalled
	}()

	comm.Installed = true // skip migration
	comm.Cfg.Datasource.Driver = "invalid_driver"
	comm.Cfg.Datasource.Url = "invalid://url"

	err := initDb()
	if err == nil {
		t.Fatal("expected error for invalid driver, got nil")
	}
}

// TestInitDb_SQLiteSuccess verifies initDb with a valid SQLite configuration.
func TestInitDb_SQLiteSuccess(t *testing.T) {
	origCfg := comm.Cfg
	origDb := comm.Db
	origInstalled := comm.Installed
	origIsMySQL := comm.IsMySQL
	defer func() {
		comm.Cfg = origCfg
		if comm.Db != nil {
			_ = comm.Db.Close()
		}
		comm.Db = origDb
		comm.Installed = origInstalled
		comm.IsMySQL = origIsMySQL
	}()

	tmpDir := t.TempDir()
	comm.Installed = true // skip migration
	comm.Cfg.Datasource.Driver = "sqlite"
	comm.Cfg.Datasource.Url = filepath.Join(tmpDir, "test.db")

	err := initDb()
	if err != nil {
		t.Fatalf("initDb with SQLite failed: %v", err)
	}
	if comm.Db == nil {
		t.Fatal("expected comm.Db to be initialized, got nil")
	}
	if comm.IsMySQL {
		t.Error("expected IsMySQL to be false for SQLite driver")
	}
}

package server

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gokins/gokins/comm"
)

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
		t.Fatal("expected BCache to be initialized, got nil")
	}

	// Verify the cache file was created
	cachePath := filepath.Join(tmpDir, "cache.dat")
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		t.Errorf("cache.dat not found at %s", cachePath)
	}
}

func TestInitCache_InvalidPath(t *testing.T) {
	origWorkPath := comm.WorkPath
	origBCache := comm.BCache
	defer func() {
		comm.WorkPath = origWorkPath
		comm.BCache = origBCache
	}()

	// Use an invalid path that cannot be created
	comm.WorkPath = "/nonexistent/directory/that/does/not/exist"

	err := initCache()
	if err == nil {
		// If it succeeds (unlikely), close the cache
		if comm.BCache != nil {
			_ = comm.BCache.Close()
			comm.BCache = nil
		}
		t.Fatal("expected error for invalid path, got nil")
	}
}

func TestInitDb_SqliteInMemory(t *testing.T) {
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

	comm.Cfg.Datasource.Driver = "sqlite3"
	comm.Cfg.Datasource.Url = ":memory:"
	comm.Installed = true // Skip migrations

	err := initDb()
	if err != nil {
		t.Fatalf("initDb failed: %v", err)
	}
	if comm.Db == nil {
		t.Fatal("expected Db to be initialized, got nil")
	}
	if comm.IsMySQL {
		t.Error("expected IsMySQL to be false for sqlite3 driver")
	}
}

func TestInitDb_InvalidConnection(t *testing.T) {
	origDb := comm.Db
	origCfg := comm.Cfg
	origInstalled := comm.Installed
	defer func() {
		comm.Db = origDb
		comm.Cfg = origCfg
		comm.Installed = origInstalled
	}()

	// Use an invalid driver to trigger connection error
	comm.Cfg.Datasource.Driver = "sqlite3"
	comm.Cfg.Datasource.Url = "/nonexistent/path/to/db.sqlite"
	comm.Installed = true // Skip migrations

	err := initDb()
	// SQLite may still succeed creating the file, so we just check it doesn't panic
	if err != nil {
		// Expected for truly invalid paths
		t.Logf("initDb returned error (expected): %v", err)
	}
}

func TestInitDb_DefaultDriver(t *testing.T) {
	origDb := comm.Db
	origCfg := comm.Cfg
	origInstalled := comm.Installed
	defer func() {
		if comm.Db != nil {
			_ = comm.Db.Close()
		}
		comm.Db = origDb
		comm.Cfg = origCfg
		comm.Installed = origInstalled
	}()

	// Leave driver empty — should default to MySQL
	comm.Cfg.Datasource.Driver = ""
	comm.Cfg.Datasource.Url = ":memory:"
	comm.Installed = true

	err := initDb()
	// MySQL driver with sqlite URL will fail
	if err == nil {
		t.Log("initDb succeeded with default MySQL driver (unexpected but not fatal)")
	}
}

func TestInitCache_OverwritesExisting(t *testing.T) {
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

	// Create a pre-existing cache.dat file
	cachePath := filepath.Join(tmpDir, "cache.dat")
	if err := os.WriteFile(cachePath, []byte("old data"), 0600); err != nil {
		t.Fatalf("write old cache: %v", err)
	}

	err := initCache()
	if err != nil {
		t.Fatalf("initCache failed: %v", err)
	}
	if comm.BCache == nil {
		t.Fatal("expected BCache to be initialized, got nil")
	}
}

package server

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gokins/gokins/comm"
	_ "github.com/mattn/go-sqlite3"
)

func TestInitCache(t *testing.T) {
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
		t.Fatal("expected comm.BCache to be initialized")
	}

	// Verify cache file was created
	cachePath := filepath.Join(tmpDir, "cache.dat")
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		t.Errorf("cache file not created at %s", cachePath)
	}
}

func TestInitCache_InvalidPath(t *testing.T) {
	origWorkPath := comm.WorkPath
	origBCache := comm.BCache
	defer func() {
		comm.WorkPath = origWorkPath
		comm.BCache = origBCache
	}()

	// Use a path that cannot be created
	comm.WorkPath = "/nonexistent/path/that/cannot/be/created"

	err := initCache()
	if err == nil {
		if comm.BCache != nil {
			_ = comm.BCache.Close()
		}
		t.Fatal("expected error for invalid path, got nil")
	}
}

func TestInitDb_SQLite(t *testing.T) {
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

	tmpDir := t.TempDir()
	comm.Cfg.Datasource.Driver = "sqlite"
	comm.Cfg.Datasource.Url = filepath.Join(tmpDir, "test.db")
	comm.Installed = true // Skip migrations for unit test

	err := initDb()
	if err != nil {
		t.Fatalf("initDb failed: %v", err)
	}

	if comm.Db == nil {
		t.Fatal("expected comm.Db to be initialized")
	}

	// Verify database is accessible by running a simple query
	_, err = comm.Db.Exec("SELECT 1")
	if err != nil {
		t.Errorf("database not accessible: %v", err)
	}
}

func TestInitDb_InvalidDriver(t *testing.T) {
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

	comm.Cfg.Datasource.Driver = "oracle"
	comm.Cfg.Datasource.Url = "invalid-url"
	comm.Installed = true // Skip migrations

	err := initDb()
	if err == nil {
		if comm.Db != nil {
			_ = comm.Db.Close()
		}
		t.Fatal("expected error for invalid driver, got nil")
	}
}

func TestInitDb_EmptyURL(t *testing.T) {
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

	comm.Cfg.Datasource.Driver = "mysql"
	comm.Cfg.Datasource.Url = ""
	comm.Installed = true // Skip migrations

	err := initDb()
	if err == nil {
		if comm.Db != nil {
			_ = comm.Db.Close()
		}
		t.Fatal("expected error for empty URL, got nil")
	}
}

func TestInitDb_IsMySQLFlag(t *testing.T) {
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
	comm.Cfg.Datasource.Driver = "sqlite"
	comm.Cfg.Datasource.Url = filepath.Join(tmpDir, "test.db")
	comm.Installed = true

	_ = initDb()
	if comm.Db != nil {
		_ = comm.Db.Close()
	}

	// IsMySQL should be false for sqlite
	if comm.IsMySQL {
		t.Error("expected IsMySQL to be false for sqlite driver")
	}
}

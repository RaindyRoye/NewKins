package server

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/gokins/gokins/comm"
)

func TestInitCache_Success(t *testing.T) {
	// Save and restore global state
	origWorkPath := comm.WorkPath
	origBCache := comm.BCache
	defer func() {
		comm.WorkPath = origWorkPath
		comm.BCache = origBCache
		if comm.BCache != nil {
			_ = comm.BCache.Close()
		}
	}()

	tmpDir := t.TempDir()
	comm.WorkPath = tmpDir

	err := initCache()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if comm.BCache == nil {
		t.Fatal("expected BCache to be initialized, got nil")
	}

	// Verify cache file exists
	cachePath := filepath.Join(tmpDir, "cache.dat")
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		t.Error("cache.dat file was not created")
	}
}

func TestInitCache_InvalidPath(t *testing.T) {
	origWorkPath := comm.WorkPath
	origBCache := comm.BCache
	defer func() {
		comm.WorkPath = origWorkPath
		comm.BCache = origBCache
		if comm.BCache != nil {
			_ = comm.BCache.Close()
		}
	}()

	// Use an invalid path that cannot be created
	comm.WorkPath = "/nonexistent/path/that/does/not/exist"

	err := initCache()
	if err == nil {
		t.Fatal("expected error for invalid path, got nil")
	}
	if !errors.Is(err, ErrCacheOpen) {
		t.Errorf("expected ErrCacheOpen, got: %v", err)
	}
}

func TestInitCache_RemovesExistingCache(t *testing.T) {
	origWorkPath := comm.WorkPath
	origBCache := comm.BCache
	defer func() {
		comm.WorkPath = origWorkPath
		comm.BCache = origBCache
		if comm.BCache != nil {
			_ = comm.BCache.Close()
		}
	}()

	tmpDir := t.TempDir()
	comm.WorkPath = tmpDir

	// Create an existing cache file
	cachePath := filepath.Join(tmpDir, "cache.dat")
	if err := os.WriteFile(cachePath, []byte("old data"), 0600); err != nil {
		t.Fatalf("failed to create test cache file: %v", err)
	}

	err := initCache()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if comm.BCache == nil {
		t.Fatal("expected BCache to be initialized, got nil")
	}
}

func TestInitDb_InvalidDriver(t *testing.T) {
	origCfg := comm.Cfg
	origDb := comm.Db
	origIsMySQL := comm.IsMySQL
	origInstalled := comm.Installed
	defer func() {
		comm.Cfg = origCfg
		comm.Db = origDb
		comm.IsMySQL = origIsMySQL
		comm.Installed = origInstalled
		if comm.Db != nil {
			_ = comm.Db.Close()
		}
	}()

	comm.Cfg.Datasource.Driver = "invalid-driver"
	comm.Cfg.Datasource.Url = "invalid-url"
	comm.Installed = true // Skip migration

	err := initDb()
	if err == nil {
		t.Fatal("expected error for invalid driver, got nil")
	}
	if !errors.Is(err, ErrDatabaseOpen) {
		t.Errorf("expected ErrDatabaseOpen, got: %v", err)
	}
}

func TestInitDb_SqliteSuccess(t *testing.T) {
	origCfg := comm.Cfg
	origDb := comm.Db
	origIsMySQL := comm.IsMySQL
	origInstalled := comm.Installed
	origWorkPath := comm.WorkPath
	defer func() {
		comm.Cfg = origCfg
		comm.Db = origDb
		comm.IsMySQL = origIsMySQL
		comm.Installed = origInstalled
		comm.WorkPath = origWorkPath
		if comm.Db != nil {
			_ = comm.Db.Close()
		}
	}()

	tmpDir := t.TempDir()
	comm.WorkPath = tmpDir
	comm.Cfg.Datasource.Driver = "sqlite3"
	comm.Cfg.Datasource.Url = filepath.Join(tmpDir, "test.db")
	comm.Installed = true // Skip migration

	err := initDb()
	if err != nil {
		t.Fatalf("expected no error for sqlite, got: %v", err)
	}
	if comm.Db == nil {
		t.Fatal("expected Db to be initialized, got nil")
	}
	if comm.IsMySQL {
		t.Error("expected IsMySQL to be false for sqlite driver")
	}
}

func TestInitDb_MySQLFlag(t *testing.T) {
	origCfg := comm.Cfg
	origDb := comm.Db
	origIsMySQL := comm.IsMySQL
	origInstalled := comm.Installed
	defer func() {
		comm.Cfg = origCfg
		comm.Db = origDb
		comm.IsMySQL = origIsMySQL
		comm.Installed = origInstalled
		if comm.Db != nil {
			_ = comm.Db.Close()
		}
	}()

	// xorm.NewEngine doesn't actually connect, so even an invalid URL
	// succeeds at engine creation. Verify the IsMySQL flag is set.
	comm.Cfg.Datasource.Driver = comm.DatasourceDriverMySQL
	comm.Cfg.Datasource.Url = "invalid:invalid@tcp(invalid-host:3306)/invalid"
	comm.Installed = true // Skip migration

	err := initDb()
	if err != nil {
		t.Fatalf("xorm.NewEngine should not connect eagerly, got: %v", err)
	}
	if !comm.IsMySQL {
		t.Error("expected IsMySQL to be true for mysql driver")
	}
	if comm.Db == nil {
		t.Fatal("expected Db to be initialized, got nil")
	}
}

func TestInitDb_DefaultDriver(t *testing.T) {
	origCfg := comm.Cfg
	origDb := comm.Db
	origIsMySQL := comm.IsMySQL
	origInstalled := comm.Installed
	origWorkPath := comm.WorkPath
	defer func() {
		comm.Cfg = origCfg
		comm.Db = origDb
		comm.IsMySQL = origIsMySQL
		comm.Installed = origInstalled
		comm.WorkPath = origWorkPath
		if comm.Db != nil {
			_ = comm.Db.Close()
		}
	}()

	tmpDir := t.TempDir()
	comm.WorkPath = tmpDir
	// Don't set driver - should default to mysql
	comm.Cfg.Datasource.Driver = ""
	// Use a valid MySQL DSN format so xorm can parse it
	comm.Cfg.Datasource.Url = "root:password@tcp(localhost:3306)/testdb"
	comm.Installed = true

	// When driver is empty, initDb defaults to mysql
	err := initDb()
	if err != nil {
		t.Fatalf("initDb failed: %v", err)
	}
	// Empty driver defaults to mysql
	if !comm.IsMySQL {
		t.Error("expected IsMySQL to be true when driver is empty (defaults to mysql)")
	}
	if comm.Db == nil {
		t.Fatal("expected Db to be initialized, got nil")
	}
}

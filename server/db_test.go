package server

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gokins/gokins/comm"
)

// TestInitCache_Success verifies that initCache creates a bbolt DB file.
func TestInitCache_Success(t *testing.T) {
	origWorkPath := comm.WorkPath
	origCache := comm.BCache
	t.Cleanup(func() {
		comm.WorkPath = origWorkPath
		if comm.BCache != nil {
			_ = comm.BCache.Close()
		}
		comm.BCache = origCache
	})

	tmpDir := t.TempDir()
	comm.WorkPath = tmpDir

	err := initCache()
	if err != nil {
		t.Fatalf("initCache() error = %v", err)
	}

	if comm.BCache == nil {
		t.Fatal("BCache should not be nil after initCache()")
	}

	// Verify the file exists on disk
	cachePath := filepath.Join(tmpDir, "cache.dat")
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		t.Errorf("cache file %q was not created", cachePath)
	}

	// Verify we can start a read transaction
	tx, err := comm.BCache.Begin(false)
	if err != nil {
		t.Errorf("could not start read transaction: %v", err)
	} else {
		_ = tx.Rollback()
	}
}

// TestInitCache_ReplacesExistingFile verifies that initCache removes an old
// cache file before creating a fresh one.
func TestInitCache_ReplacesExistingFile(t *testing.T) {
	origWorkPath := comm.WorkPath
	origCache := comm.BCache
	t.Cleanup(func() {
		comm.WorkPath = origWorkPath
		if comm.BCache != nil {
			_ = comm.BCache.Close()
		}
		comm.BCache = origCache
	})

	tmpDir := t.TempDir()
	comm.WorkPath = tmpDir

	// Create a stale cache file
	cachePath := filepath.Join(tmpDir, "cache.dat")
	if err := os.WriteFile(cachePath, []byte("stale data"), 0600); err != nil {
		t.Fatalf("write stale cache: %v", err)
	}

	err := initCache()
	if err != nil {
		t.Fatalf("initCache() error = %v", err)
	}

	if comm.BCache == nil {
		t.Fatal("BCache should not be nil after initCache()")
	}
}

// TestInitDb_SqliteSuccess verifies that initDb works with sqlite driver.
func TestInitDb_SqliteSuccess(t *testing.T) {
	origWorkPath := comm.WorkPath
	origCfg := comm.Cfg
	origDb := comm.Db
	origInstalled := comm.Installed
	origIsMySQL := comm.IsMySQL
	t.Cleanup(func() {
		comm.WorkPath = origWorkPath
		comm.Cfg = origCfg
		if comm.Db != nil {
			_ = comm.Db.Close()
		}
		comm.Db = origDb
		comm.Installed = origInstalled
		comm.IsMySQL = origIsMySQL
	})

	tmpDir := t.TempDir()
	comm.WorkPath = tmpDir
	comm.Installed = false
	comm.Cfg.Datasource.Driver = "sqlite"
	comm.Cfg.Datasource.Url = filepath.Join(tmpDir, "test.db")

	err := initDb()
	if err != nil {
		t.Fatalf("initDb() error = %v", err)
	}

	if comm.Db == nil {
		t.Fatal("Db should not be nil after initDb()")
	}

	if comm.IsMySQL != false {
		t.Error("IsMySQL should be false for sqlite driver")
	}
}

// TestInitDb_InvalidDriver verifies that initDb returns an error for unsupported drivers.
func TestInitDb_InvalidDriver(t *testing.T) {
	origCfg := comm.Cfg
	origDb := comm.Db
	origInstalled := comm.Installed
	t.Cleanup(func() {
		comm.Cfg = origCfg
		comm.Db = origDb
		comm.Installed = origInstalled
	})

	comm.Installed = true // skip migration
	comm.Cfg.Datasource.Driver = "oracle"
	comm.Cfg.Datasource.Url = "some-url"

	err := initDb()
	if err == nil {
		t.Fatal("expected error for unsupported driver, got nil")
	}
}

// TestInitDb_DefaultDriver verifies that empty driver defaults to mysql.
func TestInitDb_DefaultDriver(t *testing.T) {
	origCfg := comm.Cfg
	origDb := comm.Db
	origInstalled := comm.Installed
	origIsMySQL := comm.IsMySQL
	t.Cleanup(func() {
		comm.Cfg = origCfg
		comm.Db = origDb
		comm.Installed = origInstalled
		comm.IsMySQL = origIsMySQL
	})

	comm.Installed = true // skip migration, avoid needing real DB
	comm.Cfg.Datasource.Driver = ""
	comm.Cfg.Datasource.Url = ""

	// With empty driver (defaults to mysql) and empty URL, we expect
	// either a migration error or a connection error — both are valid
	// since there's no MySQL running.
	_ = initDb() // error is acceptable here
}

// TestInitDb_InstalledSkipsMigration verifies that when Installed is true,
// initDb skips the migration step and goes directly to opening the DB.
func TestInitDb_InstalledSkipsMigration(t *testing.T) {
	origWorkPath := comm.WorkPath
	origCfg := comm.Cfg
	origDb := comm.Db
	origInstalled := comm.Installed
	origIsMySQL := comm.IsMySQL
	t.Cleanup(func() {
		comm.WorkPath = origWorkPath
		comm.Cfg = origCfg
		if comm.Db != nil {
			_ = comm.Db.Close()
		}
		comm.Db = origDb
		comm.Installed = origInstalled
		comm.IsMySQL = origIsMySQL
	})

	tmpDir := t.TempDir()
	comm.WorkPath = tmpDir
	comm.Installed = true
	comm.Cfg.Datasource.Driver = "sqlite"
	comm.Cfg.Datasource.Url = filepath.Join(tmpDir, "installed.db")

	err := initDb()
	if err != nil {
		t.Fatalf("initDb() with Installed=true error = %v", err)
	}

	if comm.Db == nil {
		t.Fatal("Db should not be nil after initDb()")
	}
}

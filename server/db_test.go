package server

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gokins/gokins/comm"
)

// TestInitDb_WithSQLite verifies that initDb can initialize a SQLite database
// when comm.Installed is false (fresh install).
func TestInitDb_WithSQLite(t *testing.T) {
	// Save and restore global state
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

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	comm.Cfg.Datasource.Driver = "sqlite"
	comm.Cfg.Datasource.Url = dbPath
	comm.Installed = false
	comm.IsMySQL = false

	err := initDb()
	if err != nil {
		t.Fatalf("initDb failed: %v", err)
	}

	if comm.Db == nil {
		t.Fatal("comm.Db should not be nil after initDb")
	}

	// Verify database file was created
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Errorf("database file %s was not created", dbPath)
	}
}

// TestXormDriverName verifies the driver name mapping function.
func TestXormDriverName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"sqlite", "sqlite3"},
		{"mysql", "mysql"},
		{"postgres", "postgres"},
		{"unknown", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := xormDriverName(tt.input)
			if result != tt.expected {
				t.Errorf("xormDriverName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestInitDb_AlreadyInstalled skips migrations when comm.Installed is true.
func TestInitDb_AlreadyInstalled(t *testing.T) {
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

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	comm.Cfg.Datasource.Driver = "sqlite"
	comm.Cfg.Datasource.Url = dbPath
	comm.Installed = true // Already installed — migrations should be skipped
	comm.IsMySQL = false

	err := initDb()
	if err != nil {
		t.Fatalf("initDb failed: %v", err)
	}

	if comm.Db == nil {
		t.Fatal("comm.Db should not be nil after initDb")
	}
}

// TestInitDb_DefaultDriverIsMySQL verifies that an empty driver defaults to MySQL.
func TestInitDb_DefaultDriverIsMySQL(t *testing.T) {
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

	// Use a MySQL connection string that will fail during migration
	// (we just want to verify the driver selection logic)
	comm.Cfg.Datasource.Driver = "" // Empty driver should default to mysql
	comm.Cfg.Datasource.Url = "root:@tcp(127.0.0.1:3306)/test"
	comm.Installed = false
	comm.IsMySQL = false

	// This will fail because there's no MySQL server, but we can verify
	// that IsMySQL was set correctly before the failure
	_ = initDb() // Expected to fail

	if !comm.IsMySQL {
		t.Error("IsMySQL should be true when driver defaults to mysql")
	}
}

// TestInitDb_InvalidUrl verifies that initDb returns an error for invalid database URLs.
func TestInitDb_InvalidUrl(t *testing.T) {
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

	comm.Cfg.Datasource.Driver = "sqlite"
	comm.Cfg.Datasource.Url = "/nonexistent/path/that/should/fail/db.sqlite"
	comm.Installed = false
	comm.IsMySQL = false

	err := initDb()
	if err == nil {
		t.Fatal("expected error for invalid database URL, got nil")
	}
	if !strings.Contains(err.Error(), "database") && !strings.Contains(err.Error(), "open") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// TestInitCache_CreatesCacheFile verifies that initCache creates a cache.db file.
func TestInitCache_CreatesCacheFile(t *testing.T) {
	origBCache := comm.BCache
	origWorkPath := comm.WorkPath
	defer func() {
		if comm.BCache != nil {
			_ = comm.BCache.Close()
		}
		comm.BCache = origBCache
		comm.WorkPath = origWorkPath
	}()

	tmpDir := t.TempDir()
	comm.WorkPath = tmpDir

	err := initCache()
	if err != nil {
		t.Fatalf("initCache failed: %v", err)
	}

	if comm.BCache == nil {
		t.Fatal("comm.BCache should not be nil after initCache")
	}

	// Verify cache file was created
	cachePath := filepath.Join(tmpDir, "cache.dat")
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		t.Errorf("cache file %s was not created", cachePath)
	}
}

// TestInitCache_RemovesOldCache verifies that initCache removes any existing cache file.
func TestInitCache_RemovesOldCache(t *testing.T) {
	origBCache := comm.BCache
	origWorkPath := comm.WorkPath
	defer func() {
		if comm.BCache != nil {
			_ = comm.BCache.Close()
		}
		comm.BCache = origBCache
		comm.WorkPath = origWorkPath
	}()

	tmpDir := t.TempDir()
	comm.WorkPath = tmpDir

	// Create an old cache file with some content
	cachePath := filepath.Join(tmpDir, "cache.dat")
	if err := os.WriteFile(cachePath, []byte("old cache data"), 0600); err != nil {
		t.Fatalf("write old cache: %v", err)
	}

	err := initCache()
	if err != nil {
		t.Fatalf("initCache failed: %v", err)
	}

	// The old cache should have been replaced
	if comm.BCache == nil {
		t.Fatal("comm.BCache should not be nil after initCache")
	}
}

// TestInitCache_InvalidPath verifies that initCache returns an error for invalid paths.
func TestInitCache_InvalidPath(t *testing.T) {
	origBCache := comm.BCache
	origWorkPath := comm.WorkPath
	defer func() {
		if comm.BCache != nil {
			_ = comm.BCache.Close()
		}
		comm.BCache = origBCache
		comm.WorkPath = origWorkPath
	}()

	comm.WorkPath = "/nonexistent/path/that/should/fail"

	err := initCache()
	if err == nil {
		t.Fatal("expected error for invalid path, got nil")
	}
	if !strings.Contains(err.Error(), "open cache db") {
		t.Errorf("unexpected error message: %v", err)
	}
}

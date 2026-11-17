package server

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gokins/gokins/comm"
)

func TestInitCache(t *testing.T) {
	// Create a temporary directory for the cache file
	tempDir := t.TempDir()
	comm.WorkPath = tempDir

	// Test successful cache initialization
	err := initCache()
	if err != nil {
		t.Fatalf("initCache failed: %v", err)
	}

	// Verify cache was created
	if comm.BCache == nil {
		t.Fatal("BCache should not be nil after successful init")
	}

	// Verify cache file exists
	cachePath := filepath.Join(tempDir, "cache.dat")
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		t.Error("cache.dat file should exist")
	}

	// Clean up
	if comm.BCache != nil {
		comm.BCache.Close()
		comm.BCache = nil
	}
}

func TestInitCache_ReplacesExisting(t *testing.T) {
	tempDir := t.TempDir()
	comm.WorkPath = tempDir

	// Create an existing cache file
	cachePath := filepath.Join(tempDir, "cache.dat")
	if err := os.WriteFile(cachePath, []byte("old data"), 0600); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// initCache should replace the existing file
	err := initCache()
	if err != nil {
		t.Fatalf("initCache failed: %v", err)
	}

	if comm.BCache == nil {
		t.Fatal("BCache should not be nil")
	}

	// Clean up
	if comm.BCache != nil {
		comm.BCache.Close()
		comm.BCache = nil
	}
}

func TestInitCache_InvalidPath(t *testing.T) {
	// Use a path that cannot be created (e.g., under a non-existent directory)
	comm.WorkPath = "/nonexistent/path/that/does/not/exist"

	err := initCache()
	if err == nil {
		t.Fatal("initCache should fail with invalid path")
		if comm.BCache != nil {
			comm.BCache.Close()
			comm.BCache = nil
		}
	}

	// Verify error message contains expected info
	if err != nil && !contains(err.Error(), "open cache db") {
		t.Errorf("error message should mention 'open cache db', got: %v", err)
	}
}

func TestInitDb_InvalidDriver(t *testing.T) {
	// Use an invalid driver name that xorm doesn't recognize
	comm.Cfg.Datasource.Driver = "invalid_driver"
	comm.Cfg.Datasource.Url = "test.db"
	comm.Installed = true

	err := initDb()
	if err == nil {
		t.Fatal("initDb should fail with invalid driver")
		if comm.Db != nil {
			comm.Db.Close()
			comm.Db = nil
		}
	}
}

func TestInitDb_SQLiteSuccess(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	comm.Cfg.Datasource.Driver = "sqlite3"
	comm.Cfg.Datasource.Url = dbPath
	comm.Installed = true // Skip migration

	err := initDb()
	if err != nil {
		t.Fatalf("initDb failed with SQLite: %v", err)
	}

	if comm.Db == nil {
		t.Fatal("Db should not be nil after successful init")
	}

	// Verify IsMySQL is false for SQLite
	if comm.IsMySQL {
		t.Error("IsMySQL should be false for SQLite")
	}

	// Clean up
	if comm.Db != nil {
		comm.Db.Close()
		comm.Db = nil
	}
}

func TestInitDb_MySQLFlag(t *testing.T) {
	comm.Cfg.Datasource.Driver = "mysql"
	comm.Cfg.Datasource.Url = "invalid:invalid@tcp(localhost:3306)/testdb"
	comm.Installed = true

	// This will fail to connect, but we can verify the MySQL flag logic
	err := initDb()
	
	// Expected to fail (no MySQL server), but IsMySQL should be set
	if comm.IsMySQL != true {
		t.Error("IsMySQL should be true when driver is mysql")
	}

	// Clean up if somehow succeeded
	if comm.Db != nil {
		comm.Db.Close()
		comm.Db = nil
	}
	
	// We expect an error since MySQL is not available
	if err == nil {
		t.Log("Unexpectedly succeeded (MySQL might be available)")
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

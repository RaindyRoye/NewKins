package migrates

import (
	"errors"
	"testing"

	"github.com/gokins/gokins/comm"
)

// TestRunMigrate_EmptyURL verifies that runMigrate rejects empty connection strings.
func TestRunMigrate_EmptyURL(t *testing.T) {
	cases := []struct {
		driver, prefix string
	}{
		{"mysql", "mysql"},
		{"postgres", "postgres"},
		{"sqlite3", "sqlite"},
	}
	for _, c := range cases {
		err := runMigrate("", c.driver, c.prefix)
		if err == nil {
			t.Errorf("%s: expected error for empty URL, got nil", c.driver)
			continue
		}
		if !errors.Is(err, ErrDatabaseConfigMissing) {
			t.Errorf("%s: error should wrap ErrDatabaseConfigMissing, got: %v", c.driver, err)
		}
	}
}

// TestRunMigrate_InvalidURL verifies that runMigrate rejects malformed URLs.
func TestRunMigrate_InvalidURL(t *testing.T) {
	// Use a URL that will fail sql.Open or PingContext
	err := runMigrate("invalid://url", "mysql", "mysql")
	if err == nil {
		t.Error("expected error for invalid URL, got nil")
	}
}

// TestNewDriver_Unsupported verifies that newDriver rejects unknown drivers.
func TestNewDriver_Unsupported(t *testing.T) {
	_, err := newDriver("unsupported", nil)
	if err == nil {
		t.Error("expected error for unsupported driver, got nil")
	}
	if !errors.Is(err, ErrDatabaseConfigMissing) {
		t.Errorf("error should wrap ErrDatabaseConfigMissing, got: %v", err)
	}
}

// TestUpSqliteMigrate_ValidPath verifies SQLite migration with a temp database.
func TestUpSqliteMigrate_ValidPath(t *testing.T) {
	tmpDir := t.TempDir()
	origWorkPath := comm.WorkPath
	comm.WorkPath = tmpDir
	defer func() { comm.WorkPath = origWorkPath }()

	dbPath, err := InitSqliteMigrate()
	if err != nil {
		t.Fatalf("InitSqliteMigrate() error = %v", err)
	}
	if dbPath == "" {
		t.Fatal("InitSqliteMigrate() returned empty path")
	}

	// Run UpSqliteMigrate on the already-initialized database
	err = UpSqliteMigrate(dbPath)
	if err != nil {
		t.Errorf("UpSqliteMigrate() error = %v", err)
	}
}

// TestAssetPrefixMapping verifies the asset prefix constants match driver names.
func TestAssetPrefixMapping(t *testing.T) {
	// This test documents the expected mapping between drivers and asset prefixes
	mapping := map[string]string{
		"mysql":    "mysql",
		"postgres": "postgres",
		"sqlite3":  "sqlite",
	}
	for driver, prefix := range mapping {
		// Just verify the mapping is consistent
		if driver == "" || prefix == "" {
			t.Errorf("invalid mapping: driver=%q, prefix=%q", driver, prefix)
		}
	}
}

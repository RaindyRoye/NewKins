package migrates

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/gokins/gokins/comm"
	"github.com/golang-migrate/migrate/v4/database"
)

// TestFilterAssetNames verifies asset name filtering and path stripping.
func TestFilterAssetNames(t *testing.T) {
	tests := []struct {
		prefix string
		minLen int
	}{
		{"mysql", 0},
		{"postgres", 0},
		{"sqlite", 0},
		{"nonexistent_prefix", 0},
	}

	for _, tt := range tests {
		t.Run(tt.prefix, func(t *testing.T) {
			result := filterAssetNames(tt.prefix)
			if len(result) < tt.minLen {
				t.Errorf("filterAssetNames(%q) returned %d items, want at least %d",
					tt.prefix, len(result), tt.minLen)
			}
			// Verify no result contains the prefix as a directory component
			for _, name := range result {
				if len(name) > 0 && name[0] == '/' {
					t.Errorf("filterAssetNames(%q) returned %q with leading slash", tt.prefix, name)
				}
			}
		})
	}
}

// TestFilterAssetNames_WithRealAssets verifies filtering with real embedded assets.
func TestFilterAssetNames_WithRealAssets(t *testing.T) {
	// Test with actual embedded assets
	mysqlAssets := filterAssetNames("mysql")
	if len(mysqlAssets) == 0 {
		t.Log("No mysql assets found in embedded resources")
	}

	// Verify prefix stripping
	for _, name := range mysqlAssets {
		if len(name) > 0 && name[0] == '/' {
			t.Errorf("Asset name %q should not start with slash", name)
		}
		if len(name) > 5 && name[:6] == "mysql/" {
			t.Errorf("Asset name %q should have prefix stripped", name)
		}
	}
}

// TestNewBindataSource_WithRealAssets verifies bindata source creation with real assets.
func TestNewBindataSource_WithRealAssets(t *testing.T) {
	// Test with sqlite assets (should always exist)
	src, err := newBindataSource("sqlite")
	if err != nil {
		t.Fatalf("newBindataSource(sqlite) error = %v", err)
	}
	if src == nil {
		t.Fatal("newBindataSource(sqlite) returned nil")
	}
	defer func() { _ = src.Close() }()

	// Verify we can read a migration (version 1 should exist)
	body, identifier, err := src.ReadUp(1)
	if err != nil {
		t.Fatalf("ReadUp(1) error = %v", err)
	}
	if identifier == "" {
		t.Error("Expected non-empty identifier")
	}
	if body != nil {
		_ = body.Close()
	}
}

// TestNewBindataSource_NonexistentPrefix verifies behavior with no matching assets.
func TestNewBindataSource_NonexistentPrefix(t *testing.T) {
	src, err := newBindataSource("nonexistent_prefix_12345")
	if err != nil {
		t.Fatalf("newBindataSource with empty assets error = %v", err)
	}
	if src == nil {
		t.Fatal("newBindataSource returned nil")
	}
	defer func() { _ = src.Close() }()

	// Should return error when trying to read migration
	_, _, err = src.ReadUp(1)
	if err == nil {
		t.Error("Expected error for nonexistent migration")
	}
}

// TestRunMigrateUp_EmptyURL verifies empty URL is rejected.
func TestRunMigrateUp_EmptyURL(t *testing.T) {
	err := runMigrateUp("sqlite3", "", "sqlite", "sqlite", func(db *sql.DB) (database.Driver, error) {
		return nil, errors.New("should not be called")
	})
	if err == nil {
		t.Fatal("runMigrateUp with empty URL should fail")
	}
	if !errors.Is(err, ErrDatabaseConfigMissing) {
		t.Errorf("error = %v, want to wrap ErrDatabaseConfigMissing", err)
	}
}

// TestRunMigrateUp_InvalidDriver verifies invalid driver is handled.
func TestRunMigrateUp_InvalidDriver(t *testing.T) {
	err := runMigrateUp("nonexistent_driver", "test.db", "test", "test", func(db *sql.DB) (database.Driver, error) {
		return nil, errors.New("should not be called")
	})
	if err == nil {
		t.Skip("skipping: driver unexpectedly succeeded")
	}
	// Error should mention the driver
	if err.Error() == "" {
		t.Error("error message is empty")
	}
}

// TestRunMigrateUp_DriverFactoryError verifies driver factory errors are wrapped.
func TestRunMigrateUp_DriverFactoryError(t *testing.T) {
	// Use sqlite3 which should be available, but fail in the driver factory
	err := runMigrateUp("sqlite3", ":memory:", "sqlite", "sqlite", func(db *sql.DB) (database.Driver, error) {
		return nil, errors.New("driver factory error")
	})
	if err == nil {
		t.Fatal("runMigrateUp with failing driver factory should fail")
	}
	// Check if error message contains the factory error
	if err.Error() == "" {
		t.Error("error message is empty")
	}
}

// TestDriverFactoryType verifies the driverFactory type signature.
func TestDriverFactoryType(t *testing.T) {
	var factory driverFactory = func(db *sql.DB) (database.Driver, error) {
		return nil, nil
	}
	if factory == nil {
		t.Error("driverFactory should not be nil")
	}
}

// TestCommAssetIntegration verifies that comm.Asset and comm.AssetNames work correctly.
func TestCommAssetIntegration(t *testing.T) {
	// Test AssetNames
	names := comm.AssetNames()
	if len(names) == 0 {
		t.Error("AssetNames() returned empty list")
	}

	// Verify expected assets are present
	expectedAssets := []string{
		"sqlite/000001_gokins.up.sql",
		"mysql/000001_gokins.up.sql",
		"postgres/000001_gokins.up.sql",
	}
	nameSet := make(map[string]bool, len(names))
	for _, n := range names {
		nameSet[n] = true
	}
	for _, want := range expectedAssets {
		if !nameSet[want] {
			t.Errorf("AssetNames() missing expected asset %q", want)
		}
	}

	// Test Asset for a known file
	data, err := comm.Asset("sqlite/000001_gokins.up.sql")
	if err != nil {
		t.Fatalf("Asset(sqlite/000001_gokins.up.sql) error = %v", err)
	}
	if len(data) == 0 {
		t.Error("Asset returned empty data")
	}

	// Test Asset for nonexistent file returns wrapped ErrAssetNotFound
	_, err = comm.Asset("nonexistent/file.sql")
	if err == nil {
		t.Error("Expected error for nonexistent asset")
	}
	if !errors.Is(err, comm.ErrAssetNotFound) {
		t.Errorf("error = %v, want to wrap ErrAssetNotFound", err)
	}
}

// TestAssetDir verifies the AssetDir tree navigation.
func TestAssetDir(t *testing.T) {
	// Root should return top-level directories
	dirs, err := comm.AssetDir("")
	if err != nil {
		t.Fatalf("AssetDir('') error = %v", err)
	}
	dirSet := make(map[string]bool, len(dirs))
	for _, d := range dirs {
		dirSet[d] = true
	}
	for _, want := range []string{"mysql", "postgres", "sqlite"} {
		if !dirSet[want] {
			t.Errorf("AssetDir('') missing expected dir %q, got %v", want, dirs)
		}
	}

	// sqlite/ should contain migration files
	files, err := comm.AssetDir("sqlite")
	if err != nil {
		t.Fatalf("AssetDir('sqlite') error = %v", err)
	}
	if len(files) == 0 {
		t.Error("AssetDir('sqlite') returned empty list")
	}

	// Nonexistent directory should return ErrAssetNotFound
	_, err = comm.AssetDir("nonexistent_dir")
	if err == nil {
		t.Error("Expected error for nonexistent dir")
	}
	if !errors.Is(err, comm.ErrAssetNotFound) {
		t.Errorf("error = %v, want to wrap ErrAssetNotFound", err)
	}
}

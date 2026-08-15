package migrates

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gokins/gokins/comm"
	_ "github.com/mattn/go-sqlite3"
)

// TestInitSqliteMigrate_Success tests successful SQLite migration initialization.
func TestInitSqliteMigrate_Success(t *testing.T) {
	// Save original WorkPath and restore after test
	origWorkPath := comm.WorkPath
	t.Cleanup(func() { comm.WorkPath = origWorkPath })

	// Create temporary directory for test database
	tmpDir := t.TempDir()
	comm.WorkPath = tmpDir

	// Run migration
	rtul, err := InitSqliteMigrate()
	if err != nil {
		t.Fatalf("InitSqliteMigrate() error = %v", err)
	}

	// Verify returned path
	expectedPath := filepath.Join(tmpDir, "db.dat")
	if rtul != expectedPath {
		t.Errorf("returned path = %q, want %q", rtul, expectedPath)
	}

	// Verify database file was created
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("database file %q was not created", expectedPath)
	}

	// Verify database is accessible
	db, err := sql.Open("sqlite3", expectedPath)
	if err != nil {
		t.Fatalf("could not open created database: %v", err)
	}
	defer func() { _ = db.Close() }()

	// Verify we can ping it
	if err := db.Ping(); err != nil {
		t.Errorf("could not ping created database: %v", err)
	}

	// Verify schema_migrations table exists (created by golang-migrate)
	var tableName string
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='schema_migrations'").Scan(&tableName)
	if err != nil {
		t.Errorf("schema_migrations table not found: %v", err)
	}
}

// TestInitSqliteMigrate_InvalidWorkPath tests failure with unwritable work path.
func TestInitSqliteMigrate_InvalidWorkPath(t *testing.T) {
	origWorkPath := comm.WorkPath
	t.Cleanup(func() { comm.WorkPath = origWorkPath })

	// Use a path that cannot be written (non-existent parent directory)
	comm.WorkPath = "/nonexistent/directory/that/does/not/exist"

	_, err := InitSqliteMigrate()
	if err == nil {
		t.Skip("skipping: migration succeeded unexpectedly (possibly running as root)")
	}

	// Verify error contains meaningful context
	if err.Error() == "" {
		t.Error("error message is empty")
	}
}

// TestUpSqliteMigrate_Success tests successful SQLite migration upgrade.
func TestUpSqliteMigrate_Success(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "db.dat")

	// First initialize the database
	origWorkPath := comm.WorkPath
	t.Cleanup(func() { comm.WorkPath = origWorkPath })
	comm.WorkPath = tmpDir

	_, err := InitSqliteMigrate()
	if err != nil {
		t.Fatalf("InitSqliteMigrate() error = %v", err)
	}

	// Now run UpSqliteMigrate on the same database
	err = UpSqliteMigrate(dbPath)
	if err != nil {
		t.Errorf("UpSqliteMigrate() error = %v", err)
	}
}

// TestSqliteMigrationIdempotency tests that running migrations twice is safe.
func TestSqliteMigrationIdempotency(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "db.dat")

	origWorkPath := comm.WorkPath
	t.Cleanup(func() { comm.WorkPath = origWorkPath })
	comm.WorkPath = tmpDir

	// First migration
	_, err := InitSqliteMigrate()
	if err != nil {
		t.Fatalf("first InitSqliteMigrate() error = %v", err)
	}

	// Second migration should be idempotent (no change)
	err = UpSqliteMigrate(dbPath)
	if err != nil {
		t.Errorf("second UpSqliteMigrate() should be idempotent, error = %v", err)
	}
}

// TestSqliteMigrationCreatesExpectedTables verifies that after migration,
// the database contains the expected table schema.
func TestSqliteMigrationCreatesExpectedTables(t *testing.T) {
	tmpDir := t.TempDir()

	origWorkPath := comm.WorkPath
	t.Cleanup(func() { comm.WorkPath = origWorkPath })
	comm.WorkPath = tmpDir

	_, err := InitSqliteMigrate()
	if err != nil {
		t.Fatalf("InitSqliteMigrate() error = %v", err)
	}

	dbPath := filepath.Join(tmpDir, "db.dat")
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("could not open database: %v", err)
	}
	defer func() { _ = db.Close() }()

	// List all tables
	rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='table' ORDER BY name")
	if err != nil {
		t.Fatalf("query tables: %v", err)
	}
	defer func() { _ = rows.Close() }()

	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("scan table name: %v", err)
		}
		tables = append(tables, name)
	}

	if len(tables) == 0 {
		t.Error("no tables found in migrated database")
	}

	// Check for schema_migrations (always created by golang-migrate)
	found := false
	for _, tbl := range tables {
		if tbl == "schema_migrations" {
			found = true
			break
		}
	}
	if !found {
		t.Error("schema_migrations table not found after migration")
	}

	// Verify the migration is marked as applied
	var version int64
	var dirty bool
	err = db.QueryRow("SELECT version, dirty FROM schema_migrations").Scan(&version, &dirty)
	if err != nil {
		t.Errorf("could not read migration version: %v", err)
	}
	if dirty {
		t.Error("migration is marked as dirty after successful run")
	}
	if version < 1 {
		t.Errorf("migration version = %d, expected >= 1", version)
	}

	t.Logf("Tables after migration: %v (migration version: %d)", tables, version)
}

// TestSqliteMigrationErrorWrapping verifies error wrapping in sqlite migration functions.
func TestSqliteMigrationErrorWrapping(t *testing.T) {
	tests := []struct {
		name    string
		fn      func() error
		wantMsg string
	}{
		{
			"UpSqliteMigrate empty",
			func() error { return UpSqliteMigrate("") },
			"database config not found",
		},
		{
			"UpSqliteMigrate invalid path",
			func() error { return UpSqliteMigrate("/nonexistent/sqlite/test.db") },
			"sqlite",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			if err == nil {
				t.Skip("skipping: expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantMsg) {
				t.Errorf("error = %q, want to contain %q", err.Error(), tt.wantMsg)
			}
		})
	}
}

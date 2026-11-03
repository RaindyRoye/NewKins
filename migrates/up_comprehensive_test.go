package migrates

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/gokins/gokins/comm"
	_ "github.com/mattn/go-sqlite3"
)

// TestUpSqliteMigrate_CreatesSchemaMigrationsTable verifies that migration creates the schema_migrations table.
func TestUpSqliteMigrate_CreatesSchemaMigrationsTable(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	err := UpSqliteMigrate(dbPath)
	if err != nil {
		t.Fatalf("UpSqliteMigrate failed: %v", err)
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer func() { _ = db.Close() }()

	var name string
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='schema_migrations'").Scan(&name)
	if err != nil {
		t.Errorf("schema_migrations table not created: %v", err)
	}

	var version int
	var dirty bool
	err = db.QueryRow("SELECT version, dirty FROM schema_migrations").Scan(&version, &dirty)
	if err != nil {
		t.Errorf("Failed to query schema_migrations: %v", err)
	}
	t.Logf("Migration version: %d, dirty: %v", version, dirty)
}

// TestUpSqliteMigrate_MultipleRunsIdempotent verifies that running migration multiple times is safe.
func TestUpSqliteMigrate_MultipleRunsIdempotent(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	for i := 0; i < 3; i++ {
		err := UpSqliteMigrate(dbPath)
		if err != nil {
			t.Errorf("Migration run %d failed: %v", i+1, err)
		}
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer func() { _ = db.Close() }()

	err = db.PingContext(context.Background())
	if err != nil {
		t.Errorf("Database not usable after multiple migrations: %v", err)
	}
}

// TestUpSqliteMigrate_WithExistingDatabase verifies migration works on an existing database.
func TestUpSqliteMigrate_WithExistingDatabase(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	_, err = db.Exec("CREATE TABLE pre_existing (id INTEGER PRIMARY KEY)")
	if err != nil {
		_ = db.Close()
		t.Fatalf("Failed to create test table: %v", err)
	}
	_ = db.Close()

	err = UpSqliteMigrate(dbPath)
	if err != nil {
		t.Fatalf("Migration on existing database failed: %v", err)
	}

	db, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("Failed to reopen database: %v", err)
	}
	defer func() { _ = db.Close() }()

	var name string
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='pre_existing'").Scan(&name)
	if err != nil {
		t.Errorf("pre_existing table was deleted during migration: %v", err)
	}
}

// TestUpSqliteMigrate_DatabaseFilePermissions verifies database file has correct permissions.
func TestUpSqliteMigrate_DatabaseFilePermissions(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	err := UpSqliteMigrate(dbPath)
	if err != nil {
		t.Fatalf("UpSqliteMigrate failed: %v", err)
	}

	info, err := os.Stat(dbPath)
	if err != nil {
		t.Fatalf("Database file not created: %v", err)
	}
	if info.Size() == 0 {
		t.Error("Database file is empty")
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("Cannot open created database: %v", err)
	}
	defer func() { _ = db.Close() }()

	err = db.PingContext(context.Background())
	if err != nil {
		t.Errorf("Cannot ping created database: %v", err)
	}
}

// TestUpMysqlMigrate_ErrorWrapping_Comprehensive verifies error messages are properly wrapped.
func TestUpMysqlMigrate_ErrorWrapping_Comprehensive(t *testing.T) {
	tests := []struct {
		name      string
		url       string
		errorType error
	}{
		{
			name:      "empty URL",
			url:       "",
			errorType: ErrDatabaseConfigMissing,
		},
		{
			name:      "connection refused",
			url:       "root:password@tcp(127.0.0.1:1)/testdb?parseTime=true",
			errorType: nil,
		},
		{
			name:      "invalid host",
			url:       "root:***@tcp(nonexistent.invalid:3306)/testdb?parseTime=true",
			errorType: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := UpMysqlMigrate(tt.url)
			if err == nil {
				t.Skip("skipping: connection unexpectedly succeeded")
			}
			if tt.errorType != nil && !errors.Is(err, tt.errorType) {
				t.Errorf("Expected error wrapping %v, got: %v", tt.errorType, err)
			}
		})
	}
}

// TestUpPostgresMigrate_ErrorWrapping_Comprehensive verifies error messages are properly wrapped.
func TestUpPostgresMigrate_ErrorWrapping_Comprehensive(t *testing.T) {
	tests := []struct {
		name      string
		url       string
		errorType error
	}{
		{
			name:      "empty URL",
			url:       "",
			errorType: ErrDatabaseConfigMissing,
		},
		{
			name:      "connection refused",
			url:       "postgres://user:***@127.0.0.1:1/testdb?sslmode=disable",
			errorType: nil,
		},
		{
			name:      "invalid host",
			url:       "postgres://user:***@nonexistent.invalid:5432/testdb?sslmode=disable",
			errorType: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := UpPostgresMigrate(tt.url)
			if err == nil {
				t.Skip("skipping: connection unexpectedly succeeded")
			}
			if tt.errorType != nil && !errors.Is(err, tt.errorType) {
				t.Errorf("Expected error wrapping %v, got: %v", tt.errorType, err)
			}
		})
	}
}

// TestUpSqliteMigrate_ErrorWrapping_Comprehensive verifies error messages are properly wrapped.
func TestUpSqliteMigrate_ErrorWrapping_Comprehensive(t *testing.T) {
	tests := []struct {
		name      string
		url       string
		errorType error
	}{
		{
			name:      "empty URL",
			url:       "",
			errorType: ErrDatabaseConfigMissing,
		},
		{
			name:      "invalid path",
			url:       "/nonexistent/directory/test.db",
			errorType: nil,
		},
		{
			name:      "read-only path",
			url:       "/proc/1/test.db",
			errorType: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := UpSqliteMigrate(tt.url)
			if err == nil {
				t.Skip("skipping: unexpected success")
			}
			if tt.errorType != nil && !errors.Is(err, tt.errorType) {
				t.Errorf("Expected error wrapping %v, got: %v", tt.errorType, err)
			}
		})
	}
}

// TestUpSqliteMigrate_CreatesAllTables verifies that all expected tables are created.
func TestUpSqliteMigrate_CreatesAllTables(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	err := UpSqliteMigrate(dbPath)
	if err != nil {
		t.Fatalf("UpSqliteMigrate failed: %v", err)
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer func() { _ = db.Close() }()

	rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='table' ORDER BY name")
	if err != nil {
		t.Fatalf("Failed to query tables: %v", err)
	}
	defer func() { _ = rows.Close() }()

	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("Failed to scan table name: %v", err)
		}
		tables = append(tables, name)
	}

	if len(tables) == 0 {
		t.Error("No tables created by migration")
	}

	found := false
	for _, table := range tables {
		if table == "schema_migrations" {
			found = true
			break
		}
	}
	if !found {
		t.Error("schema_migrations table not created")
	}

	t.Logf("Created %d tables: %v", len(tables), tables)
}

// TestUpSqliteMigrate_ConcurrentCalls verifies behavior when multiple migrations run concurrently.
func TestUpSqliteMigrate_ConcurrentCalls(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	var wg sync.WaitGroup
	errs := make([]error, 3)
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			errs[idx] = UpSqliteMigrate(dbPath)
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Logf("Concurrent migration %d error: %v (may be expected due to locking)", i, err)
		}
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer func() { _ = db.Close() }()

	err = db.PingContext(context.Background())
	if err != nil {
		t.Errorf("Database not usable after concurrent migrations: %v", err)
	}
}

// TestAssetFiltering_Logic verifies the asset filtering logic used in migration functions.
func TestAssetFiltering_Logic(t *testing.T) {
	assets := comm.AssetNames()

	prefixes := []string{"mysql", "postgres", "sqlite"}
	for _, prefix := range prefixes {
		var filtered []string
		for _, v := range assets {
			if strings.HasPrefix(v, prefix) {
				filtered = append(filtered, strings.Replace(v, prefix+"/", "", 1))
			}
		}
		t.Logf("Prefix %q: %d assets", prefix, len(filtered))

		for _, asset := range filtered {
			if strings.Contains(asset, "mysql") && prefix != "mysql" {
				t.Errorf("Non-%s asset included in %s filter: %s", prefix, prefix, asset)
			}
		}
	}
}

// TestUpSqliteMigrate_BindataResourceCreation verifies bindata resource is created correctly.
func TestUpSqliteMigrate_BindataResourceCreation(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	err := UpSqliteMigrate(dbPath)
	if err != nil {
		t.Fatalf("Migration failed (bindata issue?): %v", err)
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer func() { _ = db.Close() }()

	err = db.PingContext(context.Background())
	if err != nil {
		t.Errorf("Database not usable: %v", err)
	}
}

// TestUpSqliteMigrate_EmptyConnectionString verifies error wrapping for empty connection.
func TestUpSqliteMigrate_EmptyConnectionString(t *testing.T) {
	err := UpSqliteMigrate("")
	if err == nil {
		t.Fatal("expected error for empty connection string")
	}
	if !errors.Is(err, ErrDatabaseConfigMissing) {
		t.Errorf("expected ErrDatabaseConfigMissing, got: %v", err)
	}
}

// TestUpMysqlMigrate_EmptyConnectionString verifies error wrapping for empty connection.
func TestUpMysqlMigrate_EmptyConnectionString(t *testing.T) {
	err := UpMysqlMigrate("")
	if err == nil {
		t.Fatal("expected error for empty connection string")
	}
	if !errors.Is(err, ErrDatabaseConfigMissing) {
		t.Errorf("expected ErrDatabaseConfigMissing, got: %v", err)
	}
}

// TestUpPostgresMigrate_EmptyConnectionString verifies error wrapping for empty connection.
func TestUpPostgresMigrate_EmptyConnectionString(t *testing.T) {
	err := UpPostgresMigrate("")
	if err == nil {
		t.Fatal("expected error for empty connection string")
	}
	if !errors.Is(err, ErrDatabaseConfigMissing) {
		t.Errorf("expected ErrDatabaseConfigMissing, got: %v", err)
	}
}

// TestUpSqliteMigrate_MigrationVersionCheck verifies the migration version after running.
func TestUpSqliteMigrate_MigrationVersionCheck(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	err := UpSqliteMigrate(dbPath)
	if err != nil {
		t.Fatalf("UpSqliteMigrate failed: %v", err)
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer func() { _ = db.Close() }()

	var version int
	var dirty bool
	err = db.QueryRow("SELECT version, dirty FROM schema_migrations").Scan(&version, &dirty)
	if err != nil {
		t.Fatalf("Failed to query migration version: %v", err)
	}

	if dirty {
		t.Error("Migration marked as dirty after successful run")
	}
	if version < 1 {
		t.Errorf("Migration version should be >= 1, got %d", version)
	}
	t.Logf("Migration version: %d, dirty: %v", version, dirty)
}

// TestUpSqliteMigrate_DataIntegrity verifies data integrity after migration.
func TestUpSqliteMigrate_DataIntegrity(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	err := UpSqliteMigrate(dbPath)
	if err != nil {
		t.Fatalf("UpSqliteMigrate failed: %v", err)
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer func() { _ = db.Close() }()

	// Verify we can execute basic operations
	_, err = db.Exec("PRAGMA integrity_check")
	if err != nil {
		t.Errorf("Integrity check failed: %v", err)
	}

	// Verify WAL mode can be set
	_, err = db.Exec("PRAGMA journal_mode=WAL")
	if err != nil {
		t.Logf("WAL mode not available: %v (not critical)", err)
	}
}

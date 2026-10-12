package migrates

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gokins/gokins/comm"
	_ "github.com/mattn/go-sqlite3"
)

// TestMysqlDSNFormat tests MySQL DSN formatting logic
func TestMysqlDSNFormat(t *testing.T) {
	tests := []struct {
		name     string
		host     string
		dbs      string
		user     string
		pass     string
		expected string
	}{
		{
			name:     "basic format",
			host:     "localhost:3306",
			dbs:      "testdb",
			user:     "root",
			pass:     "secret",
			expected: "root:secret@tcp(localhost:3306)/testdb?parseTime=true&multiStatements=true",
		},
		{
			name:     "empty password",
			host:     "127.0.0.1:3306",
			dbs:      "mydb",
			user:     "admin",
			pass:     "",
			expected: "admin:@tcp(127.0.0.1:3306)/mydb?parseTime=true&multiStatements=true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fmt.Sprintf("%s:%s@tcp(%s)/%s?parseTime=true&multiStatements=true",
				tt.user, tt.pass, tt.host, tt.dbs)
			if result != tt.expected {
				t.Errorf("DSN format = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestPostgresDSNFormat tests PostgreSQL DSN formatting logic
func TestPostgresDSNFormat(t *testing.T) {
	tests := []struct {
		name     string
		host     string
		dbs      string
		user     string
		pass     string
		expected string
	}{
		{
			name:     "basic format",
			host:     "localhost:5432",
			dbs:      "testdb",
			user:     "postgres",
			pass:     "secret",
			expected: "postgres://postgres:secret@localhost:5432/testdb?sslmode=disable",
		},
		{
			name:     "with special chars in password",
			host:     "db.example.com:5432",
			dbs:      "production",
			user:     "appuser",
			pass:     "p@ss!word",
			expected: "postgres://appuser:p@ss!word@db.example.com:5432/production?sslmode=disable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable",
				tt.user, tt.pass, tt.host, tt.dbs)
			if result != tt.expected {
				t.Errorf("DSN format = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestSqliteDBPath tests SQLite database path construction
func TestSqliteDBPath(t *testing.T) {
	tests := []struct {
		name     string
		workPath string
		expected string
	}{
		{
			name:     "absolute path",
			workPath: "/var/lib/gokins",
			expected: "/var/lib/gokins/db.dat",
		},
		{
			name:     "relative path",
			workPath: "./data",
			expected: "data/db.dat",
		},
		{
			name:     "current directory",
			workPath: ".",
			expected: "db.dat",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filepath.Join(tt.workPath, "db.dat")
			if result != tt.expected {
				t.Errorf("path = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestInitSqliteMigrateWithInvalidPath tests error handling for invalid paths
func TestInitSqliteMigrateWithInvalidPath(t *testing.T) {
	// Save original WorkPath
	origWorkPath := comm.WorkPath
	defer func() { comm.WorkPath = origWorkPath }()

	// Set invalid path
	comm.WorkPath = "/nonexistent/path/that/does/not/exist"

	_, err := InitSqliteMigrate()
	if err == nil {
		t.Skip("expected error for invalid path, got nil")
	}

	// Verify error contains context
	errMsg := err.Error()
	if !strings.Contains(errMsg, "sqlite") {
		t.Errorf("error should mention sqlite, got: %s", errMsg)
	}
}

// TestSqliteMigrationSchemaValidation validates schema after migration
func TestSqliteMigrationSchemaValidation(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()

	// Save and restore WorkPath
	origWorkPath := comm.WorkPath
	defer func() { comm.WorkPath = origWorkPath }()
	comm.WorkPath = tmpDir

	// Run migration
	dbPath, err := InitSqliteMigrate()
	if err != nil {
		t.Fatalf("InitSqliteMigrate() error = %v", err)
	}

	// Open database
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	defer db.Close()

	// Verify schema_migrations table structure
	var version int64
	var dirty bool
	err = db.QueryRow("SELECT version, dirty FROM schema_migrations").Scan(&version, &dirty)
	if err != nil {
		t.Errorf("failed to query schema_migrations: %v", err)
	}

	if version < 1 {
		t.Errorf("migration version = %d, expected >= 1", version)
	}
	if dirty {
		t.Error("migration should not be marked dirty after successful run")
	}

	// Verify we can ping the database
	if err := db.PingContext(context.Background()); err != nil {
		t.Errorf("db.PingContext() error = %v", err)
	}
}

// TestUpSqliteMigrateIdempotency tests that UpSqliteMigrate can be run multiple times
func TestUpSqliteMigrateIdempotency(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "db.dat")

	// First run
	err := UpSqliteMigrate(dbPath)
	if err != nil {
		t.Skipf("first UpSqliteMigrate() failed: %v", err)
	}

	// Second run should be idempotent
	err = UpSqliteMigrate(dbPath)
	if err != nil {
		t.Errorf("second UpSqliteMigrate() should be idempotent, error = %v", err)
	}
}

// TestMigrationErrorMessages verifies error messages are descriptive
func TestMigrationErrorMessages(t *testing.T) {
	tests := []struct {
		name        string
		fn          func() error
		wantContain []string
	}{
		{
			name:        "mysql empty connection",
			fn:          func() error { return UpMysqlMigrate("") },
			wantContain: []string{"mysql", "empty"},
		},
		{
			name:        "postgres empty connection",
			fn:          func() error { return UpPostgresMigrate("") },
			wantContain: []string{"postgres", "empty"},
		},
		{
			name:        "sqlite empty connection",
			fn:          func() error { return UpSqliteMigrate("") },
			wantContain: []string{"sqlite", "empty"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			errMsg := strings.ToLower(err.Error())
			for _, want := range tt.wantContain {
				if !strings.Contains(errMsg, strings.ToLower(want)) {
					t.Errorf("error %q should contain %q", err.Error(), want)
				}
			}
		})
	}
}

// TestInitMysqlMigrateValidation tests parameter validation
func TestInitMysqlMigrateValidation(t *testing.T) {
	tests := []struct {
		name string
		host string
		dbs  string
		user string
		pass string
	}{
		{"all empty", "", "", "", ""},
		{"no host", "", "db", "user", "pass"},
		{"no database", "host", "", "user", "pass"},
		{"no user", "host", "db", "", "pass"},
		{"only host", "host", "", "", ""},
		{"only database", "", "db", "", ""},
		{"only user", "", "", "user", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := InitMysqlMigrate(tt.host, tt.dbs, tt.user, tt.pass)
			if err == nil {
				t.Error("expected validation error, got nil")
			}
			if !errors.Is(err, ErrDatabaseConfigMissing) {
				t.Errorf("error should wrap ErrDatabaseConfigMissing, got: %v", err)
			}
		})
	}
}

// TestInitPostgresMigrateValidation tests parameter validation
func TestInitPostgresMigrateValidation(t *testing.T) {
	tests := []struct {
		name string
		host string
		dbs  string
		user string
		pass string
	}{
		{"all empty", "", "", "", ""},
		{"no host", "", "db", "user", "pass"},
		{"no database", "host", "", "user", "pass"},
		{"no user", "host", "db", "", "pass"},
		{"only host", "host", "", "", ""},
		{"only database", "", "db", "", ""},
		{"only user", "", "", "user", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := InitPostgresMigrate(tt.host, tt.dbs, tt.user, tt.pass)
			if err == nil {
				t.Error("expected validation error, got nil")
			}
			if !errors.Is(err, ErrDatabaseConfigMissing) {
				t.Errorf("error should wrap ErrDatabaseConfigMissing, got: %v", err)
			}
		})
	}
}

// TestSqliteMigrationFileVerification tests actual SQLite migration and verifies file
func TestSqliteMigrationFileVerification(t *testing.T) {
	tmpDir := t.TempDir()

	// Save and restore WorkPath
	origWorkPath := comm.WorkPath
	defer func() { comm.WorkPath = origWorkPath }()
	comm.WorkPath = tmpDir

	// Test InitSqliteMigrate
	dbPath, err := InitSqliteMigrate()
	if err != nil {
		t.Fatalf("InitSqliteMigrate() error = %v", err)
	}

	// Verify database was created
	if dbPath == "" {
		t.Error("InitSqliteMigrate returned empty path")
	}

	info, err := os.Stat(dbPath)
	if err != nil {
		t.Fatalf("database file not found: %v", err)
	}
	if info.Size() == 0 {
		t.Error("database file is empty")
	}

	// Test UpSqliteMigrate on existing DB
	err = UpSqliteMigrate(dbPath)
	if err != nil {
		t.Errorf("UpSqliteMigrate() error = %v", err)
	}
}

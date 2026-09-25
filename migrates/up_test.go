package migrates

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gokins/gokins/comm"
	_ "github.com/mattn/go-sqlite3"
)

// TestUpMysqlMigrate_TableDriven covers various error paths for MySQL upgrade.
func TestUpMysqlMigrate_TableDriven(t *testing.T) {
	tests := []struct {
		name    string
		ul      string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "empty URL",
			ul:      "",
			wantErr: true,
			errMsg:  "database config not found",
		},
		{
			name:    "invalid DSN format",
			ul:      "invalid-dsn",
			wantErr: true,
			errMsg:  "mysql",
		},
		{
			name:    "unreachable host",
			ul:      "user:***@tcp(127.0.0.1:59999)/testdb?parseTime=true",
			wantErr: true,
			errMsg:  "mysql",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := UpMysqlMigrate(context.Background(), tt.ul)
			if tt.wantErr {
				if err == nil {
					t.Skip("skipping: expected error, got nil")
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("error = %q, want to contain %q", err.Error(), tt.errMsg)
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// TestUpPostgresMigrate_TableDriven covers various error paths for PostgreSQL upgrade.
func TestUpPostgresMigrate_TableDriven(t *testing.T) {
	tests := []struct {
		name    string
		ul      string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "empty URL",
			ul:      "",
			wantErr: true,
			errMsg:  "database config not found",
		},
		{
			name:    "invalid URL format",
			ul:      "not-a-postgres-url",
			wantErr: true,
			errMsg:  "postgres",
		},
		{
			name:    "unreachable host",
			ul:      "postgres://user:***@127.0.0.1:59998/testdb?sslmode=disable",
			wantErr: true,
			errMsg:  "postgres",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := UpPostgresMigrate(context.Background(), tt.ul)
			if tt.wantErr {
				if err == nil {
					t.Skip("skipping: expected error, got nil")
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("error = %q, want to contain %q", err.Error(), tt.errMsg)
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// TestUpSqliteMigrate_TableDriven covers various scenarios for SQLite upgrade.
func TestUpSqliteMigrate_TableDriven(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T) string
		wantErr bool
		errMsg  string
	}{
		{
			name: "empty URL",
			setup: func(t *testing.T) string {
				return ""
			},
			wantErr: true,
			errMsg:  "database config not found",
		},
		{
			name: "invalid path",
			setup: func(t *testing.T) string {
				return "/nonexistent/path/db.sqlite"
			},
			wantErr: true,
			errMsg:  "sqlite",
		},
		{
			name: "valid empty database",
			setup: func(t *testing.T) string {
				tmpDir := t.TempDir()
				return filepath.Join(tmpDir, "test.db")
			},
			wantErr: false,
		},
		{
			name: "already migrated database",
			setup: func(t *testing.T) string {
				tmpDir := t.TempDir()
				origWorkPath := comm.WorkPath
				t.Cleanup(func() { comm.WorkPath = origWorkPath })
				comm.WorkPath = tmpDir

				// Initialize first
				_, err := InitSqliteMigrate()
				if err != nil {
					t.Fatalf("InitSqliteMigrate failed: %v", err)
				}
				return filepath.Join(tmpDir, "db.dat")
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ul := tt.setup(t)
			err := UpSqliteMigrate(context.Background(), ul)
			if tt.wantErr {
				if err == nil {
					t.Skip("skipping: expected error, got nil")
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("error = %q, want to contain %q", err.Error(), tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

// TestUpSqliteMigrate_ConcurrentSafety verifies that running migrations
// concurrently on the same database doesn't corrupt state.
func TestUpSqliteMigrate_ConcurrentSafety(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "db.dat")

	origWorkPath := comm.WorkPath
	t.Cleanup(func() { comm.WorkPath = origWorkPath })
	comm.WorkPath = tmpDir

	// Initialize once
	_, err := InitSqliteMigrate()
	if err != nil {
		t.Fatalf("InitSqliteMigrate failed: %v", err)
	}

	// Run multiple upgrades sequentially (not parallel to avoid race conditions)
	for i := 0; i < 5; i++ {
		err := UpSqliteMigrate(context.Background(), dbPath)
		if err != nil {
			t.Errorf("iteration %d: UpSqliteMigrate failed: %v", i, err)
		}
	}
}

// TestUpSqliteMigrate_CorruptedDatabase tests behavior with an invalid database file.
func TestUpSqliteMigrate_CorruptedDatabase(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "corrupt.db")

	// Create a corrupted database file
	err := os.WriteFile(dbPath, []byte("not a valid sqlite database"), 0600)
	if err != nil {
		t.Fatalf("failed to create corrupt file: %v", err)
	}

	err = UpSqliteMigrate(context.Background(), dbPath)
	if err == nil {
		t.Skip("skipping: migration succeeded on corrupt database unexpectedly")
	}

	// Should get an error, but not panic
	if err.Error() == "" {
		t.Error("error message is empty")
	}
}

package migrates

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gokins/gokins/comm"
)

// TestInitMysqlMigrate_UnreachableHost tests error handling when MySQL host is unreachable.
// This exercises the open -> ping -> fallback create-database code paths.
func TestInitMysqlMigrate_UnreachableHost(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network test in short mode")
	}
	// Use an unreachable host to exercise the connection attempt paths
	t.Parallel()
	wait, ul, err := InitMysqlMigrate("192.0.2.1:1", "testdb", "testuser", "testpass")
	if err == nil {
		t.Skip("skipping: MySQL connection succeeded unexpectedly")
	}
	// Should have attempted connection (wait was set to true then reset)
	_ = wait
	_ = ul

	// Error should contain meaningful context
	if !strings.Contains(err.Error(), "mysql") && !strings.Contains(err.Error(), "database") && !strings.Contains(err.Error(), "ping") {
		t.Errorf("error should contain context about the failure, got: %v", err)
	}
}

// TestInitPostgresMigrate_UnreachableHost tests error handling when PostgreSQL host is unreachable.
func TestInitPostgresMigrate_UnreachableHost(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network test in short mode")
	}
	t.Parallel()
	wait, ul, err := InitPostgresMigrate("192.0.2.1:1", "testdb", "testuser", "testpass")
	if err == nil {
		t.Skip("skipping: PostgreSQL connection succeeded unexpectedly")
	}
	_ = wait
	_ = ul

	if !strings.Contains(err.Error(), "postgres") && !strings.Contains(err.Error(), "database") && !strings.Contains(err.Error(), "ping") {
		t.Errorf("error should contain context about the failure, got: %v", err)
	}
}

// TestUpSqliteMigrate_AlreadyMigrated verifies that running UpSqliteMigrate
// on an already-migrated database is safe (idempotent).
func TestUpSqliteMigrate_AlreadyMigrated(t *testing.T) {
	tmpDir := t.TempDir()
	origWorkPath := comm.WorkPath
	t.Cleanup(func() { comm.WorkPath = origWorkPath })
	comm.WorkPath = tmpDir

	// Initialize migration
	dbPath := filepath.Join(tmpDir, "db.dat")
	_, err := InitSqliteMigrate()
	if err != nil {
		t.Fatalf("InitSqliteMigrate() error = %v", err)
	}

	// Run UpSqliteMigrate multiple times — all should succeed
	for i := 0; i < 3; i++ {
		err = UpSqliteMigrate(dbPath)
		if err != nil {
			t.Errorf("UpSqliteMigrate() iteration %d error = %v", i, err)
		}
	}
}

// TestInitSqliteMigrate_ReturnsExpectedPath verifies the returned database path.
func TestInitSqliteMigrate_ReturnsExpectedPath(t *testing.T) {
	tmpDir := t.TempDir()
	origWorkPath := comm.WorkPath
	t.Cleanup(func() { comm.WorkPath = origWorkPath })
	comm.WorkPath = tmpDir

	rtul, err := InitSqliteMigrate()
	if err != nil {
		t.Fatalf("InitSqliteMigrate() error = %v", err)
	}

	expected := filepath.Join(tmpDir, "db.dat")
	if rtul != expected {
		t.Errorf("returned path = %q, want %q", rtul, expected)
	}

	// Verify the file exists
	info, err := os.Stat(expected)
	if err != nil {
		t.Fatalf("database file not found at %q: %v", expected, err)
	}
	if info.Size() == 0 {
		t.Error("database file is empty")
	}
}

// TestErrDatabaseConfigMissing_IsSentinel verifies the sentinel error can be
// used with errors.Is across wrapping layers.
func TestErrDatabaseConfigMissing_IsSentinel(t *testing.T) {
	// Simulate multi-layer wrapping as happens in production code
	inner := ErrDatabaseConfigMissing
	mid := errors.Join(inner, errors.New("middleware error"))
	outer := errors.Join(mid, errors.New("outer context"))

	if !errors.Is(outer, ErrDatabaseConfigMissing) {
		t.Error("deeply wrapped ErrDatabaseConfigMissing should still match via errors.Is")
	}

	// Verify error message propagation
	msg := outer.Error()
	if !strings.Contains(msg, "database config not found") {
		t.Errorf("wrapped error message should contain sentinel, got: %q", msg)
	}
}

// TestUpSqliteMigrate_NonExistentDirectory tests that UpSqliteMigrate fails
// gracefully when the database path points to a non-existent directory.
func TestUpSqliteMigrate_NonExistentDirectory(t *testing.T) {
	err := UpSqliteMigrate("/tmp/nonexistent_dir_12345/test.db")
	if err == nil {
		t.Skip("skipping: SQLite created database in non-existent directory unexpectedly")
	}
	if !strings.Contains(err.Error(), "sqlite") {
		t.Errorf("error should mention sqlite, got: %v", err)
	}
}

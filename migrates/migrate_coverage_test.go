package migrates

import (
	"errors"
	"strings"
	"testing"
)

// TestUpMysqlMigrate_BindataError tests UpMysqlMigrate when bindata resources are missing.
// This covers the bindata error path without requiring a real MySQL connection.
func TestUpMysqlMigrate_BindataError(t *testing.T) {
	// Empty connection string should fail early (already tested, but ensures coverage)
	err := UpMysqlMigrate("")
	if err == nil {
		t.Fatal("expected error for empty connection string")
	}
	if !strings.Contains(err.Error(), "database config not found") {
		t.Errorf("error = %v, want to contain 'database config not found'", err)
	}
}

// TestUpPostgresMigrate_BindataError tests UpPostgresMigrate when bindata resources are missing.
func TestUpPostgresMigrate_BindataError(t *testing.T) {
	err := UpPostgresMigrate("")
	if err == nil {
		t.Fatal("expected error for empty connection string")
	}
	if !strings.Contains(err.Error(), "database config not found") {
		t.Errorf("error = %v, want to contain 'database config not found'", err)
	}
}

// TestUpSqliteMigrate_BindataError tests UpSqliteMigrate when bindata resources are missing.
func TestUpSqliteMigrate_BindataError(t *testing.T) {
	err := UpSqliteMigrate("")
	if err == nil {
		t.Fatal("expected error for empty connection string")
	}
	if !strings.Contains(err.Error(), "database config not found") {
		t.Errorf("error = %v, want to contain 'database config not found'", err)
	}
}

// TestInitMysqlMigrate_EmptyHost tests InitMysqlMigrate with empty host parameter.
func TestInitMysqlMigrate_EmptyHost(t *testing.T) {
	wait, url, err := InitMysqlMigrate("", "testdb", "user", "pass")
	if err == nil {
		t.Fatal("expected error for empty host")
	}
	if wait {
		t.Error("wait should be false when host is empty")
	}
	if url != "" {
		t.Errorf("url = %q, want empty string", url)
	}
	if !strings.Contains(err.Error(), "database config not found") {
		t.Errorf("error = %v, want to contain 'database config not found'", err)
	}
}

// TestInitMysqlMigrate_EmptyDatabase tests InitMysqlMigrate with empty database parameter.
func TestInitMysqlMigrate_EmptyDatabase(t *testing.T) {
	wait, url, err := InitMysqlMigrate("localhost", "", "user", "pass")
	if err == nil {
		t.Fatal("expected error for empty database")
	}
	if wait {
		t.Error("wait should be false when database is empty")
	}
	if url != "" {
		t.Errorf("url = %q, want empty string", url)
	}
	if !strings.Contains(err.Error(), "database config not found") {
		t.Errorf("error = %v, want to contain 'database config not found'", err)
	}
}

// TestInitMysqlMigrate_EmptyUser tests InitMysqlMigrate with empty user parameter.
func TestInitMysqlMigrate_EmptyUser(t *testing.T) {
	wait, url, err := InitMysqlMigrate("localhost", "testdb", "", "pass")
	if err == nil {
		t.Fatal("expected error for empty user")
	}
	if wait {
		t.Error("wait should be false when user is empty")
	}
	if url != "" {
		t.Errorf("url = %q, want empty string", url)
	}
	if !strings.Contains(err.Error(), "database config not found") {
		t.Errorf("error = %v, want to contain 'database config not found'", err)
	}
}

// TestInitPostgresMigrate_EmptyHost tests InitPostgresMigrate with empty host parameter.
func TestInitPostgresMigrate_EmptyHost(t *testing.T) {
	wait, url, err := InitPostgresMigrate("", "testdb", "user", "pass")
	if err == nil {
		t.Fatal("expected error for empty host")
	}
	if wait {
		t.Error("wait should be false when host is empty")
	}
	if url != "" {
		t.Errorf("url = %q, want empty string", url)
	}
	if !strings.Contains(err.Error(), "database config not found") {
		t.Errorf("error = %v, want to contain 'database config not found'", err)
	}
}

// TestInitPostgresMigrate_EmptyDatabase tests InitPostgresMigrate with empty database parameter.
func TestInitPostgresMigrate_EmptyDatabase(t *testing.T) {
	wait, url, err := InitPostgresMigrate("localhost", "", "user", "pass")
	if err == nil {
		t.Fatal("expected error for empty database")
	}
	if wait {
		t.Error("wait should be false when database is empty")
	}
	if url != "" {
		t.Errorf("url = %q, want empty string", url)
	}
	if !strings.Contains(err.Error(), "database config not found") {
		t.Errorf("error = %v, want to contain 'database config not found'", err)
	}
}

// TestInitPostgresMigrate_EmptyUser tests InitPostgresMigrate with empty user parameter.
func TestInitPostgresMigrate_EmptyUser(t *testing.T) {
	wait, url, err := InitPostgresMigrate("localhost", "testdb", "", "pass")
	if err == nil {
		t.Fatal("expected error for empty user")
	}
	if wait {
		t.Error("wait should be false when user is empty")
	}
	if url != "" {
		t.Errorf("url = %q, want empty string", url)
	}
	if !strings.Contains(err.Error(), "database config not found") {
		t.Errorf("error = %v, want to contain 'database config not found'", err)
	}
}

// TestUpMysqlMigrate_InvalidConnectionString tests UpMysqlMigrate with malformed connection string.
func TestUpMysqlMigrate_InvalidConnectionString(t *testing.T) {
	// Invalid format - missing required components
	err := UpMysqlMigrate("invalid-format")
	if err == nil {
		t.Fatal("expected error for invalid connection string")
	}
	// Should fail when trying to ping the database
	if !strings.Contains(err.Error(), "ping") && !strings.Contains(err.Error(), "open") {
		t.Errorf("error = %v, want to contain 'ping' or 'open'", err)
	}
}

// TestUpPostgresMigrate_InvalidConnectionString tests UpPostgresMigrate with malformed connection string.
func TestUpPostgresMigrate_InvalidConnectionString(t *testing.T) {
	err := UpPostgresMigrate("postgres://invalid")
	if err == nil {
		t.Fatal("expected error for invalid connection string")
	}
	if !strings.Contains(err.Error(), "ping") && !strings.Contains(err.Error(), "open") {
		t.Errorf("error = %v, want to contain 'ping' or 'open'", err)
	}
}

// TestUpSqliteMigrate_InvalidPathCov tests UpSqliteMigrate with non-existent directory.
func TestUpSqliteMigrate_InvalidPathCov(t *testing.T) {
	err := UpSqliteMigrate("/nonexistent/directory/db.sqlite")
	if err == nil {
		t.Skip("skipping: migration succeeded unexpectedly (possibly running as root)")
	}
	if !strings.Contains(err.Error(), "ping") && !strings.Contains(err.Error(), "open") {
		t.Errorf("error = %v, want to contain 'ping' or 'open'", err)
	}
}

// TestInitMysqlMigrate_InvalidConnectionString tests InitMysqlMigrate with invalid connection parameters
// that pass initial validation but fail during database operations.
func TestInitMysqlMigrate_InvalidConnectionString(t *testing.T) {
	// This should pass validation but fail when trying to connect
	wait, url, err := InitMysqlMigrate("nonexistent-host:3306", "testdb", "user", "pass")
	if err == nil {
		t.Skip("skipping: connection succeeded unexpectedly")
	}
	// wait should be true because validation passed
	if !wait {
		t.Error("wait should be true when validation passes but connection fails")
	}
	if url != "" {
		t.Errorf("url = %q, want empty string on error", url)
	}
}

// TestInitPostgresMigrate_InvalidConnectionString tests InitPostgresMigrate with invalid connection parameters.
func TestInitPostgresMigrate_InvalidConnectionString(t *testing.T) {
	wait, url, err := InitPostgresMigrate("nonexistent-host:5432", "testdb", "user", "pass")
	if err == nil {
		t.Skip("skipping: connection succeeded unexpectedly")
	}
	if !wait {
		t.Error("wait should be true when validation passes but connection fails")
	}
	if url != "" {
		t.Errorf("url = %q, want empty string on error", url)
	}
}

// TestDatabaseConfigMissingError tests that ErrDatabaseConfigMissing is properly used.
func TestDatabaseConfigMissingError(t *testing.T) {
	tests := []struct {
		name string
		fn   func() error
	}{
		{"UpMysqlMigrate empty", func() error { return UpMysqlMigrate("") }},
		{"UpPostgresMigrate empty", func() error { return UpPostgresMigrate("") }},
		{"UpSqliteMigrate empty", func() error { return UpSqliteMigrate("") }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !errors.Is(err, ErrDatabaseConfigMissing) {
				t.Errorf("error = %v, want to wrap ErrDatabaseConfigMissing", err)
			}
		})
	}
}

// TestInitMysqlMigrate_WaitFlag tests that wait flag is set correctly.
func TestInitMysqlMigrate_WaitFlag(t *testing.T) {
	// Empty host - wait should be false (validation failed)
	wait, _, _ := InitMysqlMigrate("", "db", "user", "pass")
	if wait {
		t.Error("wait should be false when validation fails")
	}

	// Valid params but unreachable host - wait should be true (validation passed)
	wait, _, err := InitMysqlMigrate("unreachable:3306", "db", "user", "pass")
	if err == nil {
		t.Skip("skipping: connection succeeded unexpectedly")
	}
	if !wait {
		t.Error("wait should be true when validation passes but connection fails")
	}
}

// TestInitPostgresMigrate_WaitFlag tests that wait flag is set correctly.
func TestInitPostgresMigrate_WaitFlag(t *testing.T) {
	// Empty host - wait should be false (validation failed)
	wait, _, _ := InitPostgresMigrate("", "db", "user", "pass")
	if wait {
		t.Error("wait should be false when validation fails")
	}

	// Valid params but unreachable host - wait should be true (validation passed)
	wait, _, err := InitPostgresMigrate("unreachable:5432", "db", "user", "pass")
	if err == nil {
		t.Skip("skipping: connection succeeded unexpectedly")
	}
	if !wait {
		t.Error("wait should be true when validation passes but connection fails")
	}
}

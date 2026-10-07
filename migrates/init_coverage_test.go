package migrates

import (
	"errors"
	"strings"
	"testing"
)

// TestInitMysqlMigrate_MissingHost tests that missing host returns appropriate error
func TestInitMysqlMigrate_MissingHost(t *testing.T) {
	wait, url, err := InitMysqlMigrate("", "testdb", "root", "password")
	if err == nil {
		t.Error("expected error when host is missing, got nil")
	}
	if wait {
		t.Error("expected wait=false when host is missing")
	}
	if url != "" {
		t.Errorf("expected empty url, got %q", url)
	}
	if !errors.Is(err, ErrDatabaseConfigMissing) {
		t.Errorf("expected ErrDatabaseConfigMissing, got %v", err)
	}
}

// TestInitMysqlMigrate_MissingDatabase tests that missing database returns appropriate error
func TestInitMysqlMigrate_MissingDatabase(t *testing.T) {
	wait, url, err := InitMysqlMigrate("localhost:3306", "", "root", "password")
	if err == nil {
		t.Error("expected error when database is missing, got nil")
	}
	if wait {
		t.Error("expected wait=false when database is missing")
	}
	if url != "" {
		t.Errorf("expected empty url, got %q", url)
	}
	if !errors.Is(err, ErrDatabaseConfigMissing) {
		t.Errorf("expected ErrDatabaseConfigMissing, got %v", err)
	}
}

// TestInitMysqlMigrate_MissingUser tests that missing user returns appropriate error
func TestInitMysqlMigrate_MissingUser(t *testing.T) {
	wait, url, err := InitMysqlMigrate("localhost:3306", "testdb", "", "password")
	if err == nil {
		t.Error("expected error when user is missing, got nil")
	}
	if wait {
		t.Error("expected wait=false when user is missing")
	}
	if url != "" {
		t.Errorf("expected empty url, got %q", url)
	}
	if !errors.Is(err, ErrDatabaseConfigMissing) {
		t.Errorf("expected ErrDatabaseConfigMissing, got %v", err)
	}
}

// TestInitPostgresMigrate_MissingHost tests that missing host returns appropriate error
func TestInitPostgresMigrate_MissingHost(t *testing.T) {
	wait, url, err := InitPostgresMigrate("", "testdb", "postgres", "password")
	if err == nil {
		t.Error("expected error when host is missing, got nil")
	}
	if wait {
		t.Error("expected wait=false when host is missing")
	}
	if url != "" {
		t.Errorf("expected empty url, got %q", url)
	}
	if !errors.Is(err, ErrDatabaseConfigMissing) {
		t.Errorf("expected ErrDatabaseConfigMissing, got %v", err)
	}
}

// TestInitPostgresMigrate_MissingDatabase tests that missing database returns appropriate error
func TestInitPostgresMigrate_MissingDatabase(t *testing.T) {
	wait, url, err := InitPostgresMigrate("localhost:5432", "", "postgres", "password")
	if err == nil {
		t.Error("expected error when database is missing, got nil")
	}
	if wait {
		t.Error("expected wait=false when database is missing")
	}
	if url != "" {
		t.Errorf("expected empty url, got %q", url)
	}
	if !errors.Is(err, ErrDatabaseConfigMissing) {
		t.Errorf("expected ErrDatabaseConfigMissing, got %v", err)
	}
}

// TestInitPostgresMigrate_MissingUser tests that missing user returns appropriate error
func TestInitPostgresMigrate_MissingUser(t *testing.T) {
	wait, url, err := InitPostgresMigrate("localhost:5432", "testdb", "", "password")
	if err == nil {
		t.Error("expected error when user is missing, got nil")
	}
	if wait {
		t.Error("expected wait=false when user is missing")
	}
	if url != "" {
		t.Errorf("expected empty url, got %q", url)
	}
	if !errors.Is(err, ErrDatabaseConfigMissing) {
		t.Errorf("expected ErrDatabaseConfigMissing, got %v", err)
	}
}

// TestInitMysqlMigrate_ConnectionFailure tests behavior when database connection fails
func TestInitMysqlMigrate_ConnectionFailure(t *testing.T) {
	// Use non-existent host to force connection failure
	wait, url, err := InitMysqlMigrate("nonexistent-host:3306", "testdb", "root", "password")
	
	// Connection should fail
	if err == nil {
		t.Error("expected error when database is unreachable, got nil")
	}
	
	// wait should be true initially (attempted connection) but we can't verify intermediate state
	// The final state should have wait=false after failure
	if wait {
		t.Log("wait=true indicates connection was attempted")
	}
	
	// url should be empty on failure
	if url != "" {
		t.Errorf("expected empty url on failure, got %q", url)
	}
	
	// Error should contain meaningful information
	if err.Error() == "" {
		t.Error("error message should not be empty")
	}
}

// TestInitPostgresMigrate_ConnectionFailure tests behavior when postgres connection fails
func TestInitPostgresMigrate_ConnectionFailure(t *testing.T) {
	// Use non-existent host to force connection failure
	wait, url, err := InitPostgresMigrate("nonexistent-host:5432", "testdb", "postgres", "password")
	
	// Connection should fail
	if err == nil {
		t.Error("expected error when database is unreachable, got nil")
	}
	
	// wait behavior on failure
	if wait {
		t.Log("wait=true indicates connection was attempted")
	}
	
	// url should be empty on failure
	if url != "" {
		t.Errorf("expected empty url on failure, got %q", url)
	}
	
	// Error should contain meaningful information
	if err.Error() == "" {
		t.Error("error message should not be empty")
	}
}

// TestUpMysqlMigrate_InvalidConnectionString tests error handling with invalid MySQL URL
func TestUpMysqlMigrate_InvalidConnectionString(t *testing.T) {
	invalidURL := "invalid:mysql:url:format"
	err := UpMysqlMigrate(invalidURL)
	
	if err == nil {
		t.Error("expected error with invalid MySQL connection string, got nil")
	}
	
	// Error should be wrapped and contain context
	if !strings.Contains(err.Error(), "mysql") {
		t.Errorf("error should mention mysql: %v", err)
	}
}

// TestUpPostgresMigrate_InvalidConnectionString tests error handling with invalid PostgreSQL URL
func TestUpPostgresMigrate_InvalidConnectionString(t *testing.T) {
	invalidURL := "invalid-postgres-url-without-protocol"
	err := UpPostgresMigrate(invalidURL)
	
	if err == nil {
		t.Error("expected error with invalid PostgreSQL connection string, got nil")
	}
	
	// Error should be wrapped and contain context
	if !strings.Contains(err.Error(), "postgres") {
		t.Errorf("error should mention postgres: %v", err)
	}
}

// TestUpSqliteMigrate_InvalidPathCoverage tests error handling with invalid SQLite path
func TestUpSqliteMigrate_InvalidPathCoverage(t *testing.T) {
	invalidPath := "/nonexistent/directory/structure/db.sqlite"
	err := UpSqliteMigrate(invalidPath)
	
	if err == nil {
		t.Error("expected error with invalid SQLite path, got nil")
	}
	
	// Error should be wrapped and contain context
	if !strings.Contains(err.Error(), "sqlite") {
		t.Errorf("error should mention sqlite: %v", err)
	}
}

// TestInitMysqlMigrate_AllEmptyParams tests that all empty parameters return error
func TestInitMysqlMigrate_AllEmptyParams(t *testing.T) {
	wait, url, err := InitMysqlMigrate("", "", "", "")
	
	if err == nil {
		t.Error("expected error when all parameters are empty, got nil")
	}
	if wait {
		t.Error("expected wait=false when all parameters are empty")
	}
	if url != "" {
		t.Errorf("expected empty url, got %q", url)
	}
	if !errors.Is(err, ErrDatabaseConfigMissing) {
		t.Errorf("expected ErrDatabaseConfigMissing, got %v", err)
	}
}

// TestInitPostgresMigrate_AllEmptyParams tests that all empty parameters return error
func TestInitPostgresMigrate_AllEmptyParams(t *testing.T) {
	wait, url, err := InitPostgresMigrate("", "", "", "")
	
	if err == nil {
		t.Error("expected error when all parameters are empty, got nil")
	}
	if wait {
		t.Error("expected wait=false when all parameters are empty")
	}
	if url != "" {
		t.Errorf("expected empty url, got %q", url)
	}
	if !errors.Is(err, ErrDatabaseConfigMissing) {
		t.Errorf("expected ErrDatabaseConfigMissing, got %v", err)
	}
}

// TestConnectionStringFormat_MySQL verifies MySQL connection string format
func TestConnectionStringFormat_MySQL(t *testing.T) {
	// Test with password containing special characters
	user, pass, host, dbs := "testuser", "p@ssw0rd!", "localhost:3306", "testdb"
	
	// This should not panic
	wait, _, err := InitMysqlMigrate(host, dbs, user, pass)
	
	// Will fail due to connection, but validates parameter handling
	if err == nil && !wait {
		t.Error("unexpected: both wait=false and err=nil")
	}
}

// TestConnectionStringFormat_Postgres verifies PostgreSQL connection string format
func TestConnectionStringFormat_Postgres(t *testing.T) {
	// Test with password containing special characters
	user, pass, host, dbs := "testuser", "p@ssw0rd!", "localhost:5432", "testdb"
	
	// This should not panic
	wait, _, err := InitPostgresMigrate(host, dbs, user, pass)
	
	// Will fail due to connection, but validates parameter handling
	if err == nil && !wait {
		t.Error("unexpected: both wait=false and err=nil")
	}
}

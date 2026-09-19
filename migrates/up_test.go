package migrates

import (
	"errors"
	"strings"
	"testing"
)

func TestUpMysqlMigrate_EmptyConnectionString(t *testing.T) {
	err := UpMysqlMigrate("")
	if err == nil {
		t.Error("expected error for empty connection string, got nil")
	}
	if !errors.Is(err, ErrDatabaseConfigMissing) {
		t.Errorf("expected error to wrap ErrDatabaseConfigMissing, got: %v", err)
	}
	if !strings.Contains(err.Error(), "mysql connection string is empty") {
		t.Errorf("expected error message to contain 'mysql connection string is empty', got: %v", err)
	}
}

func TestUpPostgresMigrate_EmptyConnectionString(t *testing.T) {
	err := UpPostgresMigrate("")
	if err == nil {
		t.Error("expected error for empty connection string, got nil")
	}
	if !errors.Is(err, ErrDatabaseConfigMissing) {
		t.Errorf("expected error to wrap ErrDatabaseConfigMissing, got: %v", err)
	}
	if !strings.Contains(err.Error(), "postgres connection string is empty") {
		t.Errorf("expected error message to contain 'postgres connection string is empty', got: %v", err)
	}
}

func TestUpSqliteMigrate_EmptyConnectionString(t *testing.T) {
	err := UpSqliteMigrate("")
	if err == nil {
		t.Error("expected error for empty connection string, got nil")
	}
	if !errors.Is(err, ErrDatabaseConfigMissing) {
		t.Errorf("expected error to wrap ErrDatabaseConfigMissing, got: %v", err)
	}
	if !strings.Contains(err.Error(), "sqlite connection string is empty") {
		t.Errorf("expected error message to contain 'sqlite connection string is empty', got: %v", err)
	}
}

func TestUpMysqlMigrate_InvalidConnectionString(t *testing.T) {
	// Test with an invalid connection string that will fail to ping
	err := UpMysqlMigrate("invalid:invalid@tcp(localhost:9999)/nonexistent")
	if err == nil {
		t.Error("expected error for invalid connection string, got nil")
	}
	// Should fail at ping stage
	if !strings.Contains(err.Error(), "ping") {
		t.Logf("error message: %v (may fail at different stages depending on environment)", err)
	}
}

func TestUpPostgresMigrate_InvalidConnectionString(t *testing.T) {
	// Test with an invalid connection string that will fail to ping
	err := UpPostgresMigrate("postgres://invalid:***@localhost:9999/nonexistent?sslmode=disable")
	if err == nil {
		t.Error("expected error for invalid connection string, got nil")
	}
	// Should fail at ping stage
	if !strings.Contains(err.Error(), "ping") {
		t.Logf("error message: %v (may fail at different stages depending on environment)", err)
	}
}

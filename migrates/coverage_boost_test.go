package migrates

import (
	"errors"
	"strings"
	"testing"
)

// TestInitMysqlMigrate_ConfigValidation tests configuration validation for MySQL
func TestInitMysqlMigrate_ConfigValidation(t *testing.T) {
	tests := []struct {
		name     string
		host     string
		dbs      string
		user     string
		pass     string
		wantErr  bool
		wantWait bool
		errMsg   string
	}{
		{
			name:     "empty host",
			host:     "",
			dbs:      "testdb",
			user:     "root",
			pass:     "pass",
			wantErr:  true,
			wantWait: false,
			errMsg:   "database config not found",
		},
		{
			name:     "empty database",
			host:     "localhost:3306",
			dbs:      "",
			user:     "root",
			pass:     "pass",
			wantErr:  true,
			wantWait: false,
			errMsg:   "database config not found",
		},
		{
			name:     "empty user",
			host:     "localhost:3306",
			dbs:      "testdb",
			user:     "",
			pass:     "pass",
			wantErr:  true,
			wantWait: false,
			errMsg:   "database config not found",
		},
		{
			name:     "all empty",
			host:     "",
			dbs:      "",
			user:     "",
			pass:     "",
			wantErr:  true,
			wantWait: false,
			errMsg:   "database config not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wait, rtul, err := InitMysqlMigrate(tt.host, tt.dbs, tt.user, tt.pass)

			if (err != nil) != tt.wantErr {
				t.Errorf("InitMysqlMigrate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if wait != tt.wantWait {
				t.Errorf("InitMysqlMigrate() wait = %v, want %v", wait, tt.wantWait)
			}

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
					return
				}
				if !errors.Is(err, ErrDatabaseConfigMissing) {
					t.Errorf("expected ErrDatabaseConfigMissing, got %v", err)
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("error message should contain %q, got %q", tt.errMsg, err.Error())
				}
				if rtul != "" {
					t.Errorf("expected empty connection string on error, got %q", rtul)
				}
			}
		})
	}
}

// TestInitPostgresMigrate_ConfigValidation tests configuration validation for PostgreSQL
func TestInitPostgresMigrate_ConfigValidation(t *testing.T) {
	tests := []struct {
		name     string
		host     string
		dbs      string
		user     string
		pass     string
		wantErr  bool
		wantWait bool
		errMsg   string
	}{
		{
			name:     "empty host",
			host:     "",
			dbs:      "testdb",
			user:     "postgres",
			pass:     "pass",
			wantErr:  true,
			wantWait: false,
			errMsg:   "database config not found",
		},
		{
			name:     "empty database",
			host:     "localhost:5432",
			dbs:      "",
			user:     "postgres",
			pass:     "pass",
			wantErr:  true,
			wantWait: false,
			errMsg:   "database config not found",
		},
		{
			name:     "empty user",
			host:     "localhost:5432",
			dbs:      "testdb",
			user:     "",
			pass:     "pass",
			wantErr:  true,
			wantWait: false,
			errMsg:   "database config not found",
		},
		{
			name:     "all empty",
			host:     "",
			dbs:      "",
			user:     "",
			pass:     "",
			wantErr:  true,
			wantWait: false,
			errMsg:   "database config not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wait, rtul, err := InitPostgresMigrate(tt.host, tt.dbs, tt.user, tt.pass)

			if (err != nil) != tt.wantErr {
				t.Errorf("InitPostgresMigrate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if wait != tt.wantWait {
				t.Errorf("InitPostgresMigrate() wait = %v, want %v", wait, tt.wantWait)
			}

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
					return
				}
				if !errors.Is(err, ErrDatabaseConfigMissing) {
					t.Errorf("expected ErrDatabaseConfigMissing, got %v", err)
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("error message should contain %q, got %q", tt.errMsg, err.Error())
				}
				if rtul != "" {
					t.Errorf("expected empty connection string on error, got %q", rtul)
				}
			}
		})
	}
}

// TestInitMysqlMigrate_ConnectionFailure tests MySQL connection with unreachable host
func TestInitMysqlMigrate_ConnectionFailure(t *testing.T) {
	// Use localhost:1 (almost certainly not listening) to simulate connection failure quickly
	wait, rtul, err := InitMysqlMigrate("127.0.0.1:1", "testdb", "root", "password")

	// Should return wait=true initially, then fail
	if err == nil {
		t.Log("Warning: connection succeeded unexpectedly (test may need adjustment)")
		return
	}

	if wait {
		t.Log("wait=true is acceptable for connection attempts")
	}

	if rtul != "" {
		t.Errorf("expected empty connection string on error, got %q", rtul)
	}

	// Error should contain meaningful information
	if !strings.Contains(err.Error(), "mysql") && !strings.Contains(err.Error(), "database") {
		t.Errorf("error should mention mysql or database, got: %v", err)
	}
}

// TestInitPostgresMigrate_ConnectionFailure tests PostgreSQL connection with unreachable host
func TestInitPostgresMigrate_ConnectionFailure(t *testing.T) {
	// Use localhost:1 (almost certainly not listening) to simulate connection failure quickly
	wait, rtul, err := InitPostgresMigrate("127.0.0.1:1", "testdb", "postgres", "password")

	if err == nil {
		t.Log("Warning: connection succeeded unexpectedly (test may need adjustment)")
		return
	}

	if wait {
		t.Log("wait=true is acceptable for connection attempts")
	}

	if rtul != "" {
		t.Errorf("expected empty connection string on error, got %q", rtul)
	}

	// Error should contain meaningful information
	if !strings.Contains(err.Error(), "postgres") && !strings.Contains(err.Error(), "database") {
		t.Errorf("error should mention postgres or database, got: %v", err)
	}
}

// TestInitMigrateErrorWrapping verifies Init functions wrap errors properly
func TestInitMigrateErrorWrapping(t *testing.T) {
	t.Run("InitMysqlMigrate empty config", func(t *testing.T) {
		_, _, err := InitMysqlMigrate("", "", "", "")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, ErrDatabaseConfigMissing) {
			t.Errorf("error should wrap ErrDatabaseConfigMissing, got %v", err)
		}
	})

	t.Run("InitPostgresMigrate empty config", func(t *testing.T) {
		_, _, err := InitPostgresMigrate("", "", "", "")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, ErrDatabaseConfigMissing) {
			t.Errorf("error should wrap ErrDatabaseConfigMissing, got %v", err)
		}
	})
}

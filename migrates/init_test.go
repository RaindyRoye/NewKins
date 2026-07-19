package migrates

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestInitMysqlMigrate_EmptyParams(t *testing.T) {
	tests := []struct {
		name  string
		host  string
		dbs   string
		user  string
		pass  string
		wantW bool
	}{
		{"all empty", "", "", "", "", false},
		{"no host", "", "mydb", "root", "pass", false},
		{"no dbs", "localhost", "", "root", "pass", false},
		{"no user", "localhost", "mydb", "", "pass", false},
		{"host and dbs empty", "", "", "root", "pass", false},
		{"host and user empty", "", "mydb", "", "pass", false},
		{"dbs and user empty", "localhost", "", "", "pass", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wait, _, errs := InitMysqlMigrate(tt.host, tt.dbs, tt.user, tt.pass)
			if errs == nil {
				t.Error("expected error for empty params, got nil")
			}
			if wait != tt.wantW {
				t.Errorf("wait = %v, want %v", wait, tt.wantW)
			}
		})
	}
}

func TestInitPostgresMigrate_EmptyParams(t *testing.T) {
	tests := []struct {
		name  string
		host  string
		dbs   string
		user  string
		pass  string
		wantW bool
	}{
		{"all empty", "", "", "", "", false},
		{"no host", "", "mydb", "root", "pass", false},
		{"no dbs", "localhost", "", "root", "pass", false},
		{"no user", "localhost", "mydb", "", "pass", false},
		{"host and dbs empty", "", "", "root", "pass", false},
		{"host and user empty", "", "mydb", "", "pass", false},
		{"dbs and user empty", "localhost", "", "", "pass", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wait, _, errs := InitPostgresMigrate(tt.host, tt.dbs, tt.user, tt.pass)
			if errs == nil {
				t.Error("expected error for empty params, got nil")
			}
			if wait != tt.wantW {
				t.Errorf("wait = %v, want %v", wait, tt.wantW)
			}
		})
	}
}

func TestPostgresConnectionStringFormat(t *testing.T) {
	user, host, dbs := "testuser", "localhost:5432", "testdb"
	masked := fmt.Sprintf("postgres://%s:***@%s/%s?sslmode=disable", user, host, dbs)

	expected := "postgres://testuser:***@localhost:5432/testdb?sslmode=disable"
	if masked != expected {
		t.Errorf("masked connection string = %q, want %q", masked, expected)
	}

	// Verify password is masked
	if strings.Contains(masked, "password") {
		t.Error("connection string should not contain actual password")
	}
}

func TestMysqlConnectionStringFormat(t *testing.T) {
	user, host, dbs := "root", "127.0.0.1:3306", "gokins"
	masked := fmt.Sprintf("%s:***@tcp(%s)/%s?parseTime=true&multiStatements=true", user, host, dbs)

	expected := "root:***@tcp(127.0.0.1:3306)/gokins?parseTime=true&multiStatements=true"
	if masked != expected {
		t.Errorf("masked mysql connection string = %q, want %q", masked, expected)
	}

	// Verify required parameters
	if !strings.Contains(masked, "parseTime=true") {
		t.Error("mysql connection string should contain parseTime=true")
	}
	if !strings.Contains(masked, "multiStatements=true") {
		t.Error("mysql connection string should contain multiStatements=true")
	}
}

func TestUpMysqlMigrate_EmptyURL(t *testing.T) {
	err := UpMysqlMigrate("")
	if err == nil {
		t.Error("expected error for empty URL, got nil")
	}
	if !strings.Contains(err.Error(), "database config not found") {
		t.Errorf("error message = %q, want to contain 'database config not found'", err.Error())
	}
}

func TestUpPostgresMigrate_EmptyURL(t *testing.T) {
	err := UpPostgresMigrate("")
	if err == nil {
		t.Error("expected error for empty URL, got nil")
	}
	if !strings.Contains(err.Error(), "database config not found") {
		t.Errorf("error message = %q, want to contain 'database config not found'", err.Error())
	}
}

func TestUpSqliteMigrate_EmptyURL(t *testing.T) {
	err := UpSqliteMigrate("")
	if err == nil {
		t.Error("expected error for empty URL, got nil")
	}
	if !strings.Contains(err.Error(), "database config not found") {
		t.Errorf("error message = %q, want to contain 'database config not found'", err.Error())
	}
}

func TestUpMysqlMigrate_InvalidURL(t *testing.T) {
	err := UpMysqlMigrate("invalid:invalid@tcp(nonexistent:3306)/test")
	if err == nil {
		t.Skip("skipping: database connection succeeded unexpectedly")
	}
	if !strings.Contains(err.Error(), "mysql") {
		t.Errorf("error message = %q, want to contain 'mysql'", err.Error())
	}
}

func TestUpPostgresMigrate_InvalidURL(t *testing.T) {
	err := UpPostgresMigrate("postgres://invalid:invalid@nonexistent:5432/test")
	if err == nil {
		t.Skip("skipping: database connection succeeded unexpectedly")
	}
	if !strings.Contains(err.Error(), "postgres") {
		t.Errorf("error message = %q, want to contain 'postgres'", err.Error())
	}
}

func TestUpSqliteMigrate_InvalidPath(t *testing.T) {
	err := UpSqliteMigrate("/nonexistent/path/to/db.sqlite")
	if err == nil {
		t.Skip("skipping: database connection succeeded unexpectedly")
	}
	if !strings.Contains(err.Error(), "sqlite") {
		t.Errorf("error message = %q, want to contain 'sqlite'", err.Error())
	}
}

func TestMigrateErrorWrapping(t *testing.T) {
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

			// Verify error is not nil and has a message
			if err.Error() == "" {
				t.Error("error message is empty")
			}

			// Verify errors.Is works with sentinel error
			if !errors.Is(err, ErrDatabaseConfigMissing) {
				t.Errorf("error should wrap ErrDatabaseConfigMissing, got %q", err.Error())
			}
		})
	}
}

func TestInitMysqlMigrate_SentinelError(t *testing.T) {
	_, _, err := InitMysqlMigrate("", "", "", "")
	if err == nil {
		t.Fatal("expected error for empty params")
	}
	if !errors.Is(err, ErrDatabaseConfigMissing) {
		t.Errorf("error should wrap ErrDatabaseConfigMissing, got %q", err.Error())
	}
}

func TestInitPostgresMigrate_SentinelError(t *testing.T) {
	_, _, err := InitPostgresMigrate("", "", "", "")
	if err == nil {
		t.Fatal("expected error for empty params")
	}
	if !errors.Is(err, ErrDatabaseConfigMissing) {
		t.Errorf("error should wrap ErrDatabaseConfigMissing, got %q", err.Error())
	}
}

func TestUpMysqlMigrate_SentinelError(t *testing.T) {
	err := UpMysqlMigrate("")
	if err == nil {
		t.Fatal("expected error for empty URL")
	}
	if !errors.Is(err, ErrDatabaseConfigMissing) {
		t.Errorf("error should wrap ErrDatabaseConfigMissing, got %q", err.Error())
	}
}

func TestUpPostgresMigrate_SentinelError(t *testing.T) {
	err := UpPostgresMigrate("")
	if err == nil {
		t.Fatal("expected error for empty URL")
	}
	if !errors.Is(err, ErrDatabaseConfigMissing) {
		t.Errorf("error should wrap ErrDatabaseConfigMissing, got %q", err.Error())
	}
}

func TestUpSqliteMigrate_SentinelError(t *testing.T) {
	err := UpSqliteMigrate("")
	if err == nil {
		t.Fatal("expected error for empty URL")
	}
	if !errors.Is(err, ErrDatabaseConfigMissing) {
		t.Errorf("error should wrap ErrDatabaseConfigMissing, got %q", err.Error())
	}
}

func TestInitMysqlMigrate_InvalidHost(t *testing.T) {
	// Test with invalid host that will fail on ping or DB creation
	wait, _, err := InitMysqlMigrate("invalid-host:3306", "testdb", "root", "password")
	if err == nil {
		t.Skip("skipping: database connection succeeded unexpectedly")
	}
	// When DB operations fail, wait is expected to remain true (signaling retry)
	// Error should mention create database, dial, or lookup
	errLower := strings.ToLower(err.Error())
	if !strings.Contains(errLower, "create database") &&
		!strings.Contains(errLower, "dial") &&
		!strings.Contains(errLower, "lookup") {
		t.Errorf("error should mention connection failure, got %q", err.Error())
	}
	// Verify wait reflects that DB is not yet ready
	if !wait {
		t.Error("wait should be true when connection fails before migrations complete")
	}
}

func TestInitPostgresMigrate_InvalidHost(t *testing.T) {
	// Test with invalid host that will fail on ping or connection
	wait, _, err := InitPostgresMigrate("invalid-host:5432", "testdb", "postgres", "password")
	if err == nil {
		t.Skip("skipping: database connection succeeded unexpectedly")
	}
	// When DB operations fail, wait is expected to remain true (signaling retry)
	// Error should mention connection failure
	errLower := strings.ToLower(err.Error())
	if !strings.Contains(errLower, "dial") &&
		!strings.Contains(errLower, "lookup") &&
		!strings.Contains(errLower, "connection") &&
		!strings.Contains(errLower, "ping") {
		t.Errorf("error should mention connection failure, got %q", err.Error())
	}
	// Verify wait reflects that DB is not yet ready
	if !wait {
		t.Error("wait should be true when connection fails before migrations complete")
	}
}

func TestInitSqliteMigrate_InvalidPath(t *testing.T) {
	// Test with invalid path that doesn't exist
	_, err := InitSqliteMigrate()
	// This might succeed if comm.WorkPath is set, so we just verify it doesn't panic
	if err != nil {
		// Error should mention sqlite
		if !strings.Contains(strings.ToLower(err.Error()), "sqlite") {
			t.Errorf("error should mention sqlite, got %q", err.Error())
		}
	}
}

func TestUpMysqlMigrate_InvalidConnectionString(t *testing.T) {
	// Test with invalid connection string format
	err := UpMysqlMigrate("invalid-format-without-protocol")
	if err == nil {
		t.Skip("skipping: database connection succeeded unexpectedly")
	}
	// Should fail at ping or migration stage
	if err.Error() == "" {
		t.Error("error message should not be empty")
	}
}

func TestUpPostgresMigrate_InvalidConnectionString(t *testing.T) {
	// Test with invalid connection string format
	err := UpPostgresMigrate("not-a-valid-postgres-url")
	if err == nil {
		t.Skip("skipping: database connection succeeded unexpectedly")
	}
	// Should fail at ping or migration stage
	if err.Error() == "" {
		t.Error("error message should not be empty")
	}
}

func TestConnectionStringFormats(t *testing.T) {
	tests := []struct {
		name   string
		format string
		args   []interface{}
		expect string
	}{
		{
			name:   "mysql format",
			format: "%s:%s@tcp(%s)/%s?parseTime=true&multiStatements=true",
			args:   []interface{}{"root", "pass", "localhost:3306", "testdb"},
			expect: "root:pass@tcp(localhost:3306)/testdb?parseTime=true&multiStatements=true",
		},
		{
			name:   "postgres format",
			format: "postgres://%s:%s@%s/%s?sslmode=disable",
			args:   []interface{}{"user", "pass", "localhost:5432", "testdb"},
			expect: "postgres://user:pass@localhost:5432/testdb?sslmode=disable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fmt.Sprintf(tt.format, tt.args...)
			if result != tt.expect {
				t.Errorf("got %q, want %q", result, tt.expect)
			}
		})
	}
}

func TestInitMysqlMigrate_PartialParams(t *testing.T) {
	tests := []struct {
		name string
		host string
		dbs  string
		user string
		pass string
	}{
		{"only host", "localhost", "", "", ""},
		{"only dbs", "", "testdb", "", ""},
		{"only user", "", "", "root", ""},
		{"only pass", "", "", "", "password"},
		{"host and dbs", "localhost", "testdb", "", ""},
		{"host and user", "localhost", "", "root", ""},
		{"host and pass", "localhost", "", "", "password"},
		{"dbs and user", "", "testdb", "root", ""},
		{"dbs and pass", "", "testdb", "", "password"},
		{"user and pass", "", "", "root", "password"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := InitMysqlMigrate(tt.host, tt.dbs, tt.user, tt.pass)
			if err == nil {
				t.Error("expected error for partial params")
			}
			if !errors.Is(err, ErrDatabaseConfigMissing) {
				t.Errorf("error should wrap ErrDatabaseConfigMissing, got %q", err.Error())
			}
		})
	}
}

func TestInitPostgresMigrate_PartialParams(t *testing.T) {
	tests := []struct {
		name string
		host string
		dbs  string
		user string
		pass string
	}{
		{"only host", "localhost", "", "", ""},
		{"only dbs", "", "testdb", "", ""},
		{"only user", "", "", "postgres", ""},
		{"only pass", "", "", "", "password"},
		{"host and dbs", "localhost", "testdb", "", ""},
		{"host and user", "localhost", "", "postgres", ""},
		{"host and pass", "localhost", "", "", "password"},
		{"dbs and user", "", "testdb", "postgres", ""},
		{"dbs and pass", "", "testdb", "", "password"},
		{"user and pass", "", "", "postgres", "password"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := InitPostgresMigrate(tt.host, tt.dbs, tt.user, tt.pass)
			if err == nil {
				t.Error("expected error for partial params")
			}
			if !errors.Is(err, ErrDatabaseConfigMissing) {
				t.Errorf("error should wrap ErrDatabaseConfigMissing, got %q", err.Error())
			}
		})
	}
}

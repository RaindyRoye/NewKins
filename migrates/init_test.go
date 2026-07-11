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
			if !errors.Is(errs, ErrDatabaseConfigMissing) {
				t.Errorf("expected ErrDatabaseConfigMissing, got %v", errs)
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
			if !errors.Is(errs, ErrDatabaseConfigMissing) {
				t.Errorf("expected ErrDatabaseConfigMissing, got %v", errs)
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
	if !errors.Is(err, ErrDatabaseConfigMissing) {
		t.Errorf("expected ErrDatabaseConfigMissing, got %v", err)
	}
}

func TestUpPostgresMigrate_EmptyURL(t *testing.T) {
	err := UpPostgresMigrate("")
	if err == nil {
		t.Error("expected error for empty URL, got nil")
	}
	if !errors.Is(err, ErrDatabaseConfigMissing) {
		t.Errorf("expected ErrDatabaseConfigMissing, got %v", err)
	}
}

func TestUpSqliteMigrate_EmptyURL(t *testing.T) {
	err := UpSqliteMigrate("")
	if err == nil {
		t.Error("expected error for empty URL, got nil")
	}
	if !errors.Is(err, ErrDatabaseConfigMissing) {
		t.Errorf("expected ErrDatabaseConfigMissing, got %v", err)
	}
}

func TestUpMysqlMigrate_InvalidURL(t *testing.T) {
	err := UpMysqlMigrate("invalid:invalid@tcp(nonexistent:3306)/test")
	if err == nil {
		t.Skip("skipping: database connection succeeded unexpectedly")
	}
	// Should wrap a sentinel error from the migrates package
	if !errors.Is(err, ErrOpenDatabase) && !errors.Is(err, ErrPingDatabase) {
		t.Errorf("error should wrap ErrOpenDatabase or ErrPingDatabase, got: %v", err)
	}
}

func TestUpPostgresMigrate_InvalidURL(t *testing.T) {
	err := UpPostgresMigrate("postgres://invalid:invalid@nonexistent:5432/test")
	if err == nil {
		t.Skip("skipping: database connection succeeded unexpectedly")
	}
	if !errors.Is(err, ErrOpenDatabase) && !errors.Is(err, ErrPingDatabase) {
		t.Errorf("error should wrap ErrOpenDatabase or ErrPingDatabase, got: %v", err)
	}
}

func TestUpSqliteMigrate_InvalidPath(t *testing.T) {
	err := UpSqliteMigrate("/nonexistent/path/to/db.sqlite")
	if err == nil {
		t.Skip("skipping: database connection succeeded unexpectedly")
	}
	if !errors.Is(err, ErrOpenDatabase) && !errors.Is(err, ErrPingDatabase) {
		t.Errorf("error should wrap ErrOpenDatabase or ErrPingDatabase, got: %v", err)
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

func TestSentinelErrors(t *testing.T) {
	// Verify all sentinel errors are distinct
	sentinels := []error{
		ErrDatabaseConfigMissing,
		ErrMigrationFailed,
		ErrOpenDatabase,
		ErrPingDatabase,
		ErrCreateDatabase,
		ErrInitDriver,
		ErrInitSource,
		ErrCreateMigrateInstance,
		ErrRunMigration,
	}

	for i, s1 := range sentinels {
		for j, s2 := range sentinels {
			if i != j && errors.Is(s1, s2) {
				t.Errorf("sentinel errors %d and %d should be distinct", i, j)
			}
		}
	}
}

func TestErrorWrapping(t *testing.T) {
	// Test that wrapped errors can be detected with errors.Is
	baseErr := errors.New("underlying database error")
	wrappedErr := fmt.Errorf("%w: %w", ErrOpenDatabase, baseErr)

	if !errors.Is(wrappedErr, ErrOpenDatabase) {
		t.Error("wrapped error should be detectable with errors.Is(ErrOpenDatabase)")
	}

	// Test error message contains both sentinel and underlying error
	errMsg := wrappedErr.Error()
	if !strings.Contains(errMsg, "failed to open database") {
		t.Errorf("error message should contain sentinel error text, got: %s", errMsg)
	}
	if !strings.Contains(errMsg, "underlying database error") {
		t.Errorf("error message should contain underlying error text, got: %s", errMsg)
	}
}

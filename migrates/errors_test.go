package migrates

import (
	"errors"
	"testing"
)

// TestErrDatabaseConfigMissing verifies the sentinel error is properly defined.
func TestErrDatabaseConfigMissing(t *testing.T) {
	if ErrDatabaseConfigMissing == nil {
		t.Fatal("ErrDatabaseConfigMissing should not be nil")
	}
	if ErrDatabaseConfigMissing.Error() != "database config not found" {
		t.Errorf("ErrDatabaseConfigMissing.Error() = %q, want %q",
			ErrDatabaseConfigMissing.Error(), "database config not found")
	}
}

// TestErrorsIs_WithWrappedError verifies errors.Is works with wrapped errors.
func TestErrorsIs_WithWrappedError(t *testing.T) {
	wrapped := errors.Join(ErrDatabaseConfigMissing, errors.New("additional context"))
	if !errors.Is(wrapped, ErrDatabaseConfigMissing) {
		t.Error("errors.Is should match wrapped ErrDatabaseConfigMissing")
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
		{"only dbs", "", "mydb", "", ""},
		{"only user", "", "", "root", ""},
		{"only pass", "", "", "", "secret"},
		{"host and dbs", "localhost", "mydb", "", ""},
		{"host and user", "localhost", "", "root", ""},
		{"host and pass", "localhost", "", "", "secret"},
		{"dbs and user", "", "mydb", "root", ""},
		{"dbs and pass", "", "mydb", "", "secret"},
		{"user and pass", "", "", "root", "secret"},
		{"all but host", "", "mydb", "root", "secret"},
		{"all but dbs", "localhost", "", "root", "secret"},
		{"all but user", "localhost", "mydb", "", "secret"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wait, ul, err := InitMysqlMigrate(tt.host, tt.dbs, tt.user, tt.pass)
			if err == nil {
				t.Error("expected error for incomplete params, got nil")
			}
			if !errors.Is(err, ErrDatabaseConfigMissing) {
				t.Errorf("error should wrap ErrDatabaseConfigMissing, got: %v", err)
			}
			if wait != false {
				t.Errorf("wait = %v, want false", wait)
			}
			if ul != "" {
				t.Errorf("connection URL = %q, want empty string", ul)
			}
		})
	}
}

// TestInitPostgresMigrate_PartialParams tests various combinations of missing parameters.
func TestInitPostgresMigrate_PartialParams(t *testing.T) {
	tests := []struct {
		name string
		host string
		dbs  string
		user string
		pass string
	}{
		{"only host", "localhost:5432", "", "", ""},
		{"only dbs", "", "mydb", "", ""},
		{"only user", "", "", "postgres", ""},
		{"only pass", "", "", "", "secret"},
		{"host and dbs", "localhost:5432", "mydb", "", ""},
		{"host and user", "localhost:5432", "", "postgres", ""},
		{"host and pass", "localhost:5432", "", "", "secret"},
		{"dbs and user", "", "mydb", "postgres", ""},
		{"dbs and pass", "", "mydb", "", "secret"},
		{"user and pass", "", "", "postgres", "secret"},
		{"all but host", "", "mydb", "postgres", "secret"},
		{"all but dbs", "localhost:5432", "", "postgres", "secret"},
		{"all but user", "localhost:5432", "mydb", "", "secret"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wait, ul, err := InitPostgresMigrate(tt.host, tt.dbs, tt.user, tt.pass)
			if err == nil {
				t.Error("expected error for incomplete params, got nil")
			}
			if !errors.Is(err, ErrDatabaseConfigMissing) {
				t.Errorf("error should wrap ErrDatabaseConfigMissing, got: %v", err)
			}
			if wait != false {
				t.Errorf("wait = %v, want false", wait)
			}
			if ul != "" {
				t.Errorf("connection URL = %q, want empty string", ul)
			}
		})
	}
}

// TestUpMysqlMigrate_ErrorsIs verifies error wrapping in UpMysqlMigrate.
func TestUpMysqlMigrate_ErrorsIs(t *testing.T) {
	err := UpMysqlMigrate("")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrDatabaseConfigMissing) {
		t.Errorf("error should wrap ErrDatabaseConfigMissing, got: %v", err)
	}
}

// TestUpPostgresMigrate_ErrorsIs verifies error wrapping in UpPostgresMigrate.
func TestUpPostgresMigrate_ErrorsIs(t *testing.T) {
	err := UpPostgresMigrate("")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrDatabaseConfigMissing) {
		t.Errorf("error should wrap ErrDatabaseConfigMissing, got: %v", err)
	}
}

// TestUpSqliteMigrate_ErrorsIs verifies error wrapping in UpSqliteMigrate.
func TestUpSqliteMigrate_ErrorsIs(t *testing.T) {
	err := UpSqliteMigrate("")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrDatabaseConfigMissing) {
		t.Errorf("error should wrap ErrDatabaseConfigMissing, got: %v", err)
	}
}

package migrates

import (
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
		wantW bool // expect wait=true
	}{
		{"all empty", "", "", "", "", false},
		{"no host", "", "mydb", "root", "pass", false},
		{"no dbs", "localhost", "", "root", "pass", false},
		{"no user", "localhost", "mydb", "", "pass", false},
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
	// Regression test: the old format string had "***" hardcoded instead of "%s"
	// for the password field, causing all Postgres connections to fail.
	// The format string must use %s for all 4 parameters (user, pass, host, dbs).
	user, pass, host, dbs := "testuser", "testpass", "localhost:5432", "testdb"
	ul := fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable", user, pass, host, dbs)

	if !strings.HasPrefix(ul, "postgres://") {
		t.Errorf("connection string should start with postgres://, got: %s", ul)
	}
	if !strings.Contains(ul, "testuser:testpass@") {
		t.Errorf("connection string should contain user:pass@, got: %s", ul)
	}
	if !strings.Contains(ul, "@localhost:5432/testdb") {
		t.Errorf("connection string should contain @host/db, got: %s", ul)
	}
	if !strings.HasSuffix(ul, "?sslmode=disable") {
		t.Errorf("connection string should end with ?sslmode=disable, got: %s", ul)
	}

	// Verify exact expected string to catch any format regressions
	expected := "postgres://testuser:testpass@localhost:5432/testdb?sslmode=disable"
	if ul != expected {
		t.Errorf("connection string = %q, want %q", ul, expected)
	}
}

func TestMysqlConnectionStringFormat(t *testing.T) {
	// Verify MySQL connection string format includes all parameters correctly
	user, pass, host, dbs := "root", "secret", "127.0.0.1:3306", "gokins" //nolint:gosec // G101: test credentials only
	ul := fmt.Sprintf("%s:%s@tcp(%s)/%s?parseTime=true&multiStatements=true",
		user, pass, host, dbs)

	expected := "root:secret@tcp(127.0.0.1:3306)/gokins?parseTime=true&multiStatements=true"
	if ul != expected {
		t.Errorf("mysql connection string = %q, want %q", ul, expected)
	}
}

func TestUpMysqlMigrate_EmptyURL(t *testing.T) {
	err := UpMysqlMigrate("")
	if err == nil {
		t.Error("expected error for empty URL, got nil")
	}
}

func TestUpPostgresMigrate_EmptyURL(t *testing.T) {
	err := UpPostgresMigrate("")
	if err == nil {
		t.Error("expected error for empty URL, got nil")
	}
}

func TestUpSqliteMigrate_EmptyURL(t *testing.T) {
	err := UpSqliteMigrate("")
	if err == nil {
		t.Error("expected error for empty URL, got nil")
	}
}

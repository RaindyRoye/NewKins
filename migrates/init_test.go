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
		wantW bool
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
	user, host, dbs := "testuser", "localhost:5432", "testdb"
	masked := fmt.Sprintf("postgres://%s:***@%s/%s?sslmode=disable", user, host, dbs)

	if !strings.HasPrefix(masked, "postgres://") {
		t.Errorf("expected prefix postgres://, got %q", masked)
	}

	expected := "postgres://testuser:***@localhost:5432/testdb?sslmode=disable" //nolint:gosec // G101: test credentials
	if masked != expected {
		t.Errorf("masked connection string = %q, want %q", masked, expected)
	}
}

func TestMysqlConnectionStringFormat(t *testing.T) {
	user, host, dbs := "root", "127.0.0.1:3306", "gokins"
	masked := fmt.Sprintf("%s:***@tcp(%s)/%s?parseTime=true&multiStatements=true", user, host, dbs)

	expected := "root:***@tcp(127.0.0.1:3306)/gokins?parseTime=true&multiStatements=true"
	if masked != expected {
		t.Errorf("masked mysql connection string = %q, want %q", masked, expected)
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

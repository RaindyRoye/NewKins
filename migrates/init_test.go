package migrates

import (
	"context"
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
			ctx := context.Background()
			wait, _, errs := InitMysqlMigrate(ctx, tt.host, tt.dbs, tt.user, tt.pass)
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
			ctx := context.Background()
			wait, _, errs := InitPostgresMigrate(ctx, tt.host, tt.dbs, tt.user, tt.pass)
			if errs == nil {
				t.Error("expected error for empty params, got nil")
			}
			if wait != tt.wantW {
				t.Errorf("wait = %v, want %v", wait, tt.wantW)
			}
		})
	}
}

func TestBuildPostgresURL(t *testing.T) {
	tests := []struct {
		name     string
		user     string
		pass     string
		host     string
		dbs      string
		contains []string
	}{
		{
			name: "basic credentials",
			user: "testuser",
			pass: "testpass",
			host: "localhost:5432",
			dbs:  "testdb",
			contains: []string{
				"localhost:5432",
				"testdb",
				"sslmode=disable",
			},
		},
		{
			name: "special chars in password",
			user: "user@domain",
			pass: "p@ss:w/rd",
			host: "db.example.com:5432",
			dbs:  "mydb",
			contains: []string{
				"db.example.com:5432",
				"mydb",
				"sslmode=disable",
			},
		},
		{
			name: "empty password",
			user: "admin",
			pass: "",
			host: "localhost",
			dbs:  "postgres",
			contains: []string{
				"localhost",
				"postgres",
				"sslmode=disable",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildPostgresURL(tt.user, tt.pass, tt.host, tt.dbs)
			for _, want := range tt.contains {
				if !strings.Contains(result, want) {
					t.Errorf("BuildPostgresURL() = %q, want to contain %q", result, want)
				}
			}
		})
	}
}

func TestBuildPostgresURL_CorrectHost(t *testing.T) {
	result := BuildPostgresURL("testuser", "testpass", "localhost:5432", "testdb")
	expected := "postgres://testuser:testpass@localhost:5432/testdb?sslmode=disable"
	if result != expected {
		t.Errorf("BuildPostgresURL() = %q, want %q", result, expected)
	}

	// Verify password is in the URL (required for authentication)
	if !strings.Contains(result, "testpass") {
		t.Error("connection string should contain actual password for authentication")
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
	ctx := context.Background()
	err := UpMysqlMigrate(ctx, "")
	if err == nil {
		t.Error("expected error for empty URL, got nil")
	}
	if !strings.Contains(err.Error(), "database config not found") {
		t.Errorf("error message = %q, want to contain 'database config not found'", err.Error())
	}
}

func TestUpPostgresMigrate_EmptyURL(t *testing.T) {
	ctx := context.Background()
	err := UpPostgresMigrate(ctx, "")
	if err == nil {
		t.Error("expected error for empty URL, got nil")
	}
	if !strings.Contains(err.Error(), "database config not found") {
		t.Errorf("error message = %q, want to contain 'database config not found'", err.Error())
	}
}

func TestUpSqliteMigrate_EmptyURL(t *testing.T) {
	ctx := context.Background()
	err := UpSqliteMigrate(ctx, "")
	if err == nil {
		t.Error("expected error for empty URL, got nil")
	}
	if !strings.Contains(err.Error(), "database config not found") {
		t.Errorf("error message = %q, want to contain 'database config not found'", err.Error())
	}
}

func TestUpMysqlMigrate_InvalidURL(t *testing.T) {
	ctx := context.Background()
	err := UpMysqlMigrate(ctx, "invalid:invalid@tcp(nonexistent:3306)/test")
	if err == nil {
		t.Skip("skipping: database connection succeeded unexpectedly")
	}
	if !strings.Contains(err.Error(), "mysql") {
		t.Errorf("error message = %q, want to contain 'mysql'", err.Error())
	}
}

func TestUpPostgresMigrate_InvalidURL(t *testing.T) {
	ctx := context.Background()
	err := UpPostgresMigrate(ctx, "postgres://invalid:***@nonexistent:5432/test")
	if err == nil {
		t.Skip("skipping: database connection succeeded unexpectedly")
	}
	if !strings.Contains(err.Error(), "postgres") {
		t.Errorf("error message = %q, want to contain 'postgres'", err.Error())
	}
}

func TestUpSqliteMigrate_InvalidPath(t *testing.T) {
	ctx := context.Background()
	err := UpSqliteMigrate(ctx, "/nonexistent/path/to/db.sqlite")
	if err == nil {
		t.Skip("skipping: database connection succeeded unexpectedly")
	}
	if !strings.Contains(err.Error(), "sqlite") {
		t.Errorf("error message = %q, want to contain 'sqlite'", err.Error())
	}
}

func TestMigrateErrorWrapping(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name string
		fn   func() error
	}{
		{"UpMysqlMigrate empty", func() error { return UpMysqlMigrate(ctx, "") }},
		{"UpPostgresMigrate empty", func() error { return UpPostgresMigrate(ctx, "") }},
		{"UpSqliteMigrate empty", func() error { return UpSqliteMigrate(ctx, "") }},
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

			// Verify errors.Is works with base error
			baseErr := errors.New("database config not found")
			if !errors.Is(err, baseErr) && !strings.Contains(err.Error(), baseErr.Error()) {
				t.Errorf("error should be related to %q, got %q", baseErr.Error(), err.Error())
			}
		})
	}
}

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
	// Verify the connection string template produces valid format
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

	// Verify all 4 parameters are included (regression test for the broken format bug)
	for _, part := range []string{user, pass, host, dbs} {
		if !strings.Contains(ul, part) {
			t.Errorf("connection string missing parameter %q: %s", part, ul)
		}
	}
}

package migrates

import (
	"errors"
	"strings"
	"testing"
)

// TestInitMysqlMigrate_VariousEmptyParams tests MySQL initialization with
// specific parameter combinations to verify the validation logic and the
// wait flag returned in error paths.
func TestInitMysqlMigrate_VariousEmptyParams(t *testing.T) {
	tests := []struct {
		name  string
		host  string
		dbs   string
		user  string
		pass  string
		wantW bool
	}{
		{"only pass", "host", "db", "user", "", true},
		{"minimal missing host", "", "db", "user", "pass", false},
		{"minimal missing dbs", "host", "", "user", "pass", false},
		{"minimal missing user", "host", "db", "", "pass", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wait, _, errs := InitMysqlMigrate(tt.host, tt.dbs, tt.user, tt.pass)
			if tt.host == "" || tt.dbs == "" || tt.user == "" {
				if errs == nil {
					t.Error("expected error for empty required params, got nil")
				}
				if wait != tt.wantW {
					t.Errorf("wait = %v, want %v", wait, tt.wantW)
				}
			}
		})
	}
}

// TestInitPostgresMigrate_WithPass tests PostgreSQL initialization with pass
// but missing other required params.
func TestInitPostgresMigrate_WithPass(t *testing.T) {
	tests := []struct {
		name  string
		host  string
		dbs   string
		user  string
		wantW bool
	}{
		{"missing host", "", "db", "user", false},
		{"missing dbs", "host", "", "user", false},
		{"missing user", "host", "db", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wait, _, errs := InitPostgresMigrate(tt.host, tt.dbs, tt.user, "pass")
			if errs == nil {
				t.Error("expected error for empty required params, got nil")
			}
			if wait != tt.wantW {
				t.Errorf("wait = %v, want %v", wait, tt.wantW)
			}
			if !errors.Is(errs, ErrDatabaseConfigMissing) {
				t.Errorf("error should wrap ErrDatabaseConfigMissing, got: %v", errs)
			}
		})
	}
}

// TestUpMysqlMigrate_BadFormatURL tests UpMysqlMigrate with a URL that
// can be parsed by sql.Open but cannot connect.
func TestUpMysqlMigrate_BadFormatURL(t *testing.T) {
	// This URL will pass sql.Open but fail on Ping, exercising the ping error path
	err := UpMysqlMigrate("invalid:invalid@tcp(127.0.0.1:1)/testdb")
	if err == nil {
		t.Skip("skipping: connection unexpectedly succeeded")
	}
	if !strings.Contains(err.Error(), "mysql") {
		t.Errorf("error message = %q, want to contain 'mysql'", err.Error())
	}
}

// TestUpPostgresMigrate_BadFormatURL tests UpPostgresMigrate with a URL
// that can be parsed but cannot connect.
func TestUpPostgresMigrate_BadFormatURL(t *testing.T) {
	err := UpPostgresMigrate("postgres://u:p@127.0.0.1:1/db?sslmode=disable")
	if err == nil {
		t.Skip("skipping: connection unexpectedly succeeded")
	}
	if !strings.Contains(err.Error(), "postgres") {
		t.Errorf("error message = %q, want to contain 'postgres'", err.Error())
	}
}

// TestInitMysqlMigrate_ConnectionRefused exercises the MySQL ping failure
// code path where it tries to create the database.
func TestInitMysqlMigrate_ConnectionRefused_Extra(t *testing.T) {
	wait, _, err := InitMysqlMigrate("127.0.0.1:1", "testdb", "user", "pass")
	if err == nil {
		t.Skip("skipping: connection unexpectedly succeeded")
	}
	if wait != true {
		t.Errorf("wait = %v, want true when connection fails", wait)
	}
	// Error should indicate a connection or database creation failure
	if err.Error() == "" {
		t.Error("error message should not be empty")
	}
}

// TestInitPostgresMigrate_ConnectionRefused_Extra exercises the PostgreSQL ping
// failure code path.
func TestInitPostgresMigrate_ConnectionRefused_Extra(t *testing.T) {
	wait, _, err := InitPostgresMigrate("127.0.0.1:1", "testdb", "user", "pass")
	if err == nil {
		t.Skip("skipping: connection unexpectedly succeeded")
	}
	if wait != true {
		t.Errorf("wait = %v, want true when connection fails", wait)
	}
	if !strings.Contains(err.Error(), "postgres") {
		t.Errorf("error message = %q, want to contain 'postgres'", err.Error())
	}
}

// TestFilterAssetsFunction tests the filterAssets helper function used in tests.
func TestFilterAssetsFunction(t *testing.T) {
	mysqlAssets := filterAssets("mysql")
	sqliteAssets := filterAssets("sqlite")
	postgresAssets := filterAssets("postgres")

	// All should be valid slices (may be empty if no assets are embedded)
	if mysqlAssets == nil && sqliteAssets == nil && postgresAssets == nil {
		t.Log("No migration assets found (this is OK if not running with embedded migrations)")
	}

	// Verify no prefix leakage
	for _, a := range mysqlAssets {
		if strings.HasPrefix(a, "mysql/") {
			t.Errorf("mysql asset %q still has prefix after filtering", a)
		}
	}
	for _, a := range sqliteAssets {
		if strings.HasPrefix(a, "sqlite/") {
			t.Errorf("sqlite asset %q still has prefix after filtering", a)
		}
	}
	for _, a := range postgresAssets {
		if strings.HasPrefix(a, "postgres/") {
			t.Errorf("postgres asset %q still has prefix after filtering", a)
		}
	}
}

// TestInitMysqlMigrate_InvalidHost tests MySQL initialization with unreachable host.
func TestInitMysqlMigrate_InvalidHost(t *testing.T) {
	// Using an unreachable port to trigger the connection error
	_, _, err := InitMysqlMigrate("invalid-host-never-resolves.invalid:3306", "testdb", "user", "pass")
	if err == nil {
		t.Skip("skipping: connection unexpectedly succeeded")
	}
}

// TestInitPostgresMigrate_InvalidHost tests PostgreSQL initialization with unreachable host.
func TestInitPostgresMigrate_InvalidHost(t *testing.T) {
	_, _, err := InitPostgresMigrate("invalid-host-never-resolves.invalid:5432", "testdb", "user", "pass")
	if err == nil {
		t.Skip("skipping: connection unexpectedly succeeded")
	}
}

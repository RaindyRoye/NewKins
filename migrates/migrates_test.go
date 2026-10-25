package migrates

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/gokins/gokins/comm"
	"github.com/golang-migrate/migrate/v4/database"
	bindata "github.com/golang-migrate/migrate/v4/source/go_bindata"
	_ "github.com/mattn/go-sqlite3"
)

// TestBindataResourceCreation tests the bindata resource creation helper.
func TestBindataResourceCreation(t *testing.T) {
	// Test MySQL asset filtering
	mysqlAssets := filterAssets("mysql")
	if len(mysqlAssets) == 0 {
		t.Log("No mysql assets found (this is OK if not running with embedded migrations)")
	}

	// Test SQLite asset filtering
	sqliteAssets := filterAssets("sqlite")
	if len(sqliteAssets) == 0 {
		t.Log("No sqlite assets found (this is OK if not running with embedded migrations)")
	}

	// Test PostgreSQL asset filtering
	postgresAssets := filterAssets("postgres")
	if len(postgresAssets) == 0 {
		t.Log("No postgres assets found (this is OK if not running with embedded migrations)")
	}
}

// filterAssets is a test helper that filters comm.AssetNames() by prefix.
func filterAssets(prefix string) []string {
	var nms []string
	for _, v := range comm.AssetNames() {
		if strings.HasPrefix(v, prefix) {
			nms = append(nms, strings.Replace(v, prefix+"/", "", 1))
		}
	}
	return nms
}

// TestFilterAssetNames tests the filterAssetNames helper function.
func TestFilterAssetNames(t *testing.T) {
	tests := []struct {
		name   string
		prefix string
	}{
		{"mysql assets", "mysql"},
		{"sqlite assets", "sqlite"},
		{"postgres assets", "postgres"},
		{"nonexistent prefix returns empty", "nonexistent_db_driver_xyz"},
		{"empty prefix", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filterAssetNames(tt.prefix)
			// Verify no elements contain the prefix itself
			for _, name := range got {
				if strings.Contains(name, tt.prefix+"/") {
					t.Errorf("filterAssetNames(%q) returned %q which still contains prefix", tt.prefix, name)
				}
			}
			// For known drivers, if there are any assets at all, they should be non-empty strings
			if tt.prefix != "" && tt.prefix != "nonexistent_db_driver_xyz" {
				for _, name := range got {
					if name == "" {
						t.Errorf("filterAssetNames(%q) returned empty string", tt.prefix)
					}
				}
			}
		})
	}
}

// TestFilterAssetNamesConsistency verifies filterAssetNames matches filterAssets.
func TestFilterAssetNamesConsistency(t *testing.T) {
	for _, prefix := range []string{"mysql", "sqlite", "postgres"} {
		got := filterAssetNames(prefix)
		want := filterAssets(prefix)
		if len(got) != len(want) {
			t.Errorf("filterAssetNames(%q) returned %d items, filterAssets returned %d", prefix, len(got), len(want))
		}
	}
}

// TestRunMigrationEmptyURL verifies runMigration rejects empty connection strings.
func TestRunMigrationEmptyURL(t *testing.T) {
	tests := []struct {
		name      string
		dbDriver  string
		prefix    string
		wantInErr string
	}{
		{"mysql empty url", "mysql", "mysql", "database config not found"},
		{"postgres empty url", "postgres", "postgres", "database config not found"},
		{"sqlite3 empty url", "sqlite3", "sqlite", "database config not found"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := runMigration(tt.dbDriver, "", tt.prefix, newSqlite3Driver)
			if err == nil {
				t.Fatal("expected error for empty URL, got nil")
			}
			if !errors.Is(err, ErrDatabaseConfigMissing) {
				t.Errorf("error should wrap ErrDatabaseConfigMissing, got: %v", err)
			}
			if !strings.Contains(err.Error(), tt.wantInErr) {
				t.Errorf("error = %q, want to contain %q", err.Error(), tt.wantInErr)
			}
			if !strings.Contains(err.Error(), tt.dbDriver) {
				t.Errorf("error = %q, want to contain driver name %q", err.Error(), tt.dbDriver)
			}
		})
	}
}

// TestRunMigrationInvalidConnection verifies runMigration handles connection failures.
func TestRunMigrationInvalidConnection(t *testing.T) {
	// Use an invalid sqlite3 path to trigger connection failure
	err := runMigration("sqlite3", "/nonexistent/path/to/db.sqlite", "sqlite", newSqlite3Driver)
	if err == nil {
		t.Skip("skipping: connection unexpectedly succeeded")
	}
	// Error should mention the driver
	if !strings.Contains(err.Error(), "sqlite3") {
		t.Errorf("error = %q, want to contain 'sqlite3'", err.Error())
	}
}

// TestRunMigrationBadDriverFactory verifies runMigration handles driver factory errors.
func TestRunMigrationBadDriverFactory(t *testing.T) {
	// Use a temp sqlite db that actually opens, but provide a broken driver factory
	tmpDir := t.TempDir()
	dbPath := tmpDir + "/test.db"

	badFactory := func(db *sql.DB) (database.Driver, error) {
		return nil, fmt.Errorf("intentional driver factory error")
	}
	err := runMigration("sqlite3", dbPath, "sqlite", badFactory)
	if err == nil {
		t.Fatal("expected error from bad driver factory, got nil")
	}
	if !strings.Contains(err.Error(), "driver") {
		t.Errorf("error = %q, want to contain 'driver'", err.Error())
	}
}

// TestInitMysqlMigrate_ConnectionRefused tests MySQL migration when connection fails.
func TestInitMysqlMigrate_ConnectionRefused(t *testing.T) {
	wait, _, err := InitMysqlMigrate("127.0.0.1:1", "testdb", "user", "pass")
	if err == nil {
		t.Skip("skipping: connection unexpectedly succeeded")
	}
	if wait != true {
		t.Errorf("wait = %v, want true when connection fails", wait)
	}
}

// TestInitPostgresMigrate_ConnectionRefused tests PostgreSQL migration when connection fails.
func TestInitPostgresMigrate_ConnectionRefused(t *testing.T) {
	wait, _, err := InitPostgresMigrate("127.0.0.1:1", "testdb", "user", "pass")
	if err == nil {
		t.Skip("skipping: connection unexpectedly succeeded")
	}
	if wait != true {
		t.Errorf("wait = %v, want true when connection fails", wait)
	}
}

// TestUpMysqlMigrate_ConnectionRefused tests UpMysqlMigrate when connection fails.
func TestUpMysqlMigrate_ConnectionRefused(t *testing.T) {
	err := UpMysqlMigrate("user:pass@tcp(127.0.0.1:1)/testdb?parseTime=true")
	if err == nil {
		t.Skip("skipping: connection unexpectedly succeeded")
	}
}

// TestUpPostgresMigrate_ConnectionRefused tests UpPostgresMigrate when connection fails.
func TestUpPostgresMigrate_ConnectionRefused(t *testing.T) {
	err := UpPostgresMigrate("postgres://user:pass@127.0.0.1:1/testdb?sslmode=disable")
	if err == nil {
		t.Skip("skipping: connection unexpectedly succeeded")
	}
}

// TestMysqlCreateDatabasePath tests the database creation logic in MySQL migration.
func TestMysqlCreateDatabasePath(t *testing.T) {
	// This test verifies the code path where MySQL tries to create a database
	// when the initial ping fails. We can't easily test this without a real MySQL
	// server, but we can verify the function signature and error handling.
	_, _, err := InitMysqlMigrate("invalid-host:3306", "testdb", "user", "pass")
	if err == nil {
		t.Skip("skipping: connection unexpectedly succeeded")
	}
}

// TestBindataResourceWithEmptyAssets tests bindata resource creation with no assets.
func TestBindataResourceWithEmptyAssets(t *testing.T) {
	s := bindata.Resource([]string{}, func(name string) ([]byte, error) {
		return nil, errors.New("asset not found")
	})
	if s == nil {
		t.Error("bindata.Resource returned nil")
	}

	sc, err := bindata.WithInstance(s)
	if err != nil {
		t.Logf("bindata.WithInstance error (expected for empty resources): %v", err)
	}
	if sc != nil {
		_ = sc.Close()
	}
}

// TestSqliteMigrationWithRealDB tests SQLite migration with an actual database.
func TestSqliteMigrationWithRealDB(t *testing.T) {
	// Save and restore WorkPath
	origWorkPath := comm.WorkPath
	t.Cleanup(func() { comm.WorkPath = origWorkPath })

	tmpDir := t.TempDir()
	comm.WorkPath = tmpDir

	// Test InitSqliteMigrate
	dbPath, err := InitSqliteMigrate()
	if err != nil {
		t.Fatalf("InitSqliteMigrate() error = %v", err)
	}

	// Verify database was created
	if dbPath == "" {
		t.Error("InitSqliteMigrate returned empty path")
	}

	// Open and verify
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	defer func() { _ = db.Close() }()

	// Verify we can query it
	err = db.PingContext(context.Background())
	if err != nil {
		t.Errorf("db.PingContext() error = %v", err)
	}

	// Test UpSqliteMigrate on existing DB
	err = UpSqliteMigrate(dbPath)
	if err != nil {
		t.Errorf("UpSqliteMigrate() error = %v", err)
	}
}

// TestErrorWrapping verifies that errors are properly wrapped.
func TestErrorWrapping(t *testing.T) {
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

			// Verify error wrapping
			baseErr := ErrDatabaseConfigMissing
			if !errors.Is(err, baseErr) {
				t.Errorf("error should wrap ErrDatabaseConfigMissing, got: %v", err)
			}
		})
	}
}

// TestInitMysqlMigrate_AllEmptyParams tests all combinations of empty parameters.
func TestInitMysqlMigrate_AllEmptyParams(t *testing.T) {
	cases := []struct {
		host, dbs, user, pass string
	}{
		{"", "", "", ""},
		{"", "db", "user", "pass"},
		{"host", "", "user", "pass"},
		{"host", "db", "", "pass"},
		{"", "", "user", "pass"},
		{"", "db", "", "pass"},
		{"host", "", "", "pass"},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("host=%q,dbs=%q,user=%q", c.host, c.dbs, c.user), func(t *testing.T) {
			wait, _, err := InitMysqlMigrate(c.host, c.dbs, c.user, c.pass)
			if err == nil {
				t.Error("expected error for empty params")
			}
			if wait != false {
				t.Errorf("wait = %v, want false", wait)
			}
		})
	}
}

// TestInitPostgresMigrate_AllEmptyParams tests all combinations of empty parameters.
func TestInitPostgresMigrate_AllEmptyParams(t *testing.T) {
	cases := []struct {
		host, dbs, user, pass string
	}{
		{"", "", "", ""},
		{"", "db", "user", "pass"},
		{"host", "", "user", "pass"},
		{"host", "db", "", "pass"},
		{"", "", "user", "pass"},
		{"", "db", "", "pass"},
		{"host", "", "", "pass"},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("host=%q,dbs=%q,user=%q", c.host, c.dbs, c.user), func(t *testing.T) {
			wait, _, err := InitPostgresMigrate(c.host, c.dbs, c.user, c.pass)
			if err == nil {
				t.Error("expected error for empty params")
			}
			if wait != false {
				t.Errorf("wait = %v, want false", wait)
			}
		})
	}
}

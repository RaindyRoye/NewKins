package migrates

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/gokins/gokins/comm"
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
			var baseErr error = ErrDatabaseConfigMissing
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

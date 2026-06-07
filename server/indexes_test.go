package server

import (
	"testing"

	"github.com/gokins/gokins/comm"
	_ "github.com/mattn/go-sqlite3"
	"xorm.io/xorm"
)

func TestCreateIndexIfNotExists_NilDb(t *testing.T) {
	// ensureIndexes should not panic when comm.Db is nil.
	origDb := comm.Db
	comm.Db = nil
	defer func() { comm.Db = origDb }()

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("ensureIndexes panicked with nil Db: %v", r)
		}
	}()
	ensureIndexes()
}

func TestEnsureIndexes_WithSQLite(t *testing.T) {
	origDb := comm.Db
	origIsMySQL := comm.IsMySQL

	db, err := xorm.NewEngine("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer func() {
		_ = db.Close()
		comm.Db = origDb
		comm.IsMySQL = origIsMySQL
	}()

	comm.Db = db
	comm.IsMySQL = false

	// Create a test table
	_, err = db.Exec(`CREATE TABLE t_build (
		id VARCHAR(64) PRIMARY KEY,
		pipeline_id VARCHAR(64),
		pipeline_version_id VARCHAR(64)
	)`)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}

	// Run ensureIndexes — should not error or panic
	ensureIndexes()

	// Verify the index was created by querying sqlite_master
	var count int
	_, err = db.SQL("SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name='idx_build_pipeline_id'").Get(&count)
	if err != nil {
		t.Fatalf("query index: %v", err)
	}
	if count != 1 {
		t.Errorf("expected index idx_build_pipeline_id to exist, got count=%d", count)
	}

	// Run ensureIndexes again — should be idempotent (no error)
	ensureIndexes()
}

func TestCreateIndexIfNotExists_AlreadyExists(t *testing.T) {
	origDb := comm.Db
	origIsMySQL := comm.IsMySQL

	db, err := xorm.NewEngine("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer func() {
		_ = db.Close()
		comm.Db = origDb
		comm.IsMySQL = origIsMySQL
	}()

	comm.Db = db
	comm.IsMySQL = false

	_, err = db.Exec(`CREATE TABLE t_test (id VARCHAR(64) PRIMARY KEY, col1 VARCHAR(64))`)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}

	// Create the index first time
	if err := createIndexIfNotExists("t_test", "idx_test_col1", "col1"); err != nil {
		t.Fatalf("first create: %v", err)
	}

	// Create the index second time — should not error
	if err := createIndexIfNotExists("t_test", "idx_test_col1", "col1"); err != nil {
		t.Fatalf("second create (should be idempotent): %v", err)
	}
}

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
		pipeline_version_id VARCHAR(64),
		status VARCHAR(50)
	)`)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE t_org_pipe (
		aid BIGINT PRIMARY KEY,
		org_id VARCHAR(64),
		pipe_id VARCHAR(64)
	)`)
	if err != nil {
		t.Fatalf("create org_pipe table: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE t_artifact_package (
		id VARCHAR(64) PRIMARY KEY,
		repo_id VARCHAR(64),
		deleted INT
	)`)
	if err != nil {
		t.Fatalf("create artifact_package table: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE t_pipeline (
		id VARCHAR(64) PRIMARY KEY,
		uid VARCHAR(64),
		deleted INT
	)`)
	if err != nil {
		t.Fatalf("create pipeline table: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE t_pipeline_version (
		id VARCHAR(64) PRIMARY KEY,
		pipeline_id VARCHAR(64),
		name VARCHAR(100),
		deleted INT
	)`)
	if err != nil {
		t.Fatalf("create pipeline_version table: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE t_artifactory (
		id VARCHAR(64) PRIMARY KEY,
		uid VARCHAR(64),
		org_id VARCHAR(64),
		identifier VARCHAR(50),
		deleted INT
	)`)
	if err != nil {
		t.Fatalf("create artifactory table: %v", err)
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

	// Verify composite index on t_org_pipe
	var orgPipeCount int
	_, err = db.SQL("SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name='idx_orgpipe_org_pipe'").Get(&orgPipeCount)
	if err != nil {
		t.Fatalf("query org_pipe index: %v", err)
	}
	if orgPipeCount != 1 {
		t.Errorf("expected index idx_orgpipe_org_pipe to exist, got count=%d", orgPipeCount)
	}

	// Verify index on t_artifact_package
	var artPkgCount int
	_, err = db.SQL("SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name='idx_artpkg_deleted_repo'").Get(&artPkgCount)
	if err != nil {
		t.Fatalf("query artifact_package index: %v", err)
	}
	if artPkgCount != 1 {
		t.Errorf("expected index idx_artpkg_deleted_repo to exist, got count=%d", artPkgCount)
	}

	// Verify index on t_build status
	var buildStatusCount int
	_, err = db.SQL("SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name='idx_build_status'").Get(&buildStatusCount)
	if err != nil {
		t.Fatalf("query build status index: %v", err)
	}
	if buildStatusCount != 1 {
		t.Errorf("expected index idx_build_status to exist, got count=%d", buildStatusCount)
	}

	// Verify new composite indexes
	compositeTests := []struct {
		name  string
		table string
	}{
		{"idx_pipeline_uid_deleted", "t_pipeline"},
		{"idx_pipever_pipeid_deleted", "t_pipeline_version"},
		{"idx_pipever_pipeid_name", "t_pipeline_version"},
		{"idx_artifactory_org_deleted", "t_artifactory"},
		{"idx_artifactory_uid_deleted", "t_artifactory"},
		{"idx_build_pipeid_status", "t_build"},
	}
	for _, tt := range compositeTests {
		var cnt int
		_, err = db.SQL("SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name=?", tt.name).Get(&cnt)
		if err != nil {
			t.Fatalf("query index %s: %v", tt.name, err)
		}
		if cnt != 1 {
			t.Errorf("expected index %s on %s to exist, got count=%d", tt.name, tt.table, cnt)
		}
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

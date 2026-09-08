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

	_, err = db.Exec(`CREATE TABLE t_cmd_line (
		id VARCHAR(64) PRIMARY KEY,
		build_id VARCHAR(64),
		step_id VARCHAR(64)
	)`)
	if err != nil {
		t.Fatalf("create cmd_line table: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE t_pipeline (
		id VARCHAR(64) PRIMARY KEY,
		deleted INT
	)`)
	if err != nil {
		t.Fatalf("create pipeline table: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE t_pipeline_version (
		id VARCHAR(64) PRIMARY KEY,
		pipeline_id VARCHAR(64),
		deleted INT
	)`)
	if err != nil {
		t.Fatalf("create pipeline_version table: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE t_artifactory (
		id VARCHAR(64) PRIMARY KEY,
		identifier VARCHAR(64),
		org_id VARCHAR(64)
	)`)
	if err != nil {
		t.Fatalf("create artifactory table: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE t_user_org (
		id VARCHAR(64) PRIMARY KEY,
		uid VARCHAR(64),
		org_id VARCHAR(64)
	)`)
	if err != nil {
		t.Fatalf("create user_org table: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE t_artifact_package (
		id VARCHAR(64) PRIMARY KEY,
		repo_id VARCHAR(64),
		name VARCHAR(64),
		deleted INT
	)`)
	if err != nil {
		t.Fatalf("create artifact_package table: %v", err)
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

	// Verify composite index on t_cmd_line (build_id, step_id)
	var cmdLineBuildStepCount int
	_, err = db.SQL("SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name='idx_cmdline_build_step'").Get(&cmdLineBuildStepCount)
	if err != nil {
		t.Fatalf("query cmd_line build_step index: %v", err)
	}
	if cmdLineBuildStepCount != 1 {
		t.Errorf("expected index idx_cmdline_build_step to exist, got count=%d", cmdLineBuildStepCount)
	}

	// Verify composite index on t_pipeline (id, deleted)
	var pipelineIdDeletedCount int
	_, err = db.SQL("SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name='idx_pipeline_id_deleted'").Get(&pipelineIdDeletedCount)
	if err != nil {
		t.Fatalf("query pipeline id_deleted index: %v", err)
	}
	if pipelineIdDeletedCount != 1 {
		t.Errorf("expected index idx_pipeline_id_deleted to exist, got count=%d", pipelineIdDeletedCount)
	}

	// Verify composite index on t_pipeline_version (pipeline_id, deleted)
	var pipeverPipelineDeletedCount int
	_, err = db.SQL("SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name='idx_pipever_pipeline_deleted'").Get(&pipeverPipelineDeletedCount)
	if err != nil {
		t.Fatalf("query pipeline_version pipeline_deleted index: %v", err)
	}
	if pipeverPipelineDeletedCount != 1 {
		t.Errorf("expected index idx_pipever_pipeline_deleted to exist, got count=%d", pipeverPipelineDeletedCount)
	}

	// Verify composite index on t_artifactory (identifier, org_id)
	var artifactoryIdentifierOrgCount int
	_, err = db.SQL("SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name='idx_artifactory_identifier_org'").Get(&artifactoryIdentifierOrgCount)
	if err != nil {
		t.Fatalf("query artifactory identifier_org index: %v", err)
	}
	if artifactoryIdentifierOrgCount != 1 {
		t.Errorf("expected index idx_artifactory_identifier_org to exist, got count=%d", artifactoryIdentifierOrgCount)
	}

	// Verify composite index on t_user_org (uid, org_id)
	var userorgUidOrgCount int
	_, err = db.SQL("SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name='idx_userorg_uid_org'").Get(&userorgUidOrgCount)
	if err != nil {
		t.Fatalf("query user_org uid_org index: %v", err)
	}
	if userorgUidOrgCount != 1 {
		t.Errorf("expected index idx_userorg_uid_org to exist, got count=%d", userorgUidOrgCount)
	}

	// Verify composite index on t_artifact_package (deleted, repo_id, name)
	var artpkgDeletedRepoNameCount int
	_, err = db.SQL("SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name='idx_artpkg_deleted_repo_name'").Get(&artpkgDeletedRepoNameCount)
	if err != nil {
		t.Fatalf("query artifact_package deleted_repo_name index: %v", err)
	}
	if artpkgDeletedRepoNameCount != 1 {
		t.Errorf("expected index idx_artpkg_deleted_repo_name to exist, got count=%d", artpkgDeletedRepoNameCount)
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

package service

import (
	"context"
	"testing"
	"time"

	"github.com/gokins/gokins/comm"
	"github.com/gokins/gokins/model"
	_ "github.com/mattn/go-sqlite3"
	"xorm.io/xorm"
)

func setupBatchQueryTestDb(t *testing.T) *xorm.Engine {
	t.Helper()
	origDb := comm.Db
	t.Cleanup(func() { comm.Db = origDb })

	db, err := xorm.NewEngine("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("create sqlite engine: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	comm.Db = db

	_, err = db.Exec(`CREATE TABLE t_user (
		id VARCHAR(64) NOT NULL PRIMARY KEY,
		aid BIGINT,
		name VARCHAR(100),
		nick VARCHAR(100),
		avatar VARCHAR(500),
		pass VARCHAR(255),
		active INT DEFAULT 1,
		created DATETIME,
		login_time DATETIME
	)`)
	if err != nil {
		t.Fatalf("create user table: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE t_build (
		id VARCHAR(64) NOT NULL PRIMARY KEY,
		pipeline_id VARCHAR(64),
		pipeline_version_id VARCHAR(64),
		status VARCHAR(100),
		error VARCHAR(500),
		event VARCHAR(100),
		started DATETIME,
		finished DATETIME,
		created DATETIME,
		updated DATETIME,
		version VARCHAR(255)
	)`)
	if err != nil {
		t.Fatalf("create build table: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE t_artifact_package (
		id VARCHAR(64) NOT NULL PRIMARY KEY,
		repo_id VARCHAR(64),
		name VARCHAR(255),
		created DATETIME
	)`)
	if err != nil {
		t.Fatalf("create artifact_package table: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE t_artifact_version (
		id VARCHAR(64) NOT NULL PRIMARY KEY,
		package_id VARCHAR(64),
		version VARCHAR(100),
		created DATETIME
	)`)
	if err != nil {
		t.Fatalf("create artifact_version table: %v", err)
	}

	return db
}

func TestBatchGetUsers_Empty(t *testing.T) {
	result, err := BatchGetUsers(context.TODO(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty result, got %d entries", len(result))
	}
}

func TestBatchGetUsers_Success(t *testing.T) {
	db := setupBatchQueryTestDb(t)
	users := []*model.TUser{
		{Id: "u1", Name: "alice", Nick: "Alice", Active: 1, Created: time.Now(), LoginTime: time.Now()},
		{Id: "u2", Name: "bob", Nick: "Bob", Active: 1, Created: time.Now(), LoginTime: time.Now()},
		{Id: "u3", Name: "charlie", Nick: "Charlie", Active: 1, Created: time.Now(), LoginTime: time.Now()},
	}
	for _, u := range users {
		if _, err := db.Insert(u); err != nil {
			t.Fatalf("insert user: %v", err)
		}
	}

	result, err := BatchGetUsers(context.TODO(), []string{"u1", "u3", "nonexistent"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 users, got %d", len(result))
	}
	if result["u1"].Nick != "Alice" {
		t.Errorf("expected Alice, got %s", result["u1"].Nick)
	}
	if result["u3"].Nick != "Charlie" {
		t.Errorf("expected Charlie, got %s", result["u3"].Nick)
	}
	if _, exists := result["nonexistent"]; exists {
		t.Error("nonexistent user should not be in result")
	}
}

func TestBatchBuildCounts_Empty(t *testing.T) {
	result, err := BatchBuildCounts(context.TODO(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty result, got %d entries", len(result))
	}
}

func TestBatchBuildCounts_Success(t *testing.T) {
	db := setupBatchQueryTestDb(t)
	builds := []*model.TBuild{
		{Id: "b1", PipelineId: "p1", Status: "success", Created: time.Now()},
		{Id: "b2", PipelineId: "p1", Status: "failed", Created: time.Now()},
		{Id: "b3", PipelineId: "p2", Status: "success", Created: time.Now()},
	}
	for _, b := range builds {
		if _, err := db.Insert(b); err != nil {
			t.Fatalf("insert build: %v", err)
		}
	}

	result, err := BatchBuildCounts(context.TODO(), []string{"p1", "p2", "p3"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["p1"] != 2 {
		t.Errorf("expected 2 builds for p1, got %d", result["p1"])
	}
	if result["p2"] != 1 {
		t.Errorf("expected 1 build for p2, got %d", result["p2"])
	}
	if _, exists := result["p3"]; exists {
		t.Error("p3 should not have any builds")
	}
}

func TestBatchLatestBuilds_Empty(t *testing.T) {
	result, err := BatchLatestBuilds(context.TODO(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty result, got %d entries", len(result))
	}
}

func TestBatchLatestBuilds_Success(t *testing.T) {
	db := setupBatchQueryTestDb(t)
	now := time.Now()
	builds := []*model.TBuild{
		{Id: "b1", PipelineId: "p1", Status: "old", Created: now.Add(-time.Hour)},
		{Id: "b2", PipelineId: "p1", Status: "latest", Created: now},
		{Id: "b3", PipelineId: "p2", Status: "only", Created: now},
	}
	for _, b := range builds {
		if _, err := db.Insert(b); err != nil {
			t.Fatalf("insert build: %v", err)
		}
	}

	result, err := BatchLatestBuilds(context.TODO(), []string{"p1", "p2"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 results, got %d", len(result))
	}
	if result["p1"].Status != "latest" {
		t.Errorf("expected latest build for p1, got %s", result["p1"].Status)
	}
	if result["p2"].Status != "only" {
		t.Errorf("expected only build for p2, got %s", result["p2"].Status)
	}
}

func TestBatchLatestBuildsForVersions_Empty(t *testing.T) {
	result, err := BatchLatestBuildsForVersions(context.TODO(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty result, got %d entries", len(result))
	}
}

func TestBatchLatestBuildsForVersions_Success(t *testing.T) {
	db := setupBatchQueryTestDb(t)
	now := time.Now()
	builds := []*model.TBuild{
		{Id: "b1", PipelineVersionId: "pv1", Status: "old", Created: now.Add(-time.Hour)},
		{Id: "b2", PipelineVersionId: "pv1", Status: "latest", Created: now},
		{Id: "b3", PipelineVersionId: "pv2", Status: "only", Created: now},
	}
	for _, b := range builds {
		if _, err := db.Insert(b); err != nil {
			t.Fatalf("insert build: %v", err)
		}
	}

	result, err := BatchLatestBuildsForVersions(context.TODO(), []string{"pv1", "pv2"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 results, got %d", len(result))
	}
	if result["pv1"].Status != "latest" {
		t.Errorf("expected latest build for pv1, got %s", result["pv1"].Status)
	}
}

func TestBatchCountArtifactPackages_Empty(t *testing.T) {
	result, err := BatchCountArtifactPackages(context.TODO(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty result, got %d entries", len(result))
	}
}

func TestBatchCountArtifactPackages_Success(t *testing.T) {
	db := setupBatchQueryTestDb(t)
	packages := []map[string]interface{}{
		{"id": "pkg1", "repo_id": "r1", "name": "package-1", "created": time.Now()},
		{"id": "pkg2", "repo_id": "r1", "name": "package-2", "created": time.Now()},
		{"id": "pkg3", "repo_id": "r2", "name": "package-3", "created": time.Now()},
	}
	for _, p := range packages {
		if _, err := db.Table("t_artifact_package").Insert(p); err != nil {
			t.Fatalf("insert package: %v", err)
		}
	}

	result, err := BatchCountArtifactPackages(context.TODO(), []string{"r1", "r2"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["r1"] != 2 {
		t.Errorf("expected 2 packages for r1, got %d", result["r1"])
	}
	if result["r2"] != 1 {
		t.Errorf("expected 1 package for r2, got %d", result["r2"])
	}
}

func TestBatchCountArtifactVersions_Empty(t *testing.T) {
	result, err := BatchCountArtifactVersions(context.TODO(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty result, got %d entries", len(result))
	}
}

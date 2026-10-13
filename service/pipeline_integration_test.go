package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gokins/gokins/comm"
	"github.com/gokins/gokins/model"
	_ "github.com/mattn/go-sqlite3"
	"xorm.io/xorm"
)

// setupPipelineTestDB creates an isolated in-memory SQLite DB for pipeline tests.
func setupPipelineTestDB(t *testing.T) *xorm.Engine {
	t.Helper()
	eng, err := xorm.NewEngine("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("create test engine: %v", err)
	}
	comm.Db = eng
	t.Cleanup(func() {
		_ = eng.Close()
		comm.Db = nil
	})

	tables := []interface{}{
		&model.TPipeline{},
		&model.TPipelineConf{},
		&model.TPipelineVersion{},
		&model.TBuild{},
		&model.TStage{},
		&model.TStep{},
		&model.TOrgVar{},
		&model.TPipelineVar{},
		&model.TOrgPipe{},
	}
	for _, table := range tables {
		if err := eng.Sync2(table); err != nil {
			t.Fatalf("sync table %T: %v", table, err)
		}
	}
	return eng
}

// TestRun_IntegBasicPipeline verifies that Run creates pipeline version, build, stages, and steps
func TestRun_IntegBasicPipeline(t *testing.T) {
	eng := setupPipelineTestDB(t)
	ctx := context.Background()

	// Setup test data
	pipe := &model.TPipeline{
		Id:          "pipe-1",
		Name:        "test-pipeline",
		DisplayName: "Test Pipeline",
	}
	if _, err := eng.Insert(pipe); err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	pipeConf := &model.TPipelineConf{
		PipelineId: "pipe-1",
		Url:        "https://github.com/test/repo",
		Username:   "testuser",
		YmlContent: `
version: 1.0
stages:
  - stage: build
    name: build
    displayName: Build Stage
    steps:
      - step: shell
        name: compile
        displayName: Compile
        commands:
          - go build ./...
      - step: shell
        name: test
        displayName: Test
        commands:
          - go test ./...
  - stage: deploy
    name: deploy
    displayName: Deploy Stage
    steps:
      - step: shell
        name: deploy
        displayName: Deploy
        commands:
          - echo "deploying"
`,
	}
	if _, err := eng.Insert(pipeConf); err != nil {
		t.Fatalf("insert pipeline conf: %v", err)
	}

	// Execute Run
	tvp, rb, err := Run(ctx, "user-1", "pipe-1", "abc123", "push")
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}

	// Verify pipeline version was created
	if tvp == nil {
		t.Fatal("Run returned nil pipeline version")
	}
	if tvp.PipelineId != "pipe-1" {
		t.Errorf("tvp.PipelineId = %q, want %q", tvp.PipelineId, "pipe-1")
	}
	if tvp.Number != 1 {
		t.Errorf("tvp.Number = %d, want 1", tvp.Number)
	}
	if tvp.Sha != "abc123" {
		t.Errorf("tvp.Sha = %q, want %q", tvp.Sha, "abc123")
	}
	if tvp.Events != "push" {
		t.Errorf("tvp.Events = %q, want %q", tvp.Events, "push")
	}
	if tvp.PipelineName != "test-pipeline" {
		t.Errorf("tvp.PipelineName = %q, want %q", tvp.PipelineName, "test-pipeline")
	}

	// Verify runtime build
	if rb == nil {
		t.Fatal("Run returned nil build")
	}
	if rb.PipelineId != "pipe-1" {
		t.Errorf("rb.PipelineId = %q, want %q", rb.PipelineId, "pipe-1")
	}
	if rb.Status != "pending" {
		t.Errorf("rb.Status = %q, want %q", rb.Status, "pending")
	}
	if len(rb.Stages) != 2 {
		t.Fatalf("rb.Stages length = %d, want 2", len(rb.Stages))
	}

	// Verify stages
	buildStage := rb.Stages[0]
	if buildStage.Name != "build" {
		t.Errorf("buildStage.Name = %q, want %q", buildStage.Name, "build")
	}
	if len(buildStage.Steps) != 2 {
		t.Errorf("buildStage.Steps length = %d, want 2", len(buildStage.Steps))
	}

	// Verify database records
	var dbVersion model.TPipelineVersion
	ok, err := eng.Where("id=?", tvp.Id).Get(&dbVersion)
	if err != nil {
		t.Fatalf("query pipeline version: %v", err)
	}
	if !ok {
		t.Fatal("pipeline version not found in database")
	}
	if dbVersion.Number != 1 {
		t.Errorf("dbVersion.Number = %d, want 1", dbVersion.Number)
	}

	var dbBuild model.TBuild
	ok, err = eng.Where("id=?", rb.Id).Get(&dbBuild)
	if err != nil {
		t.Fatalf("query build: %v", err)
	}
	if !ok {
		t.Fatal("build not found in database")
	}
	if dbBuild.PipelineVersionId != tvp.Id {
		t.Errorf("dbBuild.PipelineVersionId = %q, want %q", dbBuild.PipelineVersionId, tvp.Id)
	}

	var stages []model.TStage
	if err := eng.Where("build_id=?", rb.Id).Find(&stages); err != nil {
		t.Fatalf("query stages: %v", err)
	}
	if len(stages) != 2 {
		t.Errorf("database stages count = %d, want 2", len(stages))
	}

	var steps []model.TStep
	if err := eng.Where("build_id=?", rb.Id).Find(&steps); err != nil {
		t.Fatalf("query steps: %v", err)
	}
	if len(steps) != 3 {
		t.Errorf("database steps count = %d, want 3", len(steps))
	}
}

// TestRun_IntegPipelineNotFound verifies error handling when pipeline config doesn't exist
func TestRun_IntegPipelineNotFound(t *testing.T) {
	setupPipelineTestDB(t)
	ctx := context.Background()

	_, _, err := Run(ctx, "user-1", "nonexistent", "abc123", "push")
	if !errors.Is(err, ErrPipelineNotFound) {
		t.Errorf("expected ErrPipelineNotFound, got: %v", err)
	}
}

// TestRun_IntegEmptyYmlContent verifies error handling when YAML content is empty
func TestRun_IntegEmptyYmlContent(t *testing.T) {
	eng := setupPipelineTestDB(t)
	ctx := context.Background()

	pipeConf := &model.TPipelineConf{
		PipelineId: "pipe-empty",
		YmlContent: "",
	}
	if _, err := eng.Insert(pipeConf); err != nil {
		t.Fatalf("insert pipeline conf: %v", err)
	}

	_, _, err := Run(ctx, "user-1", "pipe-empty", "abc123", "push")
	if !errors.Is(err, ErrPipelineYmlEmpty) {
		t.Errorf("expected ErrPipelineYmlEmpty, got: %v", err)
	}
}

// TestRun_IntegInvalidYml verifies error handling when YAML is malformed
func TestRun_IntegInvalidYml(t *testing.T) {
	eng := setupPipelineTestDB(t)
	ctx := context.Background()

	pipeConf := &model.TPipelineConf{
		PipelineId: "pipe-invalid",
		YmlContent: "invalid: yaml: content:",
	}
	if _, err := eng.Insert(pipeConf); err != nil {
		t.Fatalf("insert pipeline conf: %v", err)
	}

	_, _, err := Run(ctx, "user-1", "pipe-invalid", "abc123", "push")
	if err == nil {
		t.Error("expected error for invalid YAML, got nil")
	}
}

// TestReBuild_IntegBasic verifies that ReBuild creates a new version from an existing one
func TestReBuild_IntegBasic(t *testing.T) {
	eng := setupPipelineTestDB(t)
	ctx := context.Background()

	// Setup initial pipeline
	pipe := &model.TPipeline{
		Id:          "pipe-rebuild",
		Name:        "rebuild-pipeline",
		DisplayName: "Rebuild Pipeline",
	}
	if _, err := eng.Insert(pipe); err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	pipeConf := &model.TPipelineConf{
		PipelineId: "pipe-rebuild",
		Url:        "https://github.com/test/repo",
		YmlContent: `
stages:
  - stage: build
    name: build
    steps:
      - step: shell
        name: compile
        commands:
          - go build ./...
`,
	}
	if _, err := eng.Insert(pipeConf); err != nil {
		t.Fatalf("insert pipeline conf: %v", err)
	}

	// First run
	tvp1, _, err := Run(ctx, "user-1", "pipe-rebuild", "abc123", "push")
	if err != nil {
		t.Fatalf("first Run error: %v", err)
	}
	if tvp1.Number != 1 {
		t.Errorf("first run Number = %d, want 1", tvp1.Number)
	}

	// Modify the version content for rebuild
	tvp1.Content = `
stages:
  - stage: build
    name: build
    steps:
      - step: shell
        name: compile
        commands:
          - go build -v ./...
`

	// ReBuild
	tvp2, rb2, err := ReBuild(ctx, "user-1", tvp1)
	if err != nil {
		t.Fatalf("ReBuild error: %v", err)
	}

	// Verify new version
	if tvp2 == nil {
		t.Fatal("ReBuild returned nil pipeline version")
	}
	if tvp2.Number != 2 {
		t.Errorf("tvp2.Number = %d, want 2", tvp2.Number)
	}
	if tvp2.Sha != "abc123" {
		t.Errorf("tvp2.Sha = %q, want %q", tvp2.Sha, "abc123")
	}
	if tvp2.Events != "rebuild" {
		t.Errorf("tvp2.Events = %q, want %q", tvp2.Events, "rebuild")
	}

	// Verify runtime build
	if rb2 == nil {
		t.Fatal("ReBuild returned nil build")
	}
	if rb2.PipelineVersionId != tvp2.Id {
		t.Errorf("rb2.PipelineVersionId = %q, want %q", rb2.PipelineVersionId, tvp2.Id)
	}
}

// TestConvertVar_IntegOrgVars verifies that org variables are loaded and merged
func TestConvertVar_IntegOrgVars(t *testing.T) {
	eng := setupPipelineTestDB(t)
	ctx := context.Background()

	// Setup org vars
	orgVar := &model.TOrgVar{
		OrgId:  "org-1",
		Name:   "ORG_VAR",
		Value:  "org-value",
		Public: 1,
	}
	if _, err := eng.Insert(orgVar); err != nil {
		t.Fatalf("insert org var: %v", err)
	}

	// Setup org pipe mapping
	orgPipe := &model.TOrgPipe{
		OrgId:  "org-1",
		PipeId: "pipe-1",
	}
	if _, err := eng.Insert(orgPipe); err != nil {
		t.Fatalf("insert org pipe: %v", err)
	}

	// Test getOrgVars
	orgVars, err := getOrgVars(ctx, "pipe-1")
	if err != nil {
		t.Fatalf("getOrgVars error: %v", err)
	}
	if len(orgVars) != 1 {
		t.Fatalf("orgVars length = %d, want 1", len(orgVars))
	}
	if orgVars[0].Name != "ORG_VAR" {
		t.Errorf("orgVars[0].Name = %q, want %q", orgVars[0].Name, "ORG_VAR")
	}
	if orgVars[0].Value != "org-value" {
		t.Errorf("orgVars[0].Value = %q, want %q", orgVars[0].Value, "org-value")
	}
}

// TestConvertVar_IntegPipelineVars verifies that pipeline variables override org variables
func TestConvertVar_IntegPipelineVars(t *testing.T) {
	eng := setupPipelineTestDB(t)
	ctx := context.Background()

	// Setup org var
	orgVar := &model.TOrgVar{
		OrgId:  "org-1",
		Name:   "SHARED_VAR",
		Value:  "org-value",
		Public: 1,
	}
	if _, err := eng.Insert(orgVar); err != nil {
		t.Fatalf("insert org var: %v", err)
	}

	orgPipe := &model.TOrgPipe{
		OrgId:  "org-1",
		PipeId: "pipe-1",
	}
	if _, err := eng.Insert(orgPipe); err != nil {
		t.Fatalf("insert org pipe: %v", err)
	}

	// Setup pipeline var (should override org var)
	pipeVar := &model.TPipelineVar{
		PipelineId: "pipe-1",
		Name:       "SHARED_VAR",
		Value:      "pipe-value",
		Public:     1,
	}
	if _, err := eng.Insert(pipeVar); err != nil {
		t.Fatalf("insert pipeline var: %v", err)
	}

	// Test convertVar
	vars, err := convertVar(ctx, "pipe-1", nil)
	if err != nil {
		t.Fatalf("convertVar error: %v", err)
	}

	// Pipeline var should override org var
	sharedVar, ok := vars["SHARED_VAR"]
	if !ok {
		t.Fatal("SHARED_VAR not found in vars")
	}
	if sharedVar.Value != "pipe-value" {
		t.Errorf("sharedVar.Value = %q, want %q", sharedVar.Value, "pipe-value")
	}
}

// TestBatchBuildCounts_IntegWithData verifies batch build counts with actual data
func TestBatchBuildCounts_IntegWithData(t *testing.T) {
	eng := setupPipelineTestDB(t)
	ctx := context.Background()

	// Insert test builds
	builds := []*model.TBuild{
		{Id: "build-1", PipelineId: "pipe-1", PipelineVersionId: "pv-1"},
		{Id: "build-2", PipelineId: "pipe-1", PipelineVersionId: "pv-2"},
		{Id: "build-3", PipelineId: "pipe-1", PipelineVersionId: "pv-3"},
		{Id: "build-4", PipelineId: "pipe-2", PipelineVersionId: "pv-4"},
		{Id: "build-5", PipelineId: "pipe-2", PipelineVersionId: "pv-5"},
	}
	for _, b := range builds {
		if _, err := eng.Insert(b); err != nil {
			t.Fatalf("insert build: %v", err)
		}
	}

	// Test batch counts
	counts, err := BatchBuildCounts(ctx, []string{"pipe-1", "pipe-2", "pipe-3"})
	if err != nil {
		t.Fatalf("BatchBuildCounts error: %v", err)
	}

	if counts["pipe-1"] != 3 {
		t.Errorf("counts[pipe-1] = %d, want 3", counts["pipe-1"])
	}
	if counts["pipe-2"] != 2 {
		t.Errorf("counts[pipe-2] = %d, want 2", counts["pipe-2"])
	}
	if _, ok := counts["pipe-3"]; ok {
		t.Error("pipe-3 should not be in counts (no builds)")
	}
}

// TestBatchGetUsers_IntegWithData verifies batch user retrieval with actual data
func TestBatchGetUsers_IntegWithData(t *testing.T) {
	eng := setupPipelineTestDB(t)
	ctx := context.Background()

	// Insert test users (need to sync TUser table)
	if err := eng.Sync2(&model.TUser{}); err != nil {
		t.Fatalf("sync TUser: %v", err)
	}

	users := []*model.TUser{
		{Id: "user-1", Name: "alice", Aid: 1},
		{Id: "user-2", Name: "bob", Aid: 2},
		{Id: "user-3", Name: "charlie", Aid: 3},
	}
	for _, u := range users {
		if _, err := eng.Insert(u); err != nil {
			t.Fatalf("insert user: %v", err)
		}
	}

	// Test batch get
	userMap, err := BatchGetUsers(ctx, []string{"user-1", "user-3", "user-999"})
	if err != nil {
		t.Fatalf("BatchGetUsers error: %v", err)
	}

	if len(userMap) != 2 {
		t.Errorf("userMap length = %d, want 2", len(userMap))
	}
	if userMap["user-1"].Name != "alice" {
		t.Errorf("userMap[user-1].Name = %q, want %q", userMap["user-1"].Name, "alice")
	}
	if userMap["user-3"].Name != "charlie" {
		t.Errorf("userMap[user-3].Name = %q, want %q", userMap["user-3"].Name, "charlie")
	}
	if _, ok := userMap["user-999"]; ok {
		t.Error("user-999 should not be in map (doesn't exist)")
	}
}

// TestBatchLatestBuilds_IntegWithData verifies latest build retrieval with actual data
func TestBatchLatestBuilds_IntegWithData(t *testing.T) {
	eng := setupPipelineTestDB(t)
	ctx := context.Background()

	// Insert builds with different created times for 2 pipelines
	// Use RunBuild (table t_build) — build table is already synced
	builds := []*model.RunBuild{
		{Id: "lb-1", PipelineId: "pipe-1", PipelineVersionId: "pv-1", Created: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
		{Id: "lb-2", PipelineId: "pipe-1", PipelineVersionId: "pv-2", Created: time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)},
		{Id: "lb-3", PipelineId: "pipe-2", PipelineVersionId: "pv-3", Created: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)},
		{Id: "lb-4", PipelineId: "pipe-2", PipelineVersionId: "pv-4", Created: time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)},
	}
	for _, b := range builds {
		if _, err := eng.Insert(b); err != nil {
			t.Fatalf("insert build: %v", err)
		}
	}

	latestBuilds, err := BatchLatestBuilds(ctx, []string{"pipe-1", "pipe-2", "pipe-99"})
	if err != nil {
		t.Fatalf("BatchLatestBuilds error: %v", err)
	}

	if len(latestBuilds) != 2 {
		t.Errorf("latestBuilds length = %d, want 2", len(latestBuilds))
	}
	// pipe-1 should return lb-2 (latest)
	if lb, ok := latestBuilds["pipe-1"]; ok {
		if lb.Id != "lb-2" {
			t.Errorf("latestBuilds[pipe-1].Id = %q, want %q", lb.Id, "lb-2")
		}
	} else {
		t.Error("latestBuilds missing pipe-1")
	}
	// pipe-2 should return lb-4 (latest)
	if lb, ok := latestBuilds["pipe-2"]; ok {
		if lb.Id != "lb-4" {
			t.Errorf("latestBuilds[pipe-2].Id = %q, want %q", lb.Id, "lb-4")
		}
	} else {
		t.Error("latestBuilds missing pipe-2")
	}
	// pipe-99 has no builds
	if _, ok := latestBuilds["pipe-99"]; ok {
		t.Error("pipe-99 should not have a latest build")
	}
}

// TestBatchCountArtifactPackages_IntegWithData verifies artifact package counting
func TestBatchCountArtifactPackages_IntegWithData(t *testing.T) {
	eng := setupPipelineTestDB(t)
	ctx := context.Background()

	// Sync artifact package table
	if err := eng.Sync2(&model.TArtifactPackage{}); err != nil {
		t.Fatalf("sync TArtifactPackage: %v", err)
	}

	// Insert test packages
	packages := []*model.TArtifactPackage{
		{Id: "pkg-1", Aid: 1, RepoId: "repo-1", Deleted: 0},
		{Id: "pkg-2", Aid: 2, RepoId: "repo-1", Deleted: 0},
		{Id: "pkg-3", Aid: 3, RepoId: "repo-1", Deleted: 0},
		{Id: "pkg-4", Aid: 4, RepoId: "repo-2", Deleted: 0},
		{Id: "pkg-5", Aid: 5, RepoId: "repo-2", Deleted: 0},
	}
	for _, p := range packages {
		if _, err := eng.Insert(p); err != nil {
			t.Fatalf("insert package: %v", err)
		}
	}

	// Test batch count (should exclude deleted)
	counts, err := BatchCountArtifactPackages(ctx, []string{"repo-1", "repo-2"})
	if err != nil {
		t.Fatalf("BatchCountArtifactPackages error: %v", err)
	}

	if counts["repo-1"] != 3 {
		t.Errorf("counts[repo-1] = %d, want 3", counts["repo-1"])
	}
	if counts["repo-2"] != 2 {
		t.Errorf("counts[repo-2] = %d, want 2 (BatchCountArtifactPackages counts all, including deleted)", counts["repo-2"])
	}
}

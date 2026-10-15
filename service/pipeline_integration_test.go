package service

import (
	"context"
	"errors"
	"testing"

	"github.com/gokins/core/common"
	"github.com/gokins/gokins/comm"
	"github.com/gokins/gokins/model"
	_ "github.com/mattn/go-sqlite3"
	"xorm.io/xorm"
)

// setupPipelineTestDB creates an in-memory SQLite database with all tables
// needed by Run and ReBuild.
func setupPipelineTestDB(t *testing.T) *xorm.Engine {
	t.Helper()
	eng, err := xorm.NewEngine("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}
	oldDb := comm.Db
	comm.Db = eng
	t.Cleanup(func() {
		comm.Db = oldDb
		_ = eng.Close()
	})
	err = eng.Sync2(
		&model.TPipeline{},
		&model.TPipelineConf{},
		&model.TPipelineVersion{},
		&model.TPipelineVar{},
		&model.TBuild{},
		&model.TStage{},
		&model.TStep{},
		&model.TOrgVar{},
		&model.TOrgPipe{},
	)
	if err != nil {
		t.Fatalf("failed to sync schema: %v", err)
	}
	return eng
}

// minimalYaml returns a valid pipeline YAML with one stage and one step.
func minimalYaml() string {
	return `stages:
  - name: build
    displayName: Build
    steps:
      - name: compile
        displayName: Compile
        step: docker
        commands:
          - echo hello
`
}

// seedPipeline inserts a TPipeline + TPipelineConf pair for testing.
func seedPipeline(t *testing.T, eng *xorm.Engine, pipeId, yml string) {
	t.Helper()
	tp := &model.TPipeline{
		Id:          pipeId,
		Name:        "test-pipe",
		DisplayName: "Test Pipeline",
	}
	if _, err := eng.Insert(tp); err != nil {
		t.Fatalf("insert TPipeline: %v", err)
	}
	conf := &model.TPipelineConf{
		PipelineId: pipeId,
		Url:        "https://example.com/repo.git",
		Username:   "testuser",
		YmlContent: yml,
	}
	if _, err := eng.Insert(conf); err != nil {
		t.Fatalf("insert TPipelineConf: %v", err)
	}
}

// --- Run integration tests ---

func TestRun_Success(t *testing.T) {
	eng := setupPipelineTestDB(t)
	ctx := context.Background()

	seedPipeline(t, eng, "pipe-1", minimalYaml())

	tpv, rb, err := Run(ctx, "user-1", "pipe-1", "abc123", "push")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if tpv == nil {
		t.Fatal("Run returned nil TPipelineVersion")
	}
	if rb == nil {
		t.Fatal("Run returned nil Build")
	}
	if tpv.PipelineId != "pipe-1" {
		t.Errorf("tpv.PipelineId = %q, want %q", tpv.PipelineId, "pipe-1")
	}
	if tpv.Number != 1 {
		t.Errorf("tpv.Number = %d, want 1", tpv.Number)
	}
	if tpv.Events != "push" {
		t.Errorf("tpv.Events = %q, want %q", tpv.Events, "push")
	}
	if tpv.Sha != "abc123" {
		t.Errorf("tpv.Sha = %q, want %q", tpv.Sha, "abc123")
	}
	if rb.Status != common.BuildStatusPending {
		t.Errorf("rb.Status = %q, want %q", rb.Status, common.BuildStatusPending)
	}
	if rb.Repo == nil {
		t.Fatal("rb.Repo is nil")
	}
	if rb.Repo.CloneURL != "https://example.com/repo.git" {
		t.Errorf("rb.Repo.CloneURL = %q, want repo URL", rb.Repo.CloneURL)
	}
	if rb.Repo.Sha != "abc123" {
		t.Errorf("rb.Repo.Sha = %q, want %q", rb.Repo.Sha, "abc123")
	}

	// Verify DB records
	var build model.TBuild
	ok, err := eng.Where("pipeline_version_id = ?", tpv.Id).Get(&build)
	if err != nil {
		t.Fatalf("query build: %v", err)
	}
	if !ok {
		t.Fatal("build record not found in DB")
	}
	if build.Status != common.BuildStatusPending {
		t.Errorf("build.Status = %q, want %q", build.Status, common.BuildStatusPending)
	}

	var stages []model.TStage
	if err := eng.Where("build_id = ?", build.Id).Find(&stages); err != nil {
		t.Fatalf("query stages: %v", err)
	}
	if len(stages) != 1 {
		t.Fatalf("expected 1 stage, got %d", len(stages))
	}
	if stages[0].Name != "build" {
		t.Errorf("stage.Name = %q, want %q", stages[0].Name, "build")
	}

	var steps []model.TStep
	if err := eng.Where("stage_id = ?", stages[0].Id).Find(&steps); err != nil {
		t.Fatalf("query steps: %v", err)
	}
	if len(steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(steps))
	}
	if steps[0].Name != "compile" {
		t.Errorf("step.Name = %q, want %q", steps[0].Name, "compile")
	}
}

func TestRun_IncrementsNumber(t *testing.T) {
	eng := setupPipelineTestDB(t)
	ctx := context.Background()

	seedPipeline(t, eng, "pipe-inc", minimalYaml())

	tpv1, _, err := Run(ctx, "user-1", "pipe-inc", "sha1", "push")
	if err != nil {
		t.Fatalf("Run #1: %v", err)
	}
	if tpv1.Number != 1 {
		t.Errorf("first run Number = %d, want 1", tpv1.Number)
	}

	tpv2, _, err := Run(ctx, "user-1", "pipe-inc", "sha2", "push")
	if err != nil {
		t.Fatalf("Run #2: %v", err)
	}
	if tpv2.Number != 2 {
		t.Errorf("second run Number = %d, want 2", tpv2.Number)
	}

	tpv3, _, err := Run(ctx, "user-1", "pipe-inc", "sha3", "pr")
	if err != nil {
		t.Fatalf("Run #3: %v", err)
	}
	if tpv3.Number != 3 {
		t.Errorf("third run Number = %d, want 3", tpv3.Number)
	}
}

func TestRun_PipelineNotFound(t *testing.T) {
	setupPipelineTestDB(t)
	ctx := context.Background()

	_, _, err := Run(ctx, "user-1", "nonexistent", "sha", "push")
	if err == nil {
		t.Fatal("expected error for nonexistent pipeline")
	}
	if !errors.Is(err, ErrPipelineNotFound) {
		t.Errorf("error = %v, want ErrPipelineNotFound", err)
	}
}

func TestRun_EmptyYmlContent(t *testing.T) {
	eng := setupPipelineTestDB(t)
	ctx := context.Background()

	seedPipeline(t, eng, "pipe-empty", "")

	_, _, err := Run(ctx, "user-1", "pipe-empty", "sha", "push")
	if err == nil {
		t.Fatal("expected error for empty YML content")
	}
	if !errors.Is(err, ErrPipelineYmlEmpty) {
		t.Errorf("error = %v, want ErrPipelineYmlEmpty", err)
	}
}

func TestRun_InvalidYaml(t *testing.T) {
	eng := setupPipelineTestDB(t)
	ctx := context.Background()

	seedPipeline(t, eng, "pipe-bad", "this is not valid yaml: [[[")

	_, _, err := Run(ctx, "user-1", "pipe-bad", "sha", "push")
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestRun_EmptyStagesFailsCheck(t *testing.T) {
	eng := setupPipelineTestDB(t)
	ctx := context.Background()

	// Valid YAML but no stages defined
	seedPipeline(t, eng, "pipe-nostages", "stages: []\n")

	_, _, err := Run(ctx, "user-1", "pipe-nostages", "sha", "push")
	if err == nil {
		t.Fatal("expected error for empty stages")
	}
}

// --- ReBuild integration tests ---

func TestReBuild_Success(t *testing.T) {
	eng := setupPipelineTestDB(t)
	ctx := context.Background()

	seedPipeline(t, eng, "pipe-re", minimalYaml())

	// First run to create a version
	tpv1, _, err := Run(ctx, "user-1", "pipe-re", "sha1", "push")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	// ReBuild using the first version
	tpv2, rb2, err := ReBuild(ctx, "user-2", tpv1)
	if err != nil {
		t.Fatalf("ReBuild: %v", err)
	}
	if tpv2 == nil {
		t.Fatal("ReBuild returned nil TPipelineVersion")
	}
	if rb2 == nil {
		t.Fatal("ReBuild returned nil Build")
	}
	if tpv2.Number != 2 {
		t.Errorf("ReBuild Number = %d, want 2", tpv2.Number)
	}
	if tpv2.Events != "rebuild" {
		t.Errorf("ReBuild Events = %q, want %q", tpv2.Events, "rebuild")
	}
	if tpv2.Sha != "sha1" {
		t.Errorf("ReBuild Sha = %q, want %q (preserved from original)", tpv2.Sha, "sha1")
	}
	if tpv2.Uid != "user-2" {
		t.Errorf("ReBuild Uid = %q, want %q", tpv2.Uid, "user-2")
	}
}

func TestReBuild_PipelineDeleted(t *testing.T) {
	eng := setupPipelineTestDB(t)
	ctx := context.Background()

	seedPipeline(t, eng, "pipe-del", minimalYaml())

	tpv1, _, err := Run(ctx, "user-1", "pipe-del", "sha1", "push")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	// Soft-delete the pipeline
	_, err = eng.Where("id = ?", "pipe-del").Cols("deleted").Update(&model.TPipeline{Deleted: 1})
	if err != nil {
		t.Fatalf("update deleted: %v", err)
	}

	_, _, err = ReBuild(ctx, "user-1", tpv1)
	if err == nil {
		t.Fatal("expected error for deleted pipeline")
	}
	if !errors.Is(err, ErrPipelineNotFound) {
		t.Errorf("error = %v, want ErrPipelineNotFound", err)
	}
}

func TestReBuild_EmptyContent(t *testing.T) {
	eng := setupPipelineTestDB(t)
	ctx := context.Background()

	seedPipeline(t, eng, "pipe-re-empty", minimalYaml())

	tpv1, _, err := Run(ctx, "user-1", "pipe-re-empty", "sha1", "push")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	// Clear content to simulate empty version
	tpv1.Content = ""

	_, _, err = ReBuild(ctx, "user-1", tpv1)
	if err == nil {
		t.Fatal("expected error for empty content in ReBuild")
	}
	if !errors.Is(err, ErrPipelineYmlEmpty) {
		t.Errorf("error = %v, want ErrPipelineYmlEmpty", err)
	}
}

// --- Run with variables ---

func TestRun_WithPipelineVars(t *testing.T) {
	eng := setupPipelineTestDB(t)
	ctx := context.Background()

	seedPipeline(t, eng, "pipe-vars", minimalYaml())

	// Add pipeline variables
	vars := []model.TPipelineVar{
		{PipelineId: "pipe-vars", Name: "APP_ENV", Value: "production", Public: 0},
		{PipelineId: "pipe-vars", Name: "SECRET_KEY", Value: "s3cret", Public: 1},
	}
	for _, v := range vars {
		if _, err := eng.Insert(&v); err != nil {
			t.Fatalf("insert var: %v", err)
		}
	}

	_, rb, err := Run(ctx, "user-1", "pipe-vars", "sha1", "push")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	// Check that variables are in the build
	if rb.Vars == nil {
		t.Fatal("rb.Vars is nil")
	}
	if _, ok := rb.Vars["APP_ENV"]; !ok {
		t.Error("APP_ENV not found in build vars")
	}
	if _, ok := rb.Vars["SECRET_KEY"]; !ok {
		t.Error("SECRET_KEY not found in build vars")
	}
	// GOKINSV_GIT_SHA should always be injected
	if _, ok := rb.Vars["GOKINSV_GIT_SHA"]; !ok {
		t.Error("GOKINSV_GIT_SHA not found in build vars")
	}
}

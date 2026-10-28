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

// setupPipelineTestDB creates an in-memory SQLite database with all tables needed for pipeline tests.
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
		&model.TUser{},
		&model.TUserInfo{},
		&model.TOrg{},
		&model.TUserOrg{},
		&model.TPipeline{},
		&model.TPipelineConf{},
		&model.TPipelineVersion{},
		&model.TPipelineVar{},
		&model.TOrgVar{},
		&model.TOrgPipe{},
		&model.TBuild{},
		&model.TStage{},
		&model.TStep{},
		&model.TCmdLine{},
	)
	if err != nil {
		t.Fatalf("failed to sync database schema: %v", err)
	}

	return eng
}

func TestRun_PipelineNotFound(t *testing.T) {
	setupPipelineTestDB(t)
	ctx := context.Background()

	_, _, err := Run(ctx, "user-1", "nonexistent-pipeline", "abc123", "push")
	if !errors.Is(err, ErrPipelineNotFound) {
		t.Errorf("Run with nonexistent pipeline: got %v, want %v", err, ErrPipelineNotFound)
	}
}

func TestRun_EmptyYmlContent(t *testing.T) {
	eng := setupPipelineTestDB(t)
	ctx := context.Background()

	// Create pipeline with empty YAML
	pipe := &model.TPipeline{
		Id:          "pipe-1",
		Name:        "test-pipeline",
		DisplayName: "Test Pipeline",
	}
	if _, err := eng.Insert(pipe); err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	conf := &model.TPipelineConf{
		PipelineId: "pipe-1",
		YmlContent: "", // Empty YAML
	}
	if _, err := eng.Insert(conf); err != nil {
		t.Fatalf("insert pipeline conf: %v", err)
	}

	_, _, err := Run(ctx, "user-1", "pipe-1", "abc123", "push")
	if !errors.Is(err, ErrPipelineYmlEmpty) {
		t.Errorf("Run with empty YAML: got %v, want %v", err, ErrPipelineYmlEmpty)
	}
}

func TestRun_InvalidYaml(t *testing.T) {
	eng := setupPipelineTestDB(t)
	ctx := context.Background()

	pipe := &model.TPipeline{
		Id:          "pipe-1",
		Name:        "test-pipeline",
		DisplayName: "Test Pipeline",
	}
	if _, err := eng.Insert(pipe); err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	conf := &model.TPipelineConf{
		PipelineId: "pipe-1",
		YmlContent: "invalid: yaml: content: [[[",
	}
	if _, err := eng.Insert(conf); err != nil {
		t.Fatalf("insert pipeline conf: %v", err)
	}

	_, _, err := Run(ctx, "user-1", "pipe-1", "abc123", "push")
	if err == nil {
		t.Error("Run with invalid YAML should return error")
	}
}

func TestRun_Success(t *testing.T) {
	eng := setupPipelineTestDB(t)
	ctx := context.Background()

	// Create pipeline
	pipe := &model.TPipeline{
		Id:          "pipe-1",
		Name:        "test-pipeline",
		DisplayName: "Test Pipeline",
	}
	if _, err := eng.Insert(pipe); err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	// Create valid pipeline config
	yml := `stages:
  - name: build
    displayName: Build Stage
    steps:
      - step: shell
        name: compile
        displayName: Compile Code
        commands:
          - go build ./...
`
	conf := &model.TPipelineConf{
		PipelineId: "pipe-1",
		YmlContent: yml,
		Username:   "testuser",
		Url:        "https://github.com/test/repo",
	}
	if _, err := eng.Insert(conf); err != nil {
		t.Fatalf("insert pipeline conf: %v", err)
	}

	// Create user
	user := &model.TUser{Id: "user-1", Aid: 1, Name: "admin", Active: 1}
	if _, err := eng.Insert(user); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	tpv, rb, err := Run(ctx, "user-1", "pipe-1", "abc123def", "push")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// Verify pipeline version was created
	if tpv == nil {
		t.Fatal("Run returned nil pipeline version")
	}
	if tpv.PipelineId != "pipe-1" {
		t.Errorf("tpv.PipelineId = %s, want pipe-1", tpv.PipelineId)
	}
	if tpv.Sha != "abc123def" {
		t.Errorf("tpv.Sha = %s, want abc123def", tpv.Sha)
	}
	if tpv.Events != "push" {
		t.Errorf("tpv.Events = %s, want push", tpv.Events)
	}
	if tpv.Number != 1 {
		t.Errorf("tpv.Number = %d, want 1", tpv.Number)
	}

	// Verify runtime build was created
	if rb == nil {
		t.Fatal("Run returned nil runtime build")
	}
	if rb.PipelineId != "pipe-1" {
		t.Errorf("rb.PipelineId = %s, want pipe-1", rb.PipelineId)
	}
	if rb.Repo == nil {
		t.Fatal("rb.Repo is nil")
	}
	if rb.Repo.CloneURL != "https://github.com/test/repo" {
		t.Errorf("rb.Repo.CloneURL = %s, want https://github.com/test/repo", rb.Repo.CloneURL)
	}

	// Verify build was inserted in database
	var build model.TBuild
	has, err := eng.Where("pipeline_version_id = ?", tpv.Id).Get(&build)
	if err != nil {
		t.Fatalf("query build: %v", err)
	}
	if !has {
		t.Error("build not found in database")
	}
	if build.PipelineId != "pipe-1" {
		t.Errorf("build.PipelineId = %s, want pipe-1", build.PipelineId)
	}
	if build.Status != "pending" {
		t.Errorf("build.Status = %s, want pending", build.Status)
	}
}

func TestRun_WithVariables(t *testing.T) {
	eng := setupPipelineTestDB(t)
	ctx := context.Background()

	pipe := &model.TPipeline{
		Id:   "pipe-1",
		Name: "test-pipeline",
	}
	if _, err := eng.Insert(pipe); err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	// Add pipeline variables
	pvar := &model.TPipelineVar{
		PipelineId: "pipe-1",
		Name:       "VERSION",
		Value:      "1.0.0",
		Public:     1,
	}
	if _, err := eng.Insert(pvar); err != nil {
		t.Fatalf("insert pipeline var: %v", err)
	}

	yml := `vars:
  APP_NAME: myapp
stages:
  - name: build
    steps:
      - step: shell
        name: compile
        commands:
          - echo ${{VERSION}}
`
	conf := &model.TPipelineConf{
		PipelineId: "pipe-1",
		YmlContent: yml,
	}
	if _, err := eng.Insert(conf); err != nil {
		t.Fatalf("insert pipeline conf: %v", err)
	}

	tpv, rb, err := Run(ctx, "user-1", "pipe-1", "sha123", "push")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// Verify variables were set
	if rb == nil || rb.Vars == nil {
		t.Fatal("runtime build or vars is nil")
	}
	if _, ok := rb.Vars["VERSION"]; !ok {
		t.Error("VERSION variable not found in runtime vars")
	}
	if _, ok := rb.Vars["APP_NAME"]; !ok {
		t.Error("APP_NAME variable not found in runtime vars")
	}
	if _, ok := rb.Vars["GOKINSV_GIT_SHA"]; !ok {
		t.Error("GOKINSV_GIT_SHA variable not found in runtime vars")
	}

	// Verify pipeline version content
	if tpv.Content != yml {
		t.Errorf("tpv.Content does not match original YAML")
	}
}

func TestReBuild_PipelineNotFound(t *testing.T) {
	setupPipelineTestDB(t)
	ctx := context.Background()

	tvp := &model.TPipelineVersion{
		PipelineId: "nonexistent",
		Content:    "stages: []",
	}

	_, _, err := ReBuild(ctx, "user-1", tvp)
	if !errors.Is(err, ErrPipelineNotFound) {
		t.Errorf("ReBuild with nonexistent pipeline: got %v, want %v", err, ErrPipelineNotFound)
	}
}

func TestReBuild_EmptyContent(t *testing.T) {
	eng := setupPipelineTestDB(t)
	ctx := context.Background()

	pipe := &model.TPipeline{
		Id:   "pipe-1",
		Name: "test-pipeline",
	}
	if _, err := eng.Insert(pipe); err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	conf := &model.TPipelineConf{
		PipelineId: "pipe-1",
		YmlContent: "stages: []",
	}
	if _, err := eng.Insert(conf); err != nil {
		t.Fatalf("insert pipeline conf: %v", err)
	}

	tvp := &model.TPipelineVersion{
		PipelineId: "pipe-1",
		Content:    "", // Empty content
	}

	_, _, err := ReBuild(ctx, "user-1", tvp)
	if !errors.Is(err, ErrPipelineYmlEmpty) {
		t.Errorf("ReBuild with empty content: got %v, want %v", err, ErrPipelineYmlEmpty)
	}
}

func TestReBuild_Success(t *testing.T) {
	eng := setupPipelineTestDB(t)
	ctx := context.Background()

	pipe := &model.TPipeline{
		Id:          "pipe-1",
		Name:        "test-pipeline",
		DisplayName: "Test Pipeline",
	}
	if _, err := eng.Insert(pipe); err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	originalYml := `stages:
  - name: build
    steps:
      - step: shell
        name: compile
        commands:
          - go build
`
	conf := &model.TPipelineConf{
		PipelineId: "pipe-1",
		YmlContent: originalYml,
		Url:        "https://github.com/test/repo",
	}
	if _, err := eng.Insert(conf); err != nil {
		t.Fatalf("insert pipeline conf: %v", err)
	}

	// Create initial pipeline version
	initialTpv := &model.TPipelineVersion{
		Id:         "v1",
		PipelineId: "pipe-1",
		Content:    originalYml,
		Sha:        "initial-sha",
		Number:     1,
		Created:    time.Now(),
	}
	if _, err := eng.Insert(initialTpv); err != nil {
		t.Fatalf("insert initial pipeline version: %v", err)
	}

	// Rebuild with new content
	newYml := `stages:
  - name: build
    steps:
      - step: shell
        name: compile
        commands:
          - go build -v
`
	tvp := &model.TPipelineVersion{
		PipelineId: "pipe-1",
		Content:    newYml,
		Sha:        "new-sha-456",
	}

	newTpv, rb, err := ReBuild(ctx, "user-1", tvp)
	if err != nil {
		t.Fatalf("ReBuild failed: %v", err)
	}

	// Verify new pipeline version was created
	if newTpv == nil {
		t.Fatal("ReBuild returned nil pipeline version")
	}
	if newTpv.Id == initialTpv.Id {
		t.Error("ReBuild should create a new pipeline version, not reuse the old one")
	}
	if newTpv.Number != 2 {
		t.Errorf("newTpv.Number = %d, want 2", newTpv.Number)
	}
	if newTpv.Sha != "new-sha-456" {
		t.Errorf("newTpv.Sha = %s, want new-sha-456", newTpv.Sha)
	}
	if newTpv.Content != newYml {
		t.Errorf("newTpv.Content should be the new YAML content")
	}
	if newTpv.Events != "rebuild" {
		t.Errorf("newTpv.Events = %s, want rebuild", newTpv.Events)
	}

	// Verify runtime build
	if rb == nil {
		t.Fatal("ReBuild returned nil runtime build")
	}
	if rb.PipelineVersionId != newTpv.Id {
		t.Errorf("rb.PipelineVersionId = %s, want %s", rb.PipelineVersionId, newTpv.Id)
	}
}

func TestRun_MultipleStagesAndSteps(t *testing.T) {
	eng := setupPipelineTestDB(t)
	ctx := context.Background()

	pipe := &model.TPipeline{
		Id:   "pipe-1",
		Name: "test-pipeline",
	}
	if _, err := eng.Insert(pipe); err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	yml := `stages:
  - name: build
    displayName: Build
    steps:
      - step: shell
        name: compile
        displayName: Compile
        commands:
          - go build
      - step: shell
        name: test
        displayName: Test
        commands:
          - go test
  - name: deploy
    displayName: Deploy
    steps:
      - step: shell
        name: deploy-prod
        displayName: Deploy to Production
        commands:
          - ./deploy.sh
`
	conf := &model.TPipelineConf{
		PipelineId: "pipe-1",
		YmlContent: yml,
	}
	if _, err := eng.Insert(conf); err != nil {
		t.Fatalf("insert pipeline conf: %v", err)
	}

	tpv, rb, err := Run(ctx, "user-1", "pipe-1", "sha789", "push")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// Verify stages in runtime build
	if len(rb.Stages) != 2 {
		t.Fatalf("expected 2 stages, got %d", len(rb.Stages))
	}

	// Verify first stage
	if rb.Stages[0].Name != "build" {
		t.Errorf("stage[0].Name = %s, want build", rb.Stages[0].Name)
	}
	if len(rb.Stages[0].Steps) != 2 {
		t.Errorf("stage[0] should have 2 steps, got %d", len(rb.Stages[0].Steps))
	}

	// Verify second stage
	if rb.Stages[1].Name != "deploy" {
		t.Errorf("stage[1].Name = %s, want deploy", rb.Stages[1].Name)
	}
	if len(rb.Stages[1].Steps) != 1 {
		t.Errorf("stage[1] should have 1 step, got %d", len(rb.Stages[1].Steps))
	}

	// Verify database inserts
	var stages []model.TStage
	if err := eng.Where("pipeline_version_id = ?", tpv.Id).Find(&stages); err != nil {
		t.Fatalf("query stages: %v", err)
	}
	if len(stages) != 2 {
		t.Errorf("expected 2 stages in database, got %d", len(stages))
	}

	var steps []model.TStep
	if err := eng.Where("pipeline_version_id = ?", tpv.Id).Find(&steps); err != nil {
		t.Fatalf("query steps: %v", err)
	}
	if len(steps) != 3 {
		t.Errorf("expected 3 steps in database, got %d", len(steps))
	}
}

func TestRun_IncrementalVersionNumbers(t *testing.T) {
	eng := setupPipelineTestDB(t)
	ctx := context.Background()

	pipe := &model.TPipeline{
		Id:   "pipe-1",
		Name: "test-pipeline",
	}
	if _, err := eng.Insert(pipe); err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	yml := `stages:
  - name: build
    steps:
      - step: shell
        name: compile
        commands:
          - echo build
`
	conf := &model.TPipelineConf{
		PipelineId: "pipe-1",
		YmlContent: yml,
	}
	if _, err := eng.Insert(conf); err != nil {
		t.Fatalf("insert pipeline conf: %v", err)
	}

	// Run pipeline multiple times
	for i := int64(1); i <= 3; i++ {
		tpv, _, err := Run(ctx, "user-1", "pipe-1", "sha"+string(rune('0'+i)), "push")
		if err != nil {
			t.Fatalf("Run #%d failed: %v", i, err)
		}
		if tpv.Number != i {
			t.Errorf("Run #%d: tpv.Number = %d, want %d", i, tpv.Number, i)
		}
	}
}

func TestPreBuild_DisabledSteps(t *testing.T) {
	eng := setupPipelineTestDB(t)
	ctx := context.Background()

	pipe := &model.TPipeline{
		Id:   "pipe-1",
		Name: "test-pipeline",
	}
	if _, err := eng.Insert(pipe); err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	yml := `stages:
  - name: build
    steps:
      - step: shell
        name: step1
        commands:
          - echo step1
      - step: shell
        name: step2
        disable: true
        commands:
          - echo step2
      - step: shell
        name: step3
        commands:
          - echo step3
`
	conf := &model.TPipelineConf{
		PipelineId: "pipe-1",
		YmlContent: yml,
	}
	if _, err := eng.Insert(conf); err != nil {
		t.Fatalf("insert pipeline conf: %v", err)
	}

	tpv, rb, err := Run(ctx, "user-1", "pipe-1", "sha123", "push")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// Verify only non-disabled steps are in runtime build
	if len(rb.Stages) != 1 {
		t.Fatalf("expected 1 stage, got %d", len(rb.Stages))
	}
	if len(rb.Stages[0].Steps) != 2 {
		t.Errorf("expected 2 steps (step2 disabled), got %d", len(rb.Stages[0].Steps))
	}

	// Verify database has only non-disabled steps
	var steps []model.TStep
	if err := eng.Where("pipeline_version_id = ?", tpv.Id).Find(&steps); err != nil {
		t.Fatalf("query steps: %v", err)
	}
	if len(steps) != 2 {
		t.Errorf("expected 2 steps in database (step2 disabled), got %d", len(steps))
	}
}

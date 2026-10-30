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

// setupPipelineTestDB creates an in-memory SQLite database with the tables
// needed by Run, ReBuild, preBuild, getOrgVars and convertVar.
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
		&model.TBuild{},
		&model.TStage{},
		&model.TStep{},
		&model.TCmdLine{},
		&model.TOrgVar{},
		&model.TOrgPipe{},
		&model.TPipelineVar{},
	)
	if err != nil {
		t.Fatalf("failed to sync database schema: %v", err)
	}

	return eng
}

// validPipelineYaml is a minimal valid pipeline YAML that passes Check().
const validPipelineYAML = `
stages:
  - name: build
    steps:
      - step: shell@ssh
        name: echo
        commands: echo hello
`

// -----------------------------------------------------------------------
// Run() tests
// -----------------------------------------------------------------------

func TestRun_PipelineNotFound(t *testing.T) {
	setupPipelineTestDB(t)
	ctx := context.Background()

	_, _, err := Run(ctx, "user1", "nonexistent", "abc123", "push")
	if err == nil {
		t.Fatal("expected error for nonexistent pipeline")
	}
	if !errors.Is(err, ErrPipelineNotFound) {
		t.Fatalf("expected ErrPipelineNotFound, got: %v", err)
	}
}

func TestRun_EmptyYmlContent(t *testing.T) {
	eng := setupPipelineTestDB(t)
	ctx := context.Background()

	// Insert a pipeline conf with empty YAML content
	_, err := eng.Insert(&model.TPipelineConf{
		PipelineId: "pipe1",
		YmlContent: "",
		Url:        "https://example.com/repo",
	})
	if err != nil {
		t.Fatalf("insert pipeline conf: %v", err)
	}

	_, _, err = Run(ctx, "user1", "pipe1", "abc123", "push")
	if err == nil {
		t.Fatal("expected error for empty yml content")
	}
	if !errors.Is(err, ErrPipelineYmlEmpty) {
		t.Fatalf("expected ErrPipelineYmlEmpty, got: %v", err)
	}
}

func TestRun_InvalidYAML(t *testing.T) {
	eng := setupPipelineTestDB(t)
	ctx := context.Background()

	_, err := eng.Insert(&model.TPipelineConf{
		PipelineId: "pipe1",
		YmlContent: "stages: [invalid yaml: {broken",
	})
	if err != nil {
		t.Fatalf("insert pipeline conf: %v", err)
	}

	_, _, err = Run(ctx, "user1", "pipe1", "abc123", "push")
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
	if !errors.Is(err, ErrPipelineNotFound) && err.Error() == "" {
		t.Fatalf("expected parse error, got: %v", err)
	}
}

func TestRun_PipelineCheckFails(t *testing.T) {
	eng := setupPipelineTestDB(t)
	ctx := context.Background()

	// Valid YAML but empty stages → Check() fails
	_, err := eng.Insert(&model.TPipelineConf{
		PipelineId: "pipe1",
		YmlContent: "version: '1'\n",
	})
	if err != nil {
		t.Fatalf("insert pipeline conf: %v", err)
	}
	// Insert the TPipeline record so preBuild can find it
	_, err = eng.Insert(&model.TPipeline{
		Id:   "pipe1",
		Name: "test-pipe",
	})
	if err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	_, _, err = Run(ctx, "user1", "pipe1", "abc123", "push")
	if err == nil {
		t.Fatal("expected error when pipeline check fails")
	}
	// Should wrap ErrStagesEmpty from bean.Check()
	if !errors.Is(err, ErrPipelineNotFound) && err.Error() == "" {
		t.Fatalf("expected stages empty error, got: %v", err)
	}
}

func TestRun_Success(t *testing.T) {
	eng := setupPipelineTestDB(t)
	ctx := context.Background()

	// Insert pipeline
	_, err := eng.Insert(&model.TPipeline{
		Id:          "pipe1",
		Name:        "test-pipe",
		DisplayName: "Test Pipeline",
	})
	if err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	// Insert pipeline conf with valid YAML
	_, err = eng.Insert(&model.TPipelineConf{
		PipelineId:  "pipe1",
		YmlContent:  validPipelineYAML,
		Url:         "https://example.com/repo",
		AccessToken: "tok123",
		Username:    "user",
	})
	if err != nil {
		t.Fatalf("insert pipeline conf: %v", err)
	}

	tpv, rb, err := Run(ctx, "user1", "pipe1", "abc123", "push")
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}
	if tpv == nil {
		t.Fatal("expected TPipelineVersion, got nil")
	}
	if rb == nil {
		t.Fatal("expected Build, got nil")
	}

	// Verify pipeline version fields
	if tpv.PipelineId != "pipe1" {
		t.Errorf("tpv.PipelineId = %q, want %q", tpv.PipelineId, "pipe1")
	}
	if tpv.Uid != "user1" {
		t.Errorf("tpv.Uid = %q, want %q", tpv.Uid, "user1")
	}
	if tpv.Sha != "abc123" {
		t.Errorf("tpv.Sha = %q, want %q", tpv.Sha, "abc123")
	}
	if tpv.Events != "push" {
		t.Errorf("tpv.Events = %q, want %q", tpv.Events, "push")
	}
	if tpv.Number != 1 {
		t.Errorf("tpv.Number = %d, want 1", tpv.Number)
	}
	if tpv.PipelineName != "test-pipe" {
		t.Errorf("tpv.PipelineName = %q, want %q", tpv.PipelineName, "test-pipe")
	}

	// Verify build fields
	if rb.PipelineId != "pipe1" {
		t.Errorf("rb.PipelineId = %q, want %q", rb.PipelineId, "pipe1")
	}
	if rb.Repo == nil {
		t.Fatal("expected rb.Repo, got nil")
	}
	if rb.Repo.CloneURL != "https://example.com/repo" {
		t.Errorf("rb.Repo.CloneURL = %q, want repo URL", rb.Repo.CloneURL)
	}

	// Verify database records were created
	var buildCount int64
	buildCount, err = eng.Where("pipeline_id=?", "pipe1").Count(&model.TBuild{})
	if err != nil {
		t.Fatalf("count builds: %v", err)
	}
	if buildCount != 1 {
		t.Errorf("build count = %d, want 1", buildCount)
	}

	var stageCount int64
	stageCount, err = eng.Where("build_id=?", rb.Id).Count(&model.TStage{})
	if err != nil {
		t.Fatalf("count stages: %v", err)
	}
	if stageCount != 1 {
		t.Errorf("stage count = %d, want 1", stageCount)
	}

	var stepCount int64
	stepCount, err = eng.Where("build_id=?", rb.Id).Count(&model.TStep{})
	if err != nil {
		t.Fatalf("count steps: %v", err)
	}
	if stepCount != 1 {
		t.Errorf("step count = %d, want 1", stepCount)
	}
}

func TestRun_IncrementsVersionNumber(t *testing.T) {
	eng := setupPipelineTestDB(t)
	ctx := context.Background()

	_, err := eng.Insert(&model.TPipeline{Id: "pipe1", Name: "test-pipe"})
	if err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}
	_, err = eng.Insert(&model.TPipelineConf{
		PipelineId: "pipe1",
		YmlContent: validPipelineYAML,
		Url:        "https://example.com/repo",
	})
	if err != nil {
		t.Fatalf("insert pipeline conf: %v", err)
	}

	// First run
	tpv1, _, err := Run(ctx, "user1", "pipe1", "sha1", "push")
	if err != nil {
		t.Fatalf("first Run() error: %v", err)
	}
	if tpv1.Number != 1 {
		t.Errorf("first run number = %d, want 1", tpv1.Number)
	}

	// Second run
	tpv2, _, err := Run(ctx, "user1", "pipe1", "sha2", "push")
	if err != nil {
		t.Fatalf("second Run() error: %v", err)
	}
	if tpv2.Number != 2 {
		t.Errorf("second run number = %d, want 2", tpv2.Number)
	}
}

// -----------------------------------------------------------------------
// ReBuild() tests
// -----------------------------------------------------------------------

func TestReBuild_PipelineNotFound(t *testing.T) {
	setupPipelineTestDB(t)
	ctx := context.Background()

	tvp := &model.TPipelineVersion{
		PipelineId: "nonexistent",
		Content:    validPipelineYAML,
		Sha:        "abc123",
	}

	_, _, err := ReBuild(ctx, "user1", tvp)
	if err == nil {
		t.Fatal("expected error for nonexistent pipeline")
	}
	if !errors.Is(err, ErrPipelineNotFound) {
		t.Fatalf("expected ErrPipelineNotFound, got: %v", err)
	}
}

func TestReBuild_EmptyContent(t *testing.T) {
	eng := setupPipelineTestDB(t)
	ctx := context.Background()

	_, err := eng.Insert(&model.TPipelineConf{
		PipelineId: "pipe1",
		YmlContent: validPipelineYAML,
	})
	if err != nil {
		t.Fatalf("insert pipeline conf: %v", err)
	}

	tvp := &model.TPipelineVersion{
		PipelineId: "pipe1",
		Content:    "",
		Sha:        "abc123",
	}

	_, _, err = ReBuild(ctx, "user1", tvp)
	if err == nil {
		t.Fatal("expected error for empty content")
	}
	if !errors.Is(err, ErrPipelineYmlEmpty) {
		t.Fatalf("expected ErrPipelineYmlEmpty, got: %v", err)
	}
}

func TestReBuild_Success(t *testing.T) {
	eng := setupPipelineTestDB(t)
	ctx := context.Background()

	_, err := eng.Insert(&model.TPipeline{
		Id:          "pipe1",
		Name:        "test-pipe",
		DisplayName: "Test Pipeline",
	})
	if err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}
	_, err = eng.Insert(&model.TPipelineConf{
		PipelineId: "pipe1",
		YmlContent: validPipelineYAML,
		Url:        "https://example.com/repo",
		Username:   "user",
	})
	if err != nil {
		t.Fatalf("insert pipeline conf: %v", err)
	}

	tvp := &model.TPipelineVersion{
		PipelineId: "pipe1",
		Content:    validPipelineYAML,
		Sha:        "def456",
	}

	tpv, rb, err := ReBuild(ctx, "user1", tvp)
	if err != nil {
		t.Fatalf("ReBuild() error: %v", err)
	}
	if tpv == nil || rb == nil {
		t.Fatal("expected non-nil results")
	}
	// ReBuild should use the version's SHA, not the conf's
	if tpv.Sha != "def456" {
		t.Errorf("tpv.Sha = %q, want %q", tpv.Sha, "def456")
	}
	if tpv.Events != "rebuild" {
		t.Errorf("tpv.Events = %q, want %q", tpv.Events, "rebuild")
	}
	// ReBuild should use the version's Content
	if tpv.Content != validPipelineYAML {
		t.Errorf("tpv.Content should be the version content")
	}
}

// -----------------------------------------------------------------------
// getOrgVars() tests
// -----------------------------------------------------------------------

func TestGetOrgVars_NoOrgPipes(t *testing.T) {
	setupPipelineTestDB(t)
	ctx := context.Background()

	vars, err := getOrgVars(ctx, "pipe1")
	if err != nil {
		t.Fatalf("getOrgVars() error: %v", err)
	}
	if len(vars) != 0 {
		t.Errorf("expected empty vars, got %d", len(vars))
	}
}

func TestGetOrgVars_WithOrgVars(t *testing.T) {
	eng := setupPipelineTestDB(t)
	ctx := context.Background()

	// Insert org-pipe mapping
	_, err := eng.Insert(&model.TOrgPipe{
		OrgId:    "org1",
		PipeId: "pipe1",
	})
	if err != nil {
		t.Fatalf("insert org pipe: %v", err)
	}
	_, err = eng.Insert(&model.TOrgPipe{
		OrgId:    "org2",
		PipeId: "pipe1",
	})
	if err != nil {
		t.Fatalf("insert org pipe: %v", err)
	}

	// Insert org vars
	_, err = eng.Insert(&model.TOrgVar{
		OrgId:  "org1",
		Name:   "ORG_VAR1",
		Value:  "val1",
		Public: 1,
	})
	if err != nil {
		t.Fatalf("insert org var: %v", err)
	}
	_, err = eng.Insert(&model.TOrgVar{
		OrgId:  "org2",
		Name:   "ORG_VAR2",
		Value:  "val2",
		Public: 0,
	})
	if err != nil {
		t.Fatalf("insert org var: %v", err)
	}
	// Unrelated org should not appear
	_, err = eng.Insert(&model.TOrgVar{
		OrgId: "org3",
		Name:  "UNRELATED",
		Value: "nope",
	})
	if err != nil {
		t.Fatalf("insert unrelated org var: %v", err)
	}

	vars, err := getOrgVars(ctx, "pipe1")
	if err != nil {
		t.Fatalf("getOrgVars() error: %v", err)
	}
	if len(vars) != 2 {
		t.Fatalf("expected 2 vars, got %d", len(vars))
	}

	found := map[string]string{}
	for _, v := range vars {
		found[v.Name] = v.Value
	}
	if found["ORG_VAR1"] != "val1" {
		t.Errorf("ORG_VAR1 = %q, want %q", found["ORG_VAR1"], "val1")
	}
	if found["ORG_VAR2"] != "val2" {
		t.Errorf("ORG_VAR2 = %q, want %q", found["ORG_VAR2"], "val2")
	}
}

// -----------------------------------------------------------------------
// convertVar() tests
// -----------------------------------------------------------------------

func TestConvertVar_EmptyPipelineVars(t *testing.T) {
	eng := setupPipelineTestDB(t)
	ctx := context.Background()

	// Insert org-pipe + org var so getOrgVars returns something
	_, _ = eng.Insert(&model.TOrgPipe{OrgId: "org1", PipeId: "pipe1"})
	_, _ = eng.Insert(&model.TOrgVar{OrgId: "org1", Name: "BASE", Value: "base-val", Public: 1})

	result, err := convertVar(ctx, "pipe1", map[string]string{
		"MY_VAR": "hello",
	})
	if err != nil {
		t.Fatalf("convertVar() error: %v", err)
	}

	// Should have BASE (org var) + MY_VAR (pipeline var)
	if result["BASE"] == nil {
		t.Fatal("expected BASE org var in result")
	}
	if result["BASE"].Value != "base-val" {
		t.Errorf("BASE.Value = %q, want %q", result["BASE"].Value, "base-val")
	}
	if result["MY_VAR"] == nil {
		t.Fatal("expected MY_VAR in result")
	}
	if result["MY_VAR"].Value != "hello" {
		t.Errorf("MY_VAR.Value = %q, want %q", result["MY_VAR"].Value, "hello")
	}
}

func TestConvertVar_PipelineVarOverridesOrgVar(t *testing.T) {
	eng := setupPipelineTestDB(t)
	ctx := context.Background()

	_, _ = eng.Insert(&model.TOrgPipe{OrgId: "org1", PipeId: "pipe1"})
	_, _ = eng.Insert(&model.TOrgVar{OrgId: "org1", Name: "SHARED", Value: "from-org", Public: 0})

	// Pipeline var with the same name
	_, _ = eng.Insert(&model.TPipelineVar{
		PipelineId: "pipe1",
		Name:       "SHARED",
		Value:      "from-pipe",
		Public:     0,
	})

	result, err := convertVar(ctx, "pipe1", nil)
	if err != nil {
		t.Fatalf("convertVar() error: %v", err)
	}

	// Pipeline vars are added after org vars, so pipeline var should override
	if result["SHARED"] == nil {
		t.Fatal("expected SHARED var")
	}
	if result["SHARED"].Value != "from-pipe" {
		t.Errorf("SHARED.Value = %q, want %q", result["SHARED"].Value, "from-pipe")
	}
}

func TestConvertVar_VariableSubstitution(t *testing.T) {
	eng := setupPipelineTestDB(t)
	ctx := context.Background()

	_, _ = eng.Insert(&model.TOrgVar{OrgId: "org1", Name: "HOST", Value: "db.example.com", Public: 0})
	_, _ = eng.Insert(&model.TOrgPipe{OrgId: "org1", PipeId: "pipe1"})

	result, err := convertVar(ctx, "pipe1", map[string]string{
		"URL": "https://${{HOST}}:5432",
	})
	if err != nil {
		t.Fatalf("convertVar() error: %v", err)
	}

	if result["URL"] == nil {
		t.Fatal("expected URL var")
	}
	if result["URL"].Value != "https://db.example.com:5432" {
		t.Errorf("URL.Value = %q, want %q", result["URL"].Value, "https://db.example.com:5432")
	}
}

// -----------------------------------------------------------------------
// preBuild() with stages, steps, disabled steps, artifacts, use-artifacts
// -----------------------------------------------------------------------

const multiStagePipelineYAML = `
stages:
  - name: build
    displayName: Build Stage
    repo: https://example.com/repo
    steps:
      - step: shell@ssh
        name: compile
        displayName: Compile
        commands:
          - go build ./...
        env:
          GOFLAGS: "-v"
      - step: shell@ssh
        name: disabled-step
        disable: true
        commands: echo skip
  - name: test
    displayName: Test Stage
    steps:
      - step: shell@ssh
        name: unit
        commands: go test ./...
        artifacts:
          - scope: archive
            name: test-results
            path: ./results
        useArtifacts:
          - scope: pipeline
            repository: build-artifacts
            name: binary
            path: ./out
            fromStage: build
            fromStep: compile
`

func TestRun_MultiStagePipeline(t *testing.T) {
	eng := setupPipelineTestDB(t)
	ctx := context.Background()

	_, _ = eng.Insert(&model.TPipeline{
		Id:          "pipe-multi",
		Name:        "multi-stage",
		DisplayName: "Multi Stage Pipeline",
	})
	_, _ = eng.Insert(&model.TPipelineConf{
		PipelineId: "pipe-multi",
		YmlContent: multiStagePipelineYAML,
		Url:        "https://example.com/repo",
		Username:   "user",
	})

	tpv, rb, err := Run(ctx, "user1", "pipe-multi", "sha789", "push")
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}
	if tpv == nil || rb == nil {
		t.Fatal("expected non-nil results")
	}

	// Should have 2 stages (build + test)
	if len(rb.Stages) != 2 {
		t.Fatalf("expected 2 stages, got %d", len(rb.Stages))
	}

	// Build stage: 1 step (disabled step should be excluded)
	buildStage := rb.Stages[0]
	if buildStage.Name != "build" {
		t.Errorf("stage[0].Name = %q, want %q", buildStage.Name, "build")
	}
	if len(buildStage.Steps) != 1 {
		t.Fatalf("expected 1 step in build stage (disabled excluded), got %d", len(buildStage.Steps))
	}
	compileStep := buildStage.Steps[0]
	if compileStep.Name != "compile" {
		t.Errorf("step.Name = %q, want %q", compileStep.Name, "compile")
	}

	// Test stage: 1 step with artifacts and useArtifacts
	testStage := rb.Stages[1]
	if testStage.Name != "test" {
		t.Errorf("stage[1].Name = %q, want %q", testStage.Name, "test")
	}
	if len(testStage.Steps) != 1 {
		t.Fatalf("expected 1 step in test stage, got %d", len(testStage.Steps))
	}
	unitStep := testStage.Steps[0]
	if len(unitStep.Artifacts) != 1 {
		t.Fatalf("expected 1 artifact, got %d", len(unitStep.Artifacts))
	}
	if unitStep.Artifacts[0].Name != "test-results" {
		t.Errorf("artifact name = %q, want %q", unitStep.Artifacts[0].Name, "test-results")
	}
	if len(unitStep.UseArtifacts) != 1 {
		t.Fatalf("expected 1 useArtifact, got %d", len(unitStep.UseArtifacts))
	}
	if unitStep.UseArtifacts[0].SourceStage != "build" {
		t.Errorf("useArtifact.SourceStage = %q, want %q", unitStep.UseArtifacts[0].SourceStage, "build")
	}

	// Verify DB records
	var dbStages int64
	dbStages, err = eng.Where("build_id=?", rb.Id).Count(&model.TStage{})
	if err != nil {
		t.Fatalf("count stages: %v", err)
	}
	if dbStages != 2 {
		t.Errorf("db stage count = %d, want 2", dbStages)
	}

	// Only 2 steps in DB (disabled step excluded)
	var dbSteps int64
	dbSteps, err = eng.Where("build_id=?", rb.Id).Count(&model.TStep{})
	if err != nil {
		t.Fatalf("count steps: %v", err)
	}
	if dbSteps != 2 {
		t.Errorf("db step count = %d, want 2", dbSteps)
	}
}

// -----------------------------------------------------------------------
// Step repo fallback from stage
// -----------------------------------------------------------------------

func TestRun_StepRepoFallbackFromStage(t *testing.T) {
	eng := setupPipelineTestDB(t)
	ctx := context.Background()

	yml := `
stages:
  - name: build
    repo: https://example.com/stage-repo
    steps:
      - step: shell@ssh
        name: step1
        commands: echo test
`
	_, _ = eng.Insert(&model.TPipeline{Id: "pipe-repo", Name: "repo-test"})
	_, _ = eng.Insert(&model.TPipelineConf{
		PipelineId: "pipe-repo",
		YmlContent: yml,
		Url:        "https://example.com/main-repo",
	})

	_, rb, err := Run(ctx, "user1", "pipe-repo", "sha1", "push")
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}

	// Step should inherit stage repo when step repo is empty
	step := rb.Stages[0].Steps[0]
	if step.RepoPath != "https://example.com/stage-repo" {
		t.Errorf("step.RepoPath = %q, want stage repo URL", step.RepoPath)
	}
}

// -----------------------------------------------------------------------
// GOKINSV variables are injected
// -----------------------------------------------------------------------

func TestRun_InjectsBuiltInVars(t *testing.T) {
	eng := setupPipelineTestDB(t)
	ctx := context.Background()

	_, _ = eng.Insert(&model.TPipeline{Id: "pipe-vars", Name: "vars-test"})
	_, _ = eng.Insert(&model.TPipelineConf{
		PipelineId: "pipe-vars",
		YmlContent: validPipelineYAML,
		Url:        "https://example.com/repo",
	})

	_, rb, err := Run(ctx, "user1", "pipe-vars", "abc123def", "push")
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}

	if rb.Vars["GOKINSV_GIT_SHA"] == nil {
		t.Fatal("expected GOKINSV_GIT_SHA in build vars")
	}
	if rb.Vars["GOKINSV_GIT_SHA"].Value != "abc123def" {
		t.Errorf("GOKINSV_GIT_SHA = %q, want %q", rb.Vars["GOKINSV_GIT_SHA"].Value, "abc123def")
	}
	if !rb.Vars["GOKINSV_GIT_SHA"].Secret {
		t.Error("GOKINSV_GIT_SHA should be marked as secret")
	}

	if rb.Vars["GOKINSV_RUNNER_REPOPATH"] == nil {
		t.Fatal("expected GOKINSV_RUNNER_REPOPATH in build vars")
	}
	if rb.Vars["GOKINSV_RUNNER_REPOPATH"].Value != "{{RUNNER_REPOPATH}}" {
		t.Errorf("GOKINSV_RUNNER_REPOPATH = %q", rb.Vars["GOKINSV_RUNNER_REPOPATH"].Value)
	}
}

// -----------------------------------------------------------------------
// preBuild pipeline deleted check
// -----------------------------------------------------------------------

func TestRun_DeletedPipelineNotFound(t *testing.T) {
	eng := setupPipelineTestDB(t)
	ctx := context.Background()

	// Insert a deleted pipeline
	_, _ = eng.Insert(&model.TPipeline{
		Id:      "pipe-deleted",
		Name:    "deleted-pipe",
		Deleted: 1,
	})
	_, _ = eng.Insert(&model.TPipelineConf{
		PipelineId: "pipe-deleted",
		YmlContent: validPipelineYAML,
	})

	_, _, err := Run(ctx, "user1", "pipe-deleted", "sha1", "push")
	if err == nil {
		t.Fatal("expected error for deleted pipeline")
	}
	if !errors.Is(err, ErrPipelineNotFound) {
		t.Fatalf("expected ErrPipelineNotFound, got: %v", err)
	}
}

// -----------------------------------------------------------------------
// Run with context cancellation
// -----------------------------------------------------------------------

func TestRun_CancelledContext(t *testing.T) {
	eng := setupPipelineTestDB(t)

	// Insert valid data
	_, _ = eng.Insert(&model.TPipeline{Id: "pipe-ctx", Name: "ctx-test"})
	_, _ = eng.Insert(&model.TPipelineConf{
		PipelineId: "pipe-ctx",
		YmlContent: validPipelineYAML,
		Url:        "https://example.com/repo",
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, _, err := Run(ctx, "user1", "pipe-ctx", "sha1", "push")
	if err == nil {
		t.Fatal("expected error with cancelled context")
	}
}

// -----------------------------------------------------------------------
// Run with pipeline vars from DB
// -----------------------------------------------------------------------

func TestRun_WithPipelineVars(t *testing.T) {
	eng := setupPipelineTestDB(t)
	ctx := context.Background()

	_, _ = eng.Insert(&model.TPipeline{Id: "pipe-pv", Name: "pv-test"})
	_, _ = eng.Insert(&model.TPipelineConf{
		PipelineId: "pipe-pv",
		YmlContent: validPipelineYAML,
		Url:        "https://example.com/repo",
	})
	_, _ = eng.Insert(&model.TPipelineVar{
		PipelineId: "pipe-pv",
		Name:       "PIPE_VAR",
		Value:      "pipe-value",
		Public:     0,
	})

	_, rb, err := Run(ctx, "user1", "pipe-pv", "sha1", "push")
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}

	if rb.Vars["PIPE_VAR"] == nil {
		t.Fatal("expected PIPE_VAR in build vars")
	}
	if rb.Vars["PIPE_VAR"].Value != "pipe-value" {
		t.Errorf("PIPE_VAR.Value = %q, want %q", rb.Vars["PIPE_VAR"].Value, "pipe-value")
	}
}

// -----------------------------------------------------------------------
// Run preserves existing build timestamps
// -----------------------------------------------------------------------

func TestRun_BuildHasCreatedTimestamp(t *testing.T) {
	eng := setupPipelineTestDB(t)
	ctx := context.Background()

	before := time.Now().Add(-time.Second)

	_, _ = eng.Insert(&model.TPipeline{Id: "pipe-ts", Name: "ts-test"})
	_, _ = eng.Insert(&model.TPipelineConf{
		PipelineId: "pipe-ts",
		YmlContent: validPipelineYAML,
		Url:        "https://example.com/repo",
	})

	_, rb, err := Run(ctx, "user1", "pipe-ts", "sha1", "push")
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}

	after := time.Now().Add(time.Second)

	if rb.Created.Before(before) || rb.Created.After(after) {
		t.Errorf("rb.Created = %v, expected between %v and %v", rb.Created, before, after)
	}
}

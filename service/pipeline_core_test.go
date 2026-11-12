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

// setupPipelineTestDB creates an in-memory SQLite database with all tables needed
// for pipeline integration tests.
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
		t.Fatalf("failed to sync database schema: %v", err)
	}

	return eng
}

// validPipelineYml is a minimal valid pipeline YAML for testing.
const validPipelineYml = `
version: "1"
stages:
  - name: build
    displayName: Build
    steps:
      - step: shell
        name: compile
        displayName: Compile
        commands: "echo hello"
`

// --- Run ---

func TestRun_PipelineNotFound(t *testing.T) {
	setupPipelineTestDB(t)
	ctx := context.Background()

	_, _, err := Run(ctx, "user1", "nonexistent-pipe-id", "abc123", "push")
	if err == nil {
		t.Fatal("expected error for missing pipeline, got nil")
	}
	if !errors.Is(err, ErrPipelineNotFound) {
		t.Errorf("expected ErrPipelineNotFound, got: %v", err)
	}
}

func TestRun_EmptyYmlContent(t *testing.T) {
	eng := setupPipelineTestDB(t)
	ctx := context.Background()

	// Insert pipeline conf with empty YAML
	conf := &model.TPipelineConf{
		Aid:        1,
		PipelineId: "pipe-1",
		YmlContent: "",
		Url:        "https://github.com/test/repo",
		Username:   "testuser",
	}
	if _, err := eng.Insert(conf); err != nil {
		t.Fatalf("insert pipeline conf: %v", err)
	}

	_, _, err := Run(ctx, "user1", "pipe-1", "abc123", "push")
	if err == nil {
		t.Fatal("expected error for empty YAML, got nil")
	}
	if !errors.Is(err, ErrPipelineYmlEmpty) {
		t.Errorf("expected ErrPipelineYmlEmpty, got: %v", err)
	}
}

func TestRun_InvalidYmlContent(t *testing.T) {
	eng := setupPipelineTestDB(t)
	ctx := context.Background()

	// Insert pipeline conf with invalid YAML (missing required fields)
	conf := &model.TPipelineConf{
		Aid:        1,
		PipelineId: "pipe-inv",
		YmlContent: `version: "1"
stages:
  - name: ""
    steps: []
`,
		Url:      "https://github.com/test/repo",
		Username: "testuser",
	}
	if _, err := eng.Insert(conf); err != nil {
		t.Fatalf("insert pipeline conf: %v", err)
	}

	_, _, err := Run(ctx, "user1", "pipe-inv", "abc123", "push")
	if err == nil {
		t.Fatal("expected error for invalid YAML, got nil")
	}
}

func TestRun_Success(t *testing.T) {
	eng := setupPipelineTestDB(t)
	ctx := context.Background()

	// Insert pipeline and config
	pipe := &model.TPipeline{
		Id:          "pipe-ok",
		Uid:         "user1",
		Name:        "test-pipeline",
		DisplayName: "Test Pipeline",
		Deleted:     0,
	}
	if _, err := eng.Insert(pipe); err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	conf := &model.TPipelineConf{
		Aid:        1,
		PipelineId: "pipe-ok",
		YmlContent: validPipelineYml,
		Url:        "https://github.com/test/repo",
		Username:   "testuser",
		AccessToken: "tok123",
	}
	if _, err := eng.Insert(conf); err != nil {
		t.Fatalf("insert pipeline conf: %v", err)
	}

	tpv, rb, err := Run(ctx, "user1", "pipe-ok", "sha123", "push")
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}

	// Verify TPipelineVersion
	if tpv == nil {
		t.Fatal("expected non-nil TPipelineVersion")
	}
	if tpv.Uid != "user1" {
		t.Errorf("tpv.Uid = %q, want %q", tpv.Uid, "user1")
	}
	if tpv.Number != 1 {
		t.Errorf("tpv.Number = %d, want 1", tpv.Number)
	}
	if tpv.Events != "push" {
		t.Errorf("tpv.Events = %q, want %q", tpv.Events, "push")
	}
	if tpv.Sha != "sha123" {
		t.Errorf("tpv.Sha = %q, want %q", tpv.Sha, "sha123")
	}
	if tpv.PipelineId != "pipe-ok" {
		t.Errorf("tpv.PipelineId = %q, want %q", tpv.PipelineId, "pipe-ok")
	}
	if tpv.PipelineName != "test-pipeline" {
		t.Errorf("tpv.PipelineName = %q, want %q", tpv.PipelineName, "test-pipeline")
	}
	if tpv.RepoCloneUrl != "https://github.com/test/repo" {
		t.Errorf("tpv.RepoCloneUrl = %q, want repo URL", tpv.RepoCloneUrl)
	}

	// Verify runtime.Build
	if rb == nil {
		t.Fatal("expected non-nil runtime.Build")
	}
	if rb.PipelineId != "pipe-ok" {
		t.Errorf("rb.PipelineId = %q, want %q", rb.PipelineId, "pipe-ok")
	}
	if rb.Status != "pending" {
		t.Errorf("rb.Status = %q, want %q", rb.Status, "pending")
	}
	if rb.Repo == nil {
		t.Fatal("expected non-nil Repo")
	}
	if rb.Repo.CloneURL != "https://github.com/test/repo" {
		t.Errorf("rb.Repo.CloneURL = %q, want repo URL", rb.Repo.CloneURL)
	}
	if rb.Repo.Sha != "sha123" {
		t.Errorf("rb.Repo.Sha = %q, want %q", rb.Repo.Sha, "sha123")
	}
	if len(rb.Stages) != 1 {
		t.Fatalf("expected 1 stage, got %d", len(rb.Stages))
	}
	if rb.Stages[0].Name != "build" {
		t.Errorf("stage name = %q, want %q", rb.Stages[0].Name, "build")
	}
	if len(rb.Stages[0].Steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(rb.Stages[0].Steps))
	}
	if rb.Stages[0].Steps[0].Name != "compile" {
		t.Errorf("step name = %q, want %q", rb.Stages[0].Steps[0].Name, "compile")
	}

	// Verify GOKINSV_GIT_SHA was injected
	if _, ok := rb.Vars["GOKINSV_GIT_SHA"]; !ok {
		t.Error("expected GOKINSV_GIT_SHA to be set in vars")
	}
	if rb.Vars["GOKINSV_GIT_SHA"].Value != "sha123" {
		t.Errorf("GOKINSV_GIT_SHA = %q, want %q", rb.Vars["GOKINSV_GIT_SHA"].Value, "sha123")
	}

	// Verify DB records were inserted
	var version model.TPipelineVersion
	ok, err := eng.ID(tpv.Id).Get(&version)
	if err != nil || !ok {
		t.Fatal("pipeline version not found in database")
	}
	var build model.TBuild
	ok, err = eng.ID(rb.Id).Get(&build)
	if err != nil || !ok {
		t.Fatal("build not found in database")
	}
}

func TestRun_IncrementsNumber(t *testing.T) {
	eng := setupPipelineTestDB(t)
	ctx := context.Background()

	pipe := &model.TPipeline{
		Id: "pipe-inc", Name: "inc-pipe", DisplayName: "Inc", Deleted: 0,
	}
	if _, err := eng.Insert(pipe); err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}
	conf := &model.TPipelineConf{
		Aid: 1, PipelineId: "pipe-inc",
		YmlContent:  validPipelineYml,
		Url:         "https://example.com",
		Username:    "u",
		AccessToken: "t",
	}
	if _, err := eng.Insert(conf); err != nil {
		t.Fatalf("insert conf: %v", err)
	}

	// First run
	tpv1, _, err := Run(ctx, "user1", "pipe-inc", "sha1", "push")
	if err != nil {
		t.Fatalf("first Run() error: %v", err)
	}
	if tpv1.Number != 1 {
		t.Errorf("first run number = %d, want 1", tpv1.Number)
	}

	// Second run
	tpv2, _, err := Run(ctx, "user1", "pipe-inc", "sha2", "push")
	if err != nil {
		t.Fatalf("second Run() error: %v", err)
	}
	if tpv2.Number != 2 {
		t.Errorf("second run number = %d, want 2", tpv2.Number)
	}
}

// --- ReBuild ---

func TestReBuild_PipelineNotFound(t *testing.T) {
	setupPipelineTestDB(t)
	ctx := context.Background()

	tvp := &model.TPipelineVersion{
		Id:         "ver1",
		PipelineId: "nonexistent",
		Content:    validPipelineYml,
		Sha:        "abc",
	}

	_, _, err := ReBuild(ctx, "user1", tvp)
	if err == nil {
		t.Fatal("expected error for missing pipeline, got nil")
	}
	if !errors.Is(err, ErrPipelineNotFound) {
		t.Errorf("expected ErrPipelineNotFound, got: %v", err)
	}
}

func TestReBuild_EmptyContent(t *testing.T) {
	eng := setupPipelineTestDB(t)
	ctx := context.Background()

	pipe := &model.TPipeline{
		Id: "pipe-rb", Name: "rebuild", DisplayName: "RB", Deleted: 0,
	}
	if _, err := eng.Insert(pipe); err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}
	conf := &model.TPipelineConf{
		Aid: 1, PipelineId: "pipe-rb", YmlContent: validPipelineYml,
		Url: "https://example.com", Username: "u", AccessToken: "t",
	}
	if _, err := eng.Insert(conf); err != nil {
		t.Fatalf("insert conf: %v", err)
	}

	tvp := &model.TPipelineVersion{
		Id: "ver-empty", PipelineId: "pipe-rb",
		Content: "", Sha: "sha1",
	}

	_, _, err := ReBuild(ctx, "user1", tvp)
	if err == nil {
		t.Fatal("expected error for empty content, got nil")
	}
	if !errors.Is(err, ErrPipelineYmlEmpty) {
		t.Errorf("expected ErrPipelineYmlEmpty, got: %v", err)
	}
}

func TestReBuild_Success(t *testing.T) {
	eng := setupPipelineTestDB(t)
	ctx := context.Background()

	pipe := &model.TPipeline{
		Id: "pipe-rbs", Name: "rebuild-s", DisplayName: "RBS", Deleted: 0,
	}
	if _, err := eng.Insert(pipe); err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}
	conf := &model.TPipelineConf{
		Aid: 1, PipelineId: "pipe-rbs", YmlContent: validPipelineYml,
		Url: "https://example.com/repo", Username: "testuser", AccessToken: "tok",
	}
	if _, err := eng.Insert(conf); err != nil {
		t.Fatalf("insert conf: %v", err)
	}

	tvp := &model.TPipelineVersion{
		Id: "ver-rebuild", PipelineId: "pipe-rbs",
		Content: validPipelineYml, Sha: "old-sha",
	}

	tpv, rb, err := ReBuild(ctx, "user2", tvp)
	if err != nil {
		t.Fatalf("ReBuild() error: %v", err)
	}

	if tpv.Events != "rebuild" {
		t.Errorf("tpv.Events = %q, want %q", tpv.Events, "rebuild")
	}
	if tpv.Sha != "old-sha" {
		t.Errorf("tpv.Sha = %q, want %q", tpv.Sha, "old-sha")
	}
	if rb == nil {
		t.Fatal("expected non-nil Build")
	}
}

// --- getOrgVars ---

func TestGetOrgVars_NoData(t *testing.T) {
	setupPipelineTestDB(t)
	ctx := context.Background()

	vars, err := getOrgVars(ctx, "nonexistent-pipe")
	if err != nil {
		t.Fatalf("getOrgVars error: %v", err)
	}
	if len(vars) != 0 {
		t.Errorf("expected empty vars, got %d", len(vars))
	}
}

func TestGetOrgVars_WithData(t *testing.T) {
	eng := setupPipelineTestDB(t)
	ctx := context.Background()

	// Create org-pipe mapping
	orgPipe := &model.TOrgPipe{
		Aid: 1, OrgId: "org-1", PipeId: "pipe-1", Created: time.Now(),
	}
	if _, err := eng.Insert(orgPipe); err != nil {
		t.Fatalf("insert org pipe: %v", err)
	}

	// Create org vars
	vars := []model.TOrgVar{
		{Aid: 1, Uid: "u1", OrgId: "org-1", Name: "ORG_VAR_1", Value: "val1", Public: 0},
		{Aid: 2, Uid: "u1", OrgId: "org-1", Name: "ORG_VAR_2", Value: "val2", Public: 1},
	}
	for i := range vars {
		if _, err := eng.Insert(&vars[i]); err != nil {
			t.Fatalf("insert org var: %v", err)
		}
	}

	// Create another org with a var but no pipe mapping
	otherVar := &model.TOrgVar{
		Aid: 3, Uid: "u2", OrgId: "org-2", Name: "OTHER_VAR", Value: "other", Public: 0,
	}
	if _, err := eng.Insert(otherVar); err != nil {
		t.Fatalf("insert other var: %v", err)
	}

	result, err := getOrgVars(ctx, "pipe-1")
	if err != nil {
		t.Fatalf("getOrgVars error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 org vars, got %d", len(result))
	}

	found := map[string]bool{}
	for _, v := range result {
		found[v.Name] = true
	}
	if !found["ORG_VAR_1"] {
		t.Error("missing ORG_VAR_1")
	}
	if !found["ORG_VAR_2"] {
		t.Error("missing ORG_VAR_2")
	}
}

// --- convertVar ---

func TestConvertVar_NoVars(t *testing.T) {
	setupPipelineTestDB(t)
	ctx := context.Background()

	result, err := convertVar(ctx, "nonexistent-pipe", nil)
	if err != nil {
		t.Fatalf("convertVar error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty vars map, got %d entries", len(result))
	}
}

func TestConvertVar_PipelineVarsOnly(t *testing.T) {
	eng := setupPipelineTestDB(t)
	ctx := context.Background()

	pVars := []model.TPipelineVar{
		{Aid: 1, PipelineId: "pipe-v", Name: "PIPE_VAR", Value: "pval", Public: 0},
		{Aid: 2, PipelineId: "pipe-v", Name: "PUBLIC_VAR", Value: "pub", Public: 1},
	}
	for i := range pVars {
		if _, err := eng.Insert(&pVars[i]); err != nil {
			t.Fatalf("insert pipeline var: %v", err)
		}
	}

	result, err := convertVar(ctx, "pipe-v", map[string]string{})
	if err != nil {
		t.Fatalf("convertVar error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 vars, got %d", len(result))
	}
	if result["PIPE_VAR"].Value != "pval" {
		t.Errorf("PIPE_VAR.Value = %q, want %q", result["PIPE_VAR"].Value, "pval")
	}
	// Public=1 means Secret=true in convertVar
	if !result["PUBLIC_VAR"].Secret {
		t.Error("PUBLIC_VAR should be secret (Public=1)")
	}
}

func TestConvertVar_OrgAndPipelineVars(t *testing.T) {
	eng := setupPipelineTestDB(t)
	ctx := context.Background()

	// Org var
	orgPipe := &model.TOrgPipe{Aid: 1, OrgId: "org-x", PipeId: "pipe-mix", Created: time.Now()}
	if _, err := eng.Insert(orgPipe); err != nil {
		t.Fatalf("insert org pipe: %v", err)
	}
	orgVar := &model.TOrgVar{Aid: 1, OrgId: "org-x", Name: "ORG_V", Value: "orgval", Public: 0}
	if _, err := eng.Insert(orgVar); err != nil {
		t.Fatalf("insert org var: %v", err)
	}

	// Pipeline var
	pipeVar := &model.TPipelineVar{Aid: 1, PipelineId: "pipe-mix", Name: "PIPE_V", Value: "pipeval", Public: 0}
	if _, err := eng.Insert(pipeVar); err != nil {
		t.Fatalf("insert pipe var: %v", err)
	}

	result, err := convertVar(ctx, "pipe-mix", map[string]string{})
	if err != nil {
		t.Fatalf("convertVar error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 vars, got %d", len(result))
	}
	if result["ORG_V"].Value != "orgval" {
		t.Errorf("ORG_V.Value = %q, want %q", result["ORG_V"].Value, "orgval")
	}
	if result["PIPE_V"].Value != "pipeval" {
		t.Errorf("PIPE_V.Value = %q, want %q", result["PIPE_V"].Value, "pipeval")
	}
}

func TestConvertVar_WithYamlVarReplacements(t *testing.T) {
	eng := setupPipelineTestDB(t)
	ctx := context.Background()

	pipeVar := &model.TPipelineVar{
		Aid: 1, PipelineId: "pipe-repl", Name: "BASE_URL", Value: "https://api.example.com", Public: 0,
	}
	if _, err := eng.Insert(pipeVar); err != nil {
		t.Fatalf("insert pipe var: %v", err)
	}

	// YAML vars that reference pipeline vars
	yamlVars := map[string]string{
		"API_ENDPOINT": "${{BASE_URL}}/v1",
	}

	result, err := convertVar(ctx, "pipe-repl", yamlVars)
	if err != nil {
		t.Fatalf("convertVar error: %v", err)
	}

	if result["BASE_URL"].Value != "https://api.example.com" {
		t.Errorf("BASE_URL.Value = %q, want API URL", result["BASE_URL"].Value)
	}
	if result["API_ENDPOINT"].Value != "https://api.example.com/v1" {
		t.Errorf("API_ENDPOINT.Value = %q, want %q", result["API_ENDPOINT"].Value, "https://api.example.com/v1")
	}
}

// --- Run with org/pipeline vars ---

func TestRun_WithVariables(t *testing.T) {
	eng := setupPipelineTestDB(t)
	ctx := context.Background()

	pipe := &model.TPipeline{
		Id: "pipe-vars", Name: "vars-pipe", DisplayName: "Vars", Deleted: 0,
	}
	if _, err := eng.Insert(pipe); err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	ymlWithVars := `
version: "1"
vars:
  DEPLOY_ENV: "${{SERVER}}"
stages:
  - name: deploy
    displayName: Deploy
    steps:
      - step: shell
        name: run
        displayName: Run deploy
        commands: "echo deploy"
`
	conf := &model.TPipelineConf{
		Aid: 1, PipelineId: "pipe-vars", YmlContent: ymlWithVars,
		Url: "https://example.com", Username: "u", AccessToken: "t",
	}
	if _, err := eng.Insert(conf); err != nil {
		t.Fatalf("insert conf: %v", err)
	}

	// Add pipeline var
	pipeVar := &model.TPipelineVar{
		Aid: 1, PipelineId: "pipe-vars", Name: "SERVER", Value: "prod-server", Public: 0,
	}
	if _, err := eng.Insert(pipeVar); err != nil {
		t.Fatalf("insert pipe var: %v", err)
	}

	tpv, rb, err := Run(ctx, "user1", "pipe-vars", "sha456", "push")
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}

	if tpv == nil || rb == nil {
		t.Fatal("expected non-nil results")
	}

	// Check that vars were resolved
	if v, ok := rb.Vars["SERVER"]; !ok {
		t.Error("missing SERVER var")
	} else if v.Value != "prod-server" {
		t.Errorf("SERVER = %q, want %q", v.Value, "prod-server")
	}
	if v, ok := rb.Vars["DEPLOY_ENV"]; !ok {
		t.Error("missing DEPLOY_ENV var")
	} else if v.Value != "prod-server" {
		t.Errorf("DEPLOY_ENV = %q, want %q", v.Value, "prod-server")
	}
}

// --- Run with disabled step ---

func TestRun_DisabledStepSkipped(t *testing.T) {
	eng := setupPipelineTestDB(t)
	ctx := context.Background()

	pipe := &model.TPipeline{
		Id: "pipe-dis", Name: "dis-pipe", DisplayName: "Dis", Deleted: 0,
	}
	if _, err := eng.Insert(pipe); err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	ymlDisabled := `
version: "1"
stages:
  - name: build
    displayName: Build
    steps:
      - step: shell
        name: enabled-step
        displayName: Enabled
        commands: "echo enabled"
      - step: shell
        name: disabled-step
        displayName: Disabled
        disable: true
        commands: "echo disabled"
`
	conf := &model.TPipelineConf{
		Aid: 1, PipelineId: "pipe-dis", YmlContent: ymlDisabled,
		Url: "https://example.com", Username: "u", AccessToken: "t",
	}
	if _, err := eng.Insert(conf); err != nil {
		t.Fatalf("insert conf: %v", err)
	}

	_, rb, err := Run(ctx, "user1", "pipe-dis", "sha789", "push")
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}

	// Only 1 step should be in the build (disabled step skipped)
	if len(rb.Stages) != 1 {
		t.Fatalf("expected 1 stage, got %d", len(rb.Stages))
	}
	if len(rb.Stages[0].Steps) != 1 {
		t.Fatalf("expected 1 step (disabled skipped), got %d", len(rb.Stages[0].Steps))
	}
	if rb.Stages[0].Steps[0].Name != "enabled-step" {
		t.Errorf("step name = %q, want %q", rb.Stages[0].Steps[0].Name, "enabled-step")
	}
}

// --- preBuild: pipeline deleted check ---

func TestRun_DeletedPipeline(t *testing.T) {
	eng := setupPipelineTestDB(t)
	ctx := context.Background()

	pipe := &model.TPipeline{
		Id: "pipe-del", Name: "del", DisplayName: "Del", Deleted: 1,
	}
	if _, err := eng.Insert(pipe); err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}
	conf := &model.TPipelineConf{
		Aid: 1, PipelineId: "pipe-del", YmlContent: validPipelineYml,
		Url: "https://example.com", Username: "u", AccessToken: "t",
	}
	if _, err := eng.Insert(conf); err != nil {
		t.Fatalf("insert conf: %v", err)
	}

	_, _, err := Run(ctx, "user1", "pipe-del", "sha", "push")
	if err == nil {
		t.Fatal("expected error for deleted pipeline, got nil")
	}
	if !errors.Is(err, ErrPipelineNotFound) {
		t.Errorf("expected ErrPipelineNotFound, got: %v", err)
	}
}

// --- Run with context cancellation ---

func TestRun_ContextCancelled(t *testing.T) {
	eng := setupPipelineTestDB(t)

	pipe := &model.TPipeline{
		Id: "pipe-ctx", Name: "ctx", DisplayName: "Ctx", Deleted: 0,
	}
	if _, err := eng.Insert(pipe); err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}
	conf := &model.TPipelineConf{
		Aid: 1, PipelineId: "pipe-ctx", YmlContent: validPipelineYml,
		Url: "https://example.com", Username: "u", AccessToken: "t",
	}
	if _, err := eng.Insert(conf); err != nil {
		t.Fatalf("insert conf: %v", err)
	}

	// Cancel context before calling Run
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// With a cancelled context, the DB query should fail
	_, _, err := Run(ctx, "user1", "pipe-ctx", "sha", "push")
	if err == nil {
		t.Fatal("expected error with cancelled context, got nil")
	}
}

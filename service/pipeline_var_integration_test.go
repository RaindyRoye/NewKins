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

// setupPipelineVarTestDB creates an in-memory SQLite database with all tables
// needed for getOrgVars and convertVar tests.
func setupPipelineVarTestDB(t *testing.T) *xorm.Engine {
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
	if err := eng.Sync2(
		&model.TOrg{},
		&model.TOrgPipe{},
		&model.TOrgVar{},
		&model.TPipeline{},
		&model.TPipelineVar{},
		&model.TPipelineConf{},
		&model.TPipelineVersion{},
		&model.TBuild{},
		&model.TStage{},
		&model.TStep{},
		&model.TUser{},
	); err != nil {
		t.Fatalf("failed to sync schema: %v", err)
	}
	return eng
}

// --- getOrgVars integration tests ---

func TestGetOrgVars_NoOrgVars(t *testing.T) {
	eng := setupPipelineVarTestDB(t)
	ctx := context.Background()

	// Pipeline exists but has no org associations
	if _, err := eng.Insert(&model.TPipeline{Id: "pipe-1", Name: "p1"}); err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	result, err := getOrgVars(ctx, "pipe-1")
	if err != nil {
		t.Fatalf("getOrgVars error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty org vars, got %d", len(result))
	}
}

func TestGetOrgVars_WithOrgVars(t *testing.T) {
	eng := setupPipelineVarTestDB(t)
	ctx := context.Background()

	// Create org, pipeline association, and org vars
	if _, err := eng.Insert(&model.TOrg{Id: "org-1", Aid: 1, Name: "org1", Created: time.Now()}); err != nil {
		t.Fatalf("insert org: %v", err)
	}
	if _, err := eng.Insert(&model.TOrgPipe{Aid: 1, OrgId: "org-1", PipeId: "pipe-1", Created: time.Now()}); err != nil {
		t.Fatalf("insert org pipe: %v", err)
	}
	orgVars := []model.TOrgVar{
		{Aid: 1, OrgId: "org-1", Name: "ORG_KEY1", Value: "org-val1", Public: 1},
		{Aid: 2, OrgId: "org-1", Name: "ORG_KEY2", Value: "org-val2", Public: 0},
	}
	for _, v := range orgVars {
		if _, err := eng.Insert(&v); err != nil {
			t.Fatalf("insert org var: %v", err)
		}
	}

	result, err := getOrgVars(ctx, "pipe-1")
	if err != nil {
		t.Fatalf("getOrgVars error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 org vars, got %d", len(result))
	}
	// Check that both org vars are returned (order may vary)
	foundKey1, foundKey2 := false, false
	for _, v := range result {
		if v.Name == "ORG_KEY1" && v.Value == "org-val1" {
			foundKey1 = true
		}
		if v.Name == "ORG_KEY2" && v.Value == "org-val2" {
			foundKey2 = true
		}
	}
	if !foundKey1 {
		t.Error("ORG_KEY1 not found in result")
	}
	if !foundKey2 {
		t.Error("ORG_KEY2 not found in result")
	}
}

func TestGetOrgVars_MultipleOrgs(t *testing.T) {
	eng := setupPipelineVarTestDB(t)
	ctx := context.Background()

	// Pipeline belongs to two orgs, each with their own vars
	if _, err := eng.Insert(&model.TOrg{Id: "org-a", Aid: 1, Name: "orgA", Created: time.Now()}); err != nil {
		t.Fatalf("insert org: %v", err)
	}
	if _, err := eng.Insert(&model.TOrg{Id: "org-b", Aid: 2, Name: "orgB", Created: time.Now()}); err != nil {
		t.Fatalf("insert org: %v", err)
	}
	if _, err := eng.Insert(&model.TOrgPipe{Aid: 1, OrgId: "org-a", PipeId: "pipe-multi", Created: time.Now()}); err != nil {
		t.Fatalf("insert org pipe: %v", err)
	}
	if _, err := eng.Insert(&model.TOrgPipe{Aid: 2, OrgId: "org-b", PipeId: "pipe-multi", Created: time.Now()}); err != nil {
		t.Fatalf("insert org pipe: %v", err)
	}
	if _, err := eng.Insert(&model.TOrgVar{Aid: 1, OrgId: "org-a", Name: "VAR_A", Value: "val-a", Public: 1}); err != nil {
		t.Fatalf("insert org var: %v", err)
	}
	if _, err := eng.Insert(&model.TOrgVar{Aid: 2, OrgId: "org-b", Name: "VAR_B", Value: "val-b", Public: 0}); err != nil {
		t.Fatalf("insert org var: %v", err)
	}

	result, err := getOrgVars(ctx, "pipe-multi")
	if err != nil {
		t.Fatalf("getOrgVars error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 org vars, got %d", len(result))
	}
}

func TestGetOrgVars_NonexistentPipeline(t *testing.T) {
	setupPipelineVarTestDB(t)
	ctx := context.Background()

	result, err := getOrgVars(ctx, "nonexistent-pipe")
	if err != nil {
		t.Fatalf("getOrgVars error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty org vars for nonexistent pipeline, got %d", len(result))
	}
}

// --- convertVar integration tests ---

func TestConvertVar_NoVars(t *testing.T) {
	setupPipelineVarTestDB(t)
	ctx := context.Background()

	// Pipeline exists but has no org or pipeline vars
	result, err := convertVar(ctx, "pipe-empty", map[string]string{"KEY": "value"})
	if err != nil {
		t.Fatalf("convertVar error: %v", err)
	}
	// Should at least contain the input var
	if result["KEY"] == nil || result["KEY"].Value != "value" {
		t.Errorf("KEY should be 'value', got %v", result["KEY"])
	}
}

func TestConvertVar_PipelineVarsOnly(t *testing.T) {
	eng := setupPipelineVarTestDB(t)
	ctx := context.Background()

	// Insert pipeline vars
	pipeVars := []model.TPipelineVar{
		{Aid: 1, PipelineId: "pipe-1", Name: "PIPE_VAR", Value: "pipe-val", Public: 1},
		{Aid: 2, PipelineId: "pipe-1", Name: "SECRET_VAR", Value: "secret-val", Public: 0},
	}
	for _, v := range pipeVars {
		if _, err := eng.Insert(&v); err != nil {
			t.Fatalf("insert pipeline var: %v", err)
		}
	}

	result, err := convertVar(ctx, "pipe-1", nil)
	if err != nil {
		t.Fatalf("convertVar error: %v", err)
	}
	if result["PIPE_VAR"] == nil || result["PIPE_VAR"].Value != "pipe-val" {
		t.Errorf("PIPE_VAR should be 'pipe-val', got %v", result["PIPE_VAR"])
	}
	if result["PIPE_VAR"] == nil || !result["PIPE_VAR"].Secret {
		t.Error("PIPE_VAR should be marked as secret (Public=1)")
	}
	if result["SECRET_VAR"] == nil || result["SECRET_VAR"].Value != "secret-val" {
		t.Errorf("SECRET_VAR should be 'secret-val', got %v", result["SECRET_VAR"])
	}
	if result["SECRET_VAR"] == nil || result["SECRET_VAR"].Secret {
		t.Error("SECRET_VAR should not be marked as secret (Public=0)")
	}
}

func TestConvertVar_OrgAndPipelineVars(t *testing.T) {
	eng := setupPipelineVarTestDB(t)
	ctx := context.Background()

	// Setup org + org pipe + org var
	if _, err := eng.Insert(&model.TOrg{Id: "org-cv", Aid: 1, Name: "orgCV", Created: time.Now()}); err != nil {
		t.Fatalf("insert org: %v", err)
	}
	if _, err := eng.Insert(&model.TOrgPipe{Aid: 1, OrgId: "org-cv", PipeId: "pipe-cv", Created: time.Now()}); err != nil {
		t.Fatalf("insert org pipe: %v", err)
	}
	if _, err := eng.Insert(&model.TOrgVar{Aid: 1, OrgId: "org-cv", Name: "ORG_VAR", Value: "org-v", Public: 1}); err != nil {
		t.Fatalf("insert org var: %v", err)
	}
	// Pipeline var
	if _, err := eng.Insert(&model.TPipelineVar{Aid: 1, PipelineId: "pipe-cv", Name: "PIPE_VAR", Value: "pipe-v", Public: 0}); err != nil {
		t.Fatalf("insert pipeline var: %v", err)
	}

	result, err := convertVar(ctx, "pipe-cv", map[string]string{"INPUT": "input-v"})
	if err != nil {
		t.Fatalf("convertVar error: %v", err)
	}

	// Should have all three: org, pipeline, and input vars
	if result["ORG_VAR"] == nil || result["ORG_VAR"].Value != "org-v" {
		t.Errorf("ORG_VAR should be 'org-v', got %v", result["ORG_VAR"])
	}
	if result["PIPE_VAR"] == nil || result["PIPE_VAR"].Value != "pipe-v" {
		t.Errorf("PIPE_VAR should be 'pipe-v', got %v", result["PIPE_VAR"])
	}
	if result["INPUT"] == nil || result["INPUT"].Value != "input-v" {
		t.Errorf("INPUT should be 'input-v', got %v", result["INPUT"])
	}
}

func TestConvertVar_InputVarOverridesOrgVar(t *testing.T) {
	eng := setupPipelineVarTestDB(t)
	ctx := context.Background()

	// Setup org var
	if _, err := eng.Insert(&model.TOrg{Id: "org-ov", Aid: 1, Name: "orgOV", Created: time.Now()}); err != nil {
		t.Fatalf("insert org: %v", err)
	}
	if _, err := eng.Insert(&model.TOrgPipe{Aid: 1, OrgId: "org-ov", PipeId: "pipe-ov", Created: time.Now()}); err != nil {
		t.Fatalf("insert org pipe: %v", err)
	}
	if _, err := eng.Insert(&model.TOrgVar{Aid: 1, OrgId: "org-ov", Name: "SHARED_KEY", Value: "org-value", Public: 1}); err != nil {
		t.Fatalf("insert org var: %v", err)
	}

	// Input has the same key - should override
	result, err := convertVar(ctx, "pipe-ov", map[string]string{"SHARED_KEY": "input-value"})
	if err != nil {
		t.Fatalf("convertVar error: %v", err)
	}

	// Input should win over org var
	if result["SHARED_KEY"] == nil || result["SHARED_KEY"].Value != "input-value" {
		t.Errorf("SHARED_KEY should be 'input-value', got %v", result["SHARED_KEY"])
	}
}

func TestConvertVar_InputVarOverridesPipelineVar(t *testing.T) {
	eng := setupPipelineVarTestDB(t)
	ctx := context.Background()

	if _, err := eng.Insert(&model.TPipelineVar{Aid: 1, PipelineId: "pipe-ovp", Name: "SHARED", Value: "pipe-value", Public: 0}); err != nil {
		t.Fatalf("insert pipeline var: %v", err)
	}

	result, err := convertVar(ctx, "pipe-ovp", map[string]string{"SHARED": "input-value"})
	if err != nil {
		t.Fatalf("convertVar error: %v", err)
	}
	if result["SHARED"] == nil || result["SHARED"].Value != "input-value" {
		t.Errorf("SHARED should be 'input-value', got %v", result["SHARED"])
	}
}

func TestConvertVar_EmptyInputVars(t *testing.T) {
	eng := setupPipelineVarTestDB(t)
	ctx := context.Background()

	if _, err := eng.Insert(&model.TPipelineVar{Aid: 1, PipelineId: "pipe-ev", Name: "ONLY_PIPE", Value: "pipe-only", Public: 1}); err != nil {
		t.Fatalf("insert pipeline var: %v", err)
	}

	result, err := convertVar(ctx, "pipe-ev", nil)
	if err != nil {
		t.Fatalf("convertVar error: %v", err)
	}
	if result["ONLY_PIPE"] == nil || result["ONLY_PIPE"].Value != "pipe-only" {
		t.Errorf("ONLY_PIPE should be 'pipe-only', got %v", result["ONLY_PIPE"])
	}
}

// --- Run function integration tests ---

func TestRun_PipelineNotFound(t *testing.T) {
	setupPipelineVarTestDB(t)
	ctx := context.Background()

	_, _, err := Run(ctx, "user-1", "nonexistent-pipe", "abc123", "push")
	if !errors.Is(err, ErrPipelineNotFound) {
		t.Errorf("Run() should return ErrPipelineNotFound, got: %v", err)
	}
}

func TestRun_PipelineYmlEmpty(t *testing.T) {
	eng := setupPipelineVarTestDB(t)
	ctx := context.Background()

	// Insert pipeline conf with empty YmlContent
	if _, err := eng.Insert(&model.TPipelineConf{
		Aid:        1,
		PipelineId: "pipe-empty-yml",
		YmlContent: "",
	}); err != nil {
		t.Fatalf("insert pipeline conf: %v", err)
	}

	_, _, err := Run(ctx, "user-1", "pipe-empty-yml", "abc123", "push")
	if !errors.Is(err, ErrPipelineYmlEmpty) {
		t.Errorf("Run() should return ErrPipelineYmlEmpty, got: %v", err)
	}
}

// --- ReBuild function integration tests ---

func TestReBuild_PipelineNotFound(t *testing.T) {
	setupPipelineVarTestDB(t)
	ctx := context.Background()

	tvp := &model.TPipelineVersion{
		Id:         "tvp-1",
		PipelineId: "nonexistent-pipe",
		Content:    "version: '1'",
	}

	_, _, err := ReBuild(ctx, "user-1", tvp)
	if !errors.Is(err, ErrPipelineNotFound) {
		t.Errorf("ReBuild() should return ErrPipelineNotFound, got: %v", err)
	}
}

func TestReBuild_PipelineYmlEmpty(t *testing.T) {
	eng := setupPipelineVarTestDB(t)
	ctx := context.Background()

	// Insert pipeline conf
	if _, err := eng.Insert(&model.TPipelineConf{
		Aid:        1,
		PipelineId: "pipe-rebuild",
		YmlContent: "version: '1'",
	}); err != nil {
		t.Fatalf("insert pipeline conf: %v", err)
	}

	tvp := &model.TPipelineVersion{
		Id:         "tvp-2",
		PipelineId: "pipe-rebuild",
		Content:    "", // empty content triggers ErrPipelineYmlEmpty
	}

	_, _, err := ReBuild(ctx, "user-1", tvp)
	if !errors.Is(err, ErrPipelineYmlEmpty) {
		t.Errorf("ReBuild() should return ErrPipelineYmlEmpty, got: %v", err)
	}
}

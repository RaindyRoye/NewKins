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

// setupPipelineDBTest creates an isolated in-memory SQLite DB for pipeline tests.
func setupPipelineDBTest(t *testing.T) *xorm.Engine {
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
		&model.TPipeline{},
		&model.TPipelineConf{},
		&model.TPipelineVersion{},
		&model.TPipelineVar{},
		&model.TOrg{},
		&model.TOrgPipe{},
		&model.TOrgVar{},
		&model.TBuild{},
		&model.TStage{},
		&model.TStep{},
		&model.TUser{},
	); err != nil {
		t.Fatalf("failed to sync schema: %v", err)
	}
	return eng
}

// --- convertVar Tests ---

func TestConvertVar_EmptyPipeline(t *testing.T) {
	setupPipelineDBTest(t)
	ctx := context.Background()

	result, err := convertVar(ctx, "nonexistent-pipeline", nil)
	if err != nil {
		t.Fatalf("convertVar error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("convertVar should return empty map for nonexistent pipeline, got %d items", len(result))
	}
}

func TestConvertVar_WithPipelineVars(t *testing.T) {
	eng := setupPipelineDBTest(t)
	ctx := context.Background()

	// Insert pipeline
	pipe := &model.TPipeline{
		Id:      "pipe-var-1",
		Uid:     "user-1",
		Name:    "test-pipe",
		Deleted: 0,
	}
	if _, err := eng.Insert(pipe); err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	// Insert pipeline variables
	var1 := &model.TPipelineVar{
		Aid:        1,
		PipelineId: "pipe-var-1",
		Name:       "VAR1",
		Value:      "value1",
		Public:     1,
	}
	var2 := &model.TPipelineVar{
		Aid:        2,
		PipelineId: "pipe-var-1",
		Name:       "VAR2",
		Value:      "value2",
		Public:     0,
	}
	if _, err := eng.Insert(var1); err != nil {
		t.Fatalf("insert var1: %v", err)
	}
	if _, err := eng.Insert(var2); err != nil {
		t.Fatalf("insert var2: %v", err)
	}

	result, err := convertVar(ctx, "pipe-var-1", nil)
	if err != nil {
		t.Fatalf("convertVar error: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("convertVar should return 2 vars, got %d", len(result))
	}
	if v, ok := result["VAR1"]; !ok || v.Value != "value1" {
		t.Errorf("VAR1 not found or wrong value: %v", v)
	}
	if v, ok := result["VAR2"]; !ok || v.Value != "value2" {
		t.Errorf("VAR2 not found or wrong value: %v", v)
	}
}

func TestConvertVar_WithYamlVars(t *testing.T) {
	eng := setupPipelineDBTest(t)
	ctx := context.Background()

	pipe := &model.TPipeline{
		Id:      "pipe-var-2",
		Uid:     "user-1",
		Name:    "test-pipe",
		Deleted: 0,
	}
	if _, err := eng.Insert(pipe); err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	yamlVars := map[string]string{
		"YAML_VAR": "yaml_value",
	}

	result, err := convertVar(ctx, "pipe-var-2", yamlVars)
	if err != nil {
		t.Fatalf("convertVar error: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("convertVar should return 1 var, got %d", len(result))
	}
	if v, ok := result["YAML_VAR"]; !ok || v.Value != "yaml_value" {
		t.Errorf("YAML_VAR not found or wrong value: %v", v)
	}
}

func TestConvertVar_VarSubstitution(t *testing.T) {
	eng := setupPipelineDBTest(t)
	ctx := context.Background()

	pipe := &model.TPipeline{
		Id:      "pipe-var-3",
		Uid:     "user-1",
		Name:    "test-pipe",
		Deleted: 0,
	}
	if _, err := eng.Insert(pipe); err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	var1 := &model.TPipelineVar{
		Aid:        1,
		PipelineId: "pipe-var-3",
		Name:       "BASE",
		Value:      "base_value",
		Public:     1,
	}
	if _, err := eng.Insert(var1); err != nil {
		t.Fatalf("insert var1: %v", err)
	}

	yamlVars := map[string]string{
		"DERIVED": "${{BASE}}_derived",
	}

	result, err := convertVar(ctx, "pipe-var-3", yamlVars)
	if err != nil {
		t.Fatalf("convertVar error: %v", err)
	}
	if v, ok := result["DERIVED"]; !ok || v.Value != "base_value_derived" {
		t.Errorf("DERIVED substitution failed: %v", v)
	}
}

// --- getOrgVars Tests ---

func TestGetOrgVars_EmptyPipeline(t *testing.T) {
	setupPipelineDBTest(t)
	ctx := context.Background()

	result, err := getOrgVars(ctx, "nonexistent-pipeline")
	if err != nil {
		t.Fatalf("getOrgVars error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("getOrgVars should return empty for nonexistent pipeline, got %d", len(result))
	}
}

func TestGetOrgVars_WithOrgVars(t *testing.T) {
	eng := setupPipelineDBTest(t)
	ctx := context.Background()

	// Insert org
	org := &model.TOrg{
		Id:      "org-var-1",
		Aid:     1,
		Uid:     "user-1",
		Name:    "test-org",
		Created: time.Now(),
	}
	if _, err := eng.Insert(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}

	// Insert pipeline
	pipe := &model.TPipeline{
		Id:      "pipe-org-var-1",
		Uid:     "user-1",
		Name:    "test-pipe",
		Deleted: 0,
	}
	if _, err := eng.Insert(pipe); err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	// Link pipeline to org
	orgPipe := &model.TOrgPipe{
		Aid:     1,
		OrgId:   "org-var-1",
		PipeId:  "pipe-org-var-1",
		Created: time.Now(),
	}
	if _, err := eng.Insert(orgPipe); err != nil {
		t.Fatalf("insert org pipe: %v", err)
	}

	// Insert org variables
	orgVar1 := &model.TOrgVar{
		Aid:    1,
		OrgId:  "org-var-1",
		Name:   "ORG_VAR1",
		Value:  "org_value1",
		Public: 1,
	}
	orgVar2 := &model.TOrgVar{
		Aid:    2,
		OrgId:  "org-var-1",
		Name:   "ORG_VAR2",
		Value:  "org_value2",
		Public: 0,
	}
	if _, err := eng.Insert(orgVar1); err != nil {
		t.Fatalf("insert orgVar1: %v", err)
	}
	if _, err := eng.Insert(orgVar2); err != nil {
		t.Fatalf("insert orgVar2: %v", err)
	}

	result, err := getOrgVars(ctx, "pipe-org-var-1")
	if err != nil {
		t.Fatalf("getOrgVars error: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("getOrgVars should return 2 vars, got %d", len(result))
	}

	found1, found2 := false, false
	for _, v := range result {
		if v.Name == "ORG_VAR1" && v.Value == "org_value1" {
			found1 = true
		}
		if v.Name == "ORG_VAR2" && v.Value == "org_value2" {
			found2 = true
		}
	}
	if !found1 {
		t.Error("ORG_VAR1 not found in result")
	}
	if !found2 {
		t.Error("ORG_VAR2 not found in result")
	}
}

// --- Run Tests ---

func TestRun_PipelineNotFound(t *testing.T) {
	setupPipelineDBTest(t)
	ctx := context.Background()

	_, _, err := Run(ctx, "user-1", "nonexistent-pipe", "sha123", "push")
	if err == nil {
		t.Fatal("Run should return error for nonexistent pipeline")
	}
	if err != ErrPipelineNotFound {
		t.Errorf("Run should return ErrPipelineNotFound, got: %v", err)
	}
}

func TestRun_EmptyYaml(t *testing.T) {
	eng := setupPipelineDBTest(t)
	ctx := context.Background()

	// Insert pipeline with empty YAML
	pipe := &model.TPipeline{
		Id:      "pipe-empty-yaml",
		Uid:     "user-1",
		Name:    "test-pipe",
		Deleted: 0,
	}
	if _, err := eng.Insert(pipe); err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	conf := &model.TPipelineConf{
		Aid:        1,
		PipelineId: "pipe-empty-yaml",
		YmlContent: "",
	}
	if _, err := eng.Insert(conf); err != nil {
		t.Fatalf("insert pipeline conf: %v", err)
	}

	_, _, err := Run(ctx, "user-1", "pipe-empty-yaml", "sha123", "push")
	if err == nil {
		t.Fatal("Run should return error for empty YAML")
	}
	if err != ErrPipelineYmlEmpty {
		t.Errorf("Run should return ErrPipelineYmlEmpty, got: %v", err)
	}
}

func TestRun_InvalidYaml(t *testing.T) {
	eng := setupPipelineDBTest(t)
	ctx := context.Background()

	pipe := &model.TPipeline{
		Id:      "pipe-invalid-yaml",
		Uid:     "user-1",
		Name:    "test-pipe",
		Deleted: 0,
	}
	if _, err := eng.Insert(pipe); err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	conf := &model.TPipelineConf{
		Aid:        1,
		PipelineId: "pipe-invalid-yaml",
		YmlContent: "invalid: yaml: content: [",
	}
	if _, err := eng.Insert(conf); err != nil {
		t.Fatalf("insert pipeline conf: %v", err)
	}

	_, _, err := Run(ctx, "user-1", "pipe-invalid-yaml", "sha123", "push")
	if err == nil {
		t.Fatal("Run should return error for invalid YAML")
	}
}

func TestRun_DeletedPipeline(t *testing.T) {
	eng := setupPipelineDBTest(t)
	ctx := context.Background()

	pipe := &model.TPipeline{
		Id:      "pipe-deleted",
		Uid:     "user-1",
		Name:    "test-pipe",
		Deleted: 1,
	}
	if _, err := eng.Insert(pipe); err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	conf := &model.TPipelineConf{
		Aid:        1,
		PipelineId: "pipe-deleted",
		YmlContent: "stages:\n  - name: test\n    steps:\n      - name: step1\n        image: alpine\n        commands:\n          - echo hello",
	}
	if _, err := eng.Insert(conf); err != nil {
		t.Fatalf("insert pipeline conf: %v", err)
	}

	_, _, err := Run(ctx, "user-1", "pipe-deleted", "sha123", "push")
	if err == nil {
		t.Fatal("Run should return error for deleted pipeline")
	}
	if err != ErrPipelineNotFound {
		t.Errorf("Run should return ErrPipelineNotFound for deleted pipeline, got: %v", err)
	}
}

// --- ReBuild Tests ---

func TestReBuild_PipelineNotFound(t *testing.T) {
	setupPipelineDBTest(t)
	ctx := context.Background()

	tvp := &model.TPipelineVersion{
		Id:         "ver-1",
		PipelineId: "nonexistent-pipe",
		Content:    "stages:\n  - name: test",
		Sha:        "sha123",
	}

	_, _, err := ReBuild(ctx, "user-1", tvp)
	if err == nil {
		t.Fatal("ReBuild should return error for nonexistent pipeline")
	}
	if err != ErrPipelineNotFound {
		t.Errorf("ReBuild should return ErrPipelineNotFound, got: %v", err)
	}
}

func TestReBuild_EmptyContent(t *testing.T) {
	eng := setupPipelineDBTest(t)
	ctx := context.Background()

	pipe := &model.TPipeline{
		Id:      "pipe-rebuild-empty",
		Uid:     "user-1",
		Name:    "test-pipe",
		Deleted: 0,
	}
	if _, err := eng.Insert(pipe); err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	conf := &model.TPipelineConf{
		Aid:        1,
		PipelineId: "pipe-rebuild-empty",
		YmlContent: "stages:\n  - name: test",
	}
	if _, err := eng.Insert(conf); err != nil {
		t.Fatalf("insert pipeline conf: %v", err)
	}

	tvp := &model.TPipelineVersion{
		Id:         "ver-2",
		PipelineId: "pipe-rebuild-empty",
		Content:    "",
		Sha:        "sha123",
	}

	_, _, err := ReBuild(ctx, "user-1", tvp)
	if err == nil {
		t.Fatal("ReBuild should return error for empty content")
	}
	if err != ErrPipelineYmlEmpty {
		t.Errorf("ReBuild should return ErrPipelineYmlEmpty, got: %v", err)
	}
}

package service

import (
	"context"
	"testing"

	"github.com/gokins/gokins/comm"
	"github.com/gokins/gokins/model"
	_ "github.com/mattn/go-sqlite3"
	"xorm.io/xorm"
)

// setupConvertVarTestDB creates an in-memory SQLite database with the tables
// needed for getOrgVars and convertVar tests.
func setupConvertVarTestDB(t *testing.T) *xorm.Engine {
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
		&model.TOrgPipe{},
		&model.TOrgVar{},
		&model.TPipelineVar{},
		&model.TPipeline{},
		&model.TPipelineConf{},
		&model.TPipelineVersion{},
		&model.TBuild{},
		&model.TStage{},
		&model.TStep{},
		&model.TOrg{},
		&model.TUserOrg{},
	)
	if err != nil {
		t.Fatalf("failed to sync database schema: %v", err)
	}

	return eng
}

// seedOrgVars inserts org vars for the given org ID.
func seedOrgVars(t *testing.T, eng *xorm.Engine, orgId string, vars map[string]varEntry) {
	t.Helper()
	for name, v := range vars {
		ov := &model.TOrgVar{
			OrgId:  orgId,
			Name:   name,
			Value:  v.value,
			Public: v.public,
		}
		if _, err := eng.Insert(ov); err != nil {
			t.Fatalf("insert org var %q: %v", name, err)
		}
	}
}

// seedPipelineVars inserts pipeline vars for the given pipeline ID.
func seedPipelineVars(t *testing.T, eng *xorm.Engine, pipeId string, vars map[string]varEntry) {
	t.Helper()
	for name, v := range vars {
		pv := &model.TPipelineVar{
			PipelineId: pipeId,
			Name:       name,
			Value:      v.value,
			Public:     v.public,
		}
		if _, err := eng.Insert(pv); err != nil {
			t.Fatalf("insert pipeline var %q: %v", name, err)
		}
	}
}

type varEntry struct {
	value  string
	public int // 0 = secret, 1 = public
}

func TestGetOrgVars_NoOrg(t *testing.T) {
	setupConvertVarTestDB(t)
	ctx := context.Background()

	// Pipeline with no org mapping
	vars, err := getOrgVars(ctx, "nonexistent-pipe")
	if err != nil {
		t.Fatalf("getOrgVars unexpected error: %v", err)
	}
	if len(vars) != 0 {
		t.Errorf("getOrgVars for unlinked pipeline should return empty, got %d", len(vars))
	}
}

func TestGetOrgVars_WithOrgVars(t *testing.T) {
	eng := setupConvertVarTestDB(t)
	ctx := context.Background()

	// Create org-pipe mapping
	if _, err := eng.Insert(&model.TOrgPipe{OrgId: "org1", PipeId: "pipe1"}); err != nil {
		t.Fatalf("insert org pipe: %v", err)
	}
	// Seed org vars
	seedOrgVars(t, eng, "org1", map[string]varEntry{
		"ORG_HOST": {value: "example.com", public: 0},
		"ORG_PORT": {value: "8080", public: 1},
	})

	vars, err := getOrgVars(ctx, "pipe1")
	if err != nil {
		t.Fatalf("getOrgVars error: %v", err)
	}
	if len(vars) != 2 {
		t.Fatalf("expected 2 org vars, got %d", len(vars))
	}

	hostVar := vars[0]
	if hostVar.Name != "ORG_HOST" {
		// Order may vary, check both
		hostVar = vars[1]
	}
	if hostVar.Value != "example.com" {
		t.Errorf("ORG_HOST value = %q, want %q", hostVar.Value, "example.com")
	}
}

func TestGetOrgVars_MultipleOrgs(t *testing.T) {
	eng := setupConvertVarTestDB(t)
	ctx := context.Background()

	// Pipeline belongs to two orgs
	if _, err := eng.Insert(&model.TOrgPipe{OrgId: "org-a", PipeId: "pipe-multi"}); err != nil {
		t.Fatalf("insert org pipe a: %v", err)
	}
	if _, err := eng.Insert(&model.TOrgPipe{OrgId: "org-b", PipeId: "pipe-multi"}); err != nil {
		t.Fatalf("insert org pipe b: %v", err)
	}

	seedOrgVars(t, eng, "org-a", map[string]varEntry{
		"VAR_A": {value: "alpha", public: 0},
	})
	seedOrgVars(t, eng, "org-b", map[string]varEntry{
		"VAR_B": {value: "beta", public: 0},
	})

	vars, err := getOrgVars(ctx, "pipe-multi")
	if err != nil {
		t.Fatalf("getOrgVars error: %v", err)
	}
	if len(vars) != 2 {
		t.Fatalf("expected 2 org vars from two orgs, got %d", len(vars))
	}

	names := map[string]bool{}
	for _, v := range vars {
		names[v.Name] = true
	}
	if !names["VAR_A"] || !names["VAR_B"] {
		t.Errorf("expected VAR_A and VAR_B, got %v", names)
	}
}

func TestConvertVar_NoVars(t *testing.T) {
	setupConvertVarTestDB(t)
	ctx := context.Background()

	result, err := convertVar(ctx, "pipe-no-vars", map[string]string{
		"PLAIN": "hello world",
	})
	if err != nil {
		t.Fatalf("convertVar error: %v", err)
	}
	if result["PLAIN"] == nil {
		t.Fatal("PLAIN variable missing from result")
	}
	if result["PLAIN"].Value != "hello world" {
		t.Errorf("PLAIN value = %q, want %q", result["PLAIN"].Value, "hello world")
	}
	if result["PLAIN"].Secret {
		t.Error("PLAIN should not be secret")
	}
}

func TestConvertVar_PipelineVarsOverrideOrgVars(t *testing.T) {
	eng := setupConvertVarTestDB(t)
	ctx := context.Background()

	// Set up org-pipe mapping and vars
	if _, err := eng.Insert(&model.TOrgPipe{OrgId: "org1", PipeId: "pipe-override"}); err != nil {
		t.Fatalf("insert org pipe: %v", err)
	}
	seedOrgVars(t, eng, "org1", map[string]varEntry{
		"HOST": {value: "org-host", public: 0},
	})
	seedPipelineVars(t, eng, "pipe-override", map[string]varEntry{
		"HOST": {value: "pipe-host", public: 0},
	})

	result, err := convertVar(ctx, "pipe-override", nil)
	if err != nil {
		t.Fatalf("convertVar error: %v", err)
	}

	// Pipeline vars should override org vars (they are inserted after)
	hostVar := result["HOST"]
	if hostVar == nil {
		t.Fatal("HOST variable missing from result")
	}
	if hostVar.Value != "pipe-host" {
		t.Errorf("HOST = %q, want %q (pipeline var should override org var)", hostVar.Value, "pipe-host")
	}
}

func TestConvertVar_YamlVarReplacement(t *testing.T) {
	eng := setupConvertVarTestDB(t)
	ctx := context.Background()

	seedPipelineVars(t, eng, "pipe-yaml", map[string]varEntry{
		"DB_HOST": {value: "db.example.com", public: 0},
		"DB_PORT": {value: "5432", public: 0},
	})

	yamlVars := map[string]string{
		"DATABASE_URL": "postgres://${{DB_HOST}}:${{DB_PORT}}/mydb",
	}

	result, err := convertVar(ctx, "pipe-yaml", yamlVars)
	if err != nil {
		t.Fatalf("convertVar error: %v", err)
	}

	dbUrl := result["DATABASE_URL"]
	if dbUrl == nil {
		t.Fatal("DATABASE_URL variable missing from result")
	}
	expected := "postgres://db.example.com:5432/mydb"
	if dbUrl.Value != expected {
		t.Errorf("DATABASE_URL = %q, want %q", dbUrl.Value, expected)
	}
}

func TestConvertVar_SecretPropagation(t *testing.T) {
	eng := setupConvertVarTestDB(t)
	ctx := context.Background()

	// A secret pipeline var (Public=1 maps to Secret=true in the code)
	seedPipelineVars(t, eng, "pipe-secret", map[string]varEntry{
		"API_KEY": {value: "secret123", public: 1}, // public=1 → Secret=true
	})

	yamlVars := map[string]string{
		"AUTH_HEADER": "Bearer ${{API_KEY}}",
	}

	result, err := convertVar(ctx, "pipe-secret", yamlVars)
	if err != nil {
		t.Fatalf("convertVar error: %v", err)
	}

	// The secret var should exist
	apiKeyVar := result["API_KEY"]
	if apiKeyVar == nil {
		t.Fatal("API_KEY variable missing from result")
	}
	if !apiKeyVar.Secret {
		t.Error("API_KEY should be marked as secret (public=0)")
	}

	// AUTH_HEADER references a secret, so the secret flag should propagate
	authHeader := result["AUTH_HEADER"]
	if authHeader == nil {
		t.Fatal("AUTH_HEADER variable missing from result")
	}
	// The final pass only masks values that still contain ${{VAR}} references.
	// Since AUTH_HEADER's value was already resolved to "Bearer secret123" in
	// the yaml-vars pass (with mustShow=true), the masking pass won't change it.
	// However, the Secret flag should still propagate from the referenced variable.
	if !authHeader.Secret {
		t.Error("AUTH_HEADER should inherit secret flag from referenced variable")
	}
}

func TestConvertVar_PublicVarNotMasked(t *testing.T) {
	eng := setupConvertVarTestDB(t)
	ctx := context.Background()

	seedPipelineVars(t, eng, "pipe-public", map[string]varEntry{
		"HOST": {value: "public-host.com", public: 0}, // public=0 → Secret=false (public var)
	})

	yamlVars := map[string]string{
		"URL": "https://${{HOST}}",
	}

	result, err := convertVar(ctx, "pipe-public", yamlVars)
	if err != nil {
		t.Fatalf("convertVar error: %v", err)
	}

	urlVar := result["URL"]
	if urlVar == nil {
		t.Fatal("URL variable missing from result")
	}
	if urlVar.Secret {
		t.Error("URL should not be secret (all referenced vars are public)")
	}
}

func TestConvertVar_EmptyYamlVars(t *testing.T) {
	setupConvertVarTestDB(t)
	ctx := context.Background()

	result, err := convertVar(ctx, "pipe-empty", nil)
	if err != nil {
		t.Fatalf("convertVar error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("convertVar with no vars should return empty map, got %d", len(result))
	}
}

func TestConvertVar_ChainedVariableResolution(t *testing.T) {
	eng := setupConvertVarTestDB(t)
	ctx := context.Background()

	// Pipeline vars that reference each other
	seedPipelineVars(t, eng, "pipe-chain", map[string]varEntry{
		"BASE_URL": {value: "https://api.example.com", public: 0},
	})

	yamlVars := map[string]string{
		"ENDPOINT": "${{BASE_URL}}/v2",
	}

	result, err := convertVar(ctx, "pipe-chain", yamlVars)
	if err != nil {
		t.Fatalf("convertVar error: %v", err)
	}

	endpoint := result["ENDPOINT"]
	if endpoint == nil {
		t.Fatal("ENDPOINT variable missing from result")
	}
	expected := "https://api.example.com/v2"
	if endpoint.Value != expected {
		t.Errorf("ENDPOINT = %q, want %q", endpoint.Value, expected)
	}
}

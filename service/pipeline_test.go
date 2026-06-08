package service

import (
	"context"
	"testing"

	"github.com/gokins/core/runtime"
	"github.com/gokins/gokins/bean"
)

func TestReplace_EmptyString(t *testing.T) {
	result, secret := replace("", nil)
	if result != "" {
		t.Errorf("replace(\"\", nil) = %q, want \"\"", result)
	}
	if secret {
		t.Error("replace(\"\", nil) secret = true, want false")
	}
}

func TestReplace_NoVariables(t *testing.T) {
	result, secret := replace("hello world", nil)
	if result != "hello world" {
		t.Errorf("replace(\"hello world\", nil) = %q, want \"hello world\"", result)
	}
	if secret {
		t.Error("replace with no vars should not be secret")
	}
}

func TestReplace_SingleVariable(t *testing.T) {
	vars := map[string]*runtime.Variables{
		"FOO": {Name: "FOO", Value: "bar", Secret: false},
	}
	result, secret := replace("${{FOO}}", vars)
	if result != "bar" {
		t.Errorf("replace(\"${{FOO}}\") = %q, want \"bar\"", result)
	}
	if secret {
		t.Error("non-secret var should not set secret=true")
	}
}

func TestReplace_MultipleVariables(t *testing.T) {
	vars := map[string]*runtime.Variables{
		"HOST": {Name: "HOST", Value: "localhost", Secret: false},
		"PORT": {Name: "PORT", Value: "8080", Secret: false},
	}
	result, secret := replace("http://${{HOST}}:${{PORT}}/api", vars)
	expected := "http://localhost:8080/api"
	if result != expected {
		t.Errorf("replace multiple vars = %q, want %q", result, expected)
	}
	if secret {
		t.Error("non-secret vars should not set secret=true")
	}
}

func TestReplace_SecretVariable(t *testing.T) {
	vars := map[string]*runtime.Variables{
		"TOKEN": {Name: "TOKEN", Value: "secret123", Secret: true},
	}
	// Without mustShow, secret values are masked
	result, secret := replace("Bearer ${{TOKEN}}", vars)
	if result != "Bearer ***" {
		t.Errorf("replace secret (no mustShow) = %q, want \"Bearer ***\"", result)
	}
	if !secret {
		t.Error("secret var should set secret=true")
	}
}

func TestReplace_SecretVariableMustShow(t *testing.T) {
	vars := map[string]*runtime.Variables{
		"TOKEN": {Name: "TOKEN", Value: "secret123", Secret: true},
	}
	// With mustShow=true, secret values are revealed
	result, secret := replace("Bearer ${{TOKEN}}", vars, true)
	if result != "Bearer secret123" {
		t.Errorf("replace secret (mustShow) = %q, want \"Bearer secret123\"", result)
	}
	if !secret {
		t.Error("secret var should set secret=true even with mustShow")
	}
}

func TestReplace_MissingVariable(t *testing.T) {
	vars := map[string]*runtime.Variables{
		"FOO": {Name: "FOO", Value: "bar", Secret: false},
	}
	result, _ := replace("${{MISSING}}", vars)
	if result != "" {
		t.Errorf("replace missing var = %q, want \"\"", result)
	}
}

func TestReplace_MixedSecretAndNonSecret(t *testing.T) {
	vars := map[string]*runtime.Variables{
		"USER": {Name: "USER", Value: "admin", Secret: false},
		"PASS": {Name: "PASS", Value: "s3cret", Secret: true},
	}
	result, secret := replace("user=${{USER}} pass=${{PASS}}", vars)
	expected := "user=admin pass=***"
	if result != expected {
		t.Errorf("replace mixed = %q, want %q", result, expected)
	}
	if !secret {
		t.Error("mixed with secret var should set secret=true")
	}
}

func TestReplace_RepeatedVariable(t *testing.T) {
	vars := map[string]*runtime.Variables{
		"X": {Name: "X", Value: "abc", Secret: false},
	}
	result, _ := replace("${{X}}-${{X}}-${{X}}", vars)
	if result != "abc-abc-abc" {
		t.Errorf("replace repeated = %q, want \"abc-abc-abc\"", result)
	}
}

func TestReplace_WhitespaceInBraces(t *testing.T) {
	vars := map[string]*runtime.Variables{
		"FOO": {Name: "FOO", Value: "bar", Secret: false},
	}
	result, _ := replace("${{ FOO }}", vars)
	if result != "bar" {
		t.Errorf("replace with whitespace = %q, want \"bar\"", result)
	}
}

func TestReplace_PlainTextPreserved(t *testing.T) {
	vars := map[string]*runtime.Variables{
		"FOO": {Name: "FOO", Value: "bar", Secret: false},
	}
	result, _ := replace("no vars here", vars)
	if result != "no vars here" {
		t.Errorf("replace plain text = %q, want \"no vars here\"", result)
	}
}

func TestReplace_LiteralDollarBraces(t *testing.T) {
	// Text with {{ but no $ prefix should not be treated as variable
	result, _ := replace("{{NOTVAR}}", nil)
	if result != "{{NOTVAR}}" {
		t.Errorf("replace non-var braces = %q, want \"{{NOTVAR}}\"", result)
	}
}

func TestReplaceMaps_Empty(t *testing.T) {
	result := replaceMaps(nil, nil)
	if len(result) != 0 {
		t.Errorf("replaceMaps(nil, nil) should return empty map, got %v", result)
	}
}

func TestReplaceMaps_BasicReplacement(t *testing.T) {
	vars := map[string]*runtime.Variables{
		"DB_HOST": {Name: "DB_HOST", Value: "db.example.com", Secret: false},
		"DB_PASS": {Name: "DB_PASS", Value: "password", Secret: true},
	}
	envs := map[string]string{
		"DATABASE_URL": "postgres://${{DB_HOST}}:5432/mydb",
		"PASSWORD":     "${{DB_PASS}}",
	}
	result := replaceMaps(envs, vars)
	if result["DATABASE_URL"] != "postgres://db.example.com:5432/mydb" {
		t.Errorf("DATABASE_URL = %q, want \"postgres://db.example.com:5432/mydb\"", result["DATABASE_URL"])
	}
	// replaceMaps calls replace with mustShow=true, so secrets should be shown
	if result["PASSWORD"] != "password" {
		t.Errorf("PASSWORD = %q, want \"password\"", result["PASSWORD"])
	}
}

func TestReplaceMaps_PreservesNonVariableKeys(t *testing.T) {
	vars := map[string]*runtime.Variables{
		"X": {Name: "X", Value: "replaced", Secret: false},
	}
	envs := map[string]string{
		"PLAIN": "no vars",
		"WITH":  "${{X}}",
	}
	result := replaceMaps(envs, vars)
	if result["PLAIN"] != "no vars" {
		t.Errorf("PLAIN = %q, want \"no vars\"", result["PLAIN"])
	}
	if result["WITH"] != "replaced" {
		t.Errorf("WITH = %q, want \"replaced\"", result["WITH"])
	}
}

func TestReplaceStage_BasicReplacement(t *testing.T) {
	vars := map[string]*runtime.Variables{
		"REPO":  {Name: "REPO", Value: "https://github.com/test/repo", Secret: false},
		"STAGE": {Name: "STAGE", Value: "build", Secret: false},
		"IMG":   {Name: "IMG", Value: "golang:1.22", Secret: false},
	}
	stage := &bean.Stage{
		Stage:       "${{STAGE}}-stage",
		Name:        "${{STAGE}}",
		DisplayName: "${{STAGE}} Stage",
		Repo:        "${{REPO}}",
		Steps: []*bean.Step{
			{
				Step:        "${{STAGE}}-step",
				Name:        "${{STAGE}}-step",
				DisplayName: "${{STAGE}} Step",
				Image:       "${{IMG}}",
			},
		},
	}
	replaceStage(stage, vars)

	if stage.Stage != "build-stage" {
		t.Errorf("stage.Stage = %q, want \"build-stage\"", stage.Stage)
	}
	if stage.Name != "build" {
		t.Errorf("stage.Name = %q, want \"build\"", stage.Name)
	}
	if stage.DisplayName != "build Stage" {
		t.Errorf("stage.DisplayName = %q, want \"build Stage\"", stage.DisplayName)
	}
	if stage.Repo != "https://github.com/test/repo" {
		t.Errorf("stage.Repo = %q, want repo URL", stage.Repo)
	}
	if len(stage.Steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(stage.Steps))
	}
	step := stage.Steps[0]
	if step.Step != "build-step" {
		t.Errorf("step.Step = %q, want \"build-step\"", step.Step)
	}
	if step.Image != "golang:1.22" {
		t.Errorf("step.Image = %q, want \"golang:1.22\"", step.Image)
	}
}

func TestReplaceStage_StepEnvAndInput(t *testing.T) {
	vars := map[string]*runtime.Variables{
		"API_KEY": {Name: "API_KEY", Value: "key123", Secret: true},
		"ENV":     {Name: "ENV", Value: "production", Secret: false},
	}
	stage := &bean.Stage{
		Stage: "deploy",
		Name:  "deploy",
		Steps: []*bean.Step{
			{
				Step: "deploy-step",
				Name: "deploy-step",
				Env: map[string]string{
					"APP_ENV": "${{ENV}}",
				},
				Input: map[string]string{
					"token": "${{API_KEY}}",
				},
			},
		},
	}
	replaceStage(stage, vars)

	step := stage.Steps[0]
	if step.Env["APP_ENV"] != "production" {
		t.Errorf("step.Env[APP_ENV] = %q, want \"production\"", step.Env["APP_ENV"])
	}
	// Input uses replaceMaps which passes mustShow=true, so secret values are shown
	if step.Input["token"] != "key123" {
		t.Errorf("step.Input[token] = %q, want \"key123\"", step.Input["token"])
	}
}

func TestReplaceStage_EmptySteps(t *testing.T) {
	vars := map[string]*runtime.Variables{
		"X": {Name: "X", Value: "y", Secret: false},
	}
	stage := &bean.Stage{
		Stage: "${{X}}",
		Name:  "${{X}}",
		Steps: nil,
	}
	replaceStage(stage, vars)
	if stage.Stage != "y" {
		t.Errorf("stage.Stage = %q, want \"y\"", stage.Stage)
	}
}

func TestReplaceStages_MultipleStages(t *testing.T) {
	vars := map[string]*runtime.Variables{
		"VER": {Name: "VER", Value: "v2", Secret: false},
	}
	stages := []*bean.Stage{
		{Stage: "build-${{VER}}", Name: "build-${{VER}}"},
		{Stage: "test-${{VER}}", Name: "test-${{VER}}"},
		{Stage: "deploy-${{VER}}", Name: "deploy-${{VER}}"},
	}
	replaceStages(stages, vars)

	expected := []string{"build-v2", "test-v2", "deploy-v2"}
	for i, s := range stages {
		if s.Stage != expected[i] {
			t.Errorf("stages[%d].Stage = %q, want %q", i, s.Stage, expected[i])
		}
	}
}

func TestReplaceStages_NilInput(t *testing.T) {
	// Should not panic with nil stages
	replaceStages(nil, nil)
}

func TestBatchBuildCounts_EmptyInput(t *testing.T) {
	result, err := BatchBuildCounts(context.Background(), nil)
	if err != nil {
		t.Fatalf("BatchBuildCounts(nil) error = %v", err)
	}
	if len(result) != 0 {
		t.Errorf("BatchBuildCounts(nil) should return empty map, got %v", result)
	}

	result, err = BatchBuildCounts(context.Background(), []string{})
	if err != nil {
		t.Fatalf("BatchBuildCounts([]) error = %v", err)
	}
	if len(result) != 0 {
		t.Errorf("BatchBuildCounts([]) should return empty map, got %v", result)
	}
}

func TestBatchLatestBuilds_EmptyInput(t *testing.T) {
	result, err := BatchLatestBuilds(context.Background(), nil)
	if err != nil {
		t.Fatalf("BatchLatestBuilds(nil) error = %v", err)
	}
	if len(result) != 0 {
		t.Errorf("BatchLatestBuilds(nil) should return empty map, got %v", result)
	}

	result, err = BatchLatestBuilds(context.Background(), []string{})
	if err != nil {
		t.Fatalf("BatchLatestBuilds([]) error = %v", err)
	}
	if len(result) != 0 {
		t.Errorf("BatchLatestBuilds([]) should return empty map, got %v", result)
	}
}

func TestBatchLatestBuildsForVersions_EmptyInput(t *testing.T) {
	result, err := BatchLatestBuildsForVersions(context.Background(), nil)
	if err != nil {
		t.Fatalf("BatchLatestBuildsForVersions(nil) error = %v", err)
	}
	if len(result) != 0 {
		t.Errorf("BatchLatestBuildsForVersions(nil) should return empty map, got %v", result)
	}

	result, err = BatchLatestBuildsForVersions(context.Background(), []string{})
	if err != nil {
		t.Fatalf("BatchLatestBuildsForVersions([]) error = %v", err)
	}
	if len(result) != 0 {
		t.Errorf("BatchLatestBuildsForVersions([]) should return empty map, got %v", result)
	}
}

func TestBatchGetUsers_EmptyInput(t *testing.T) {
	result, err := BatchGetUsers(context.Background(), nil)
	if err != nil {
		t.Fatalf("BatchGetUsers(nil) error = %v", err)
	}
	if len(result) != 0 {
		t.Errorf("BatchGetUsers(nil) should return empty map, got %v", result)
	}

	result, err = BatchGetUsers(context.Background(), []string{})
	if err != nil {
		t.Fatalf("BatchGetUsers([]) error = %v", err)
	}
	if len(result) != 0 {
		t.Errorf("BatchGetUsers([]) should return empty map, got %v", result)
	}
}

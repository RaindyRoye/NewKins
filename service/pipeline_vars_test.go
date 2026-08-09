package service

import (
	"testing"

	"github.com/gokins/core/runtime"
	"github.com/gokins/gokins/bean"
	"github.com/gokins/gokins/comm"
)

// TestReplace_SecretWithMaskedValue tests that secret variables use the MaskedValue constant
func TestReplace_SecretWithMaskedValue(t *testing.T) {
	vars := map[string]*runtime.Variables{
		"SECRET": {Name: "SECRET", Value: "sensitive", Secret: true},
	}
	result, secret := replace("Value: ${{SECRET}}", vars)
	expected := "Value: " + comm.MaskedValue
	if result != expected {
		t.Errorf("replace(secret) = %q, want %q", result, expected)
	}
	if !secret {
		t.Error("secret variable should mark result as secret")
	}
}

// TestReplace_UndefinedVariable tests behavior when a variable is not defined
func TestReplace_UndefinedVariable(t *testing.T) {
	vars := map[string]*runtime.Variables{}
	result, secret := replace("Hello ${{UNDEFINED}}!", vars)
	if result != "Hello !" {
		t.Errorf("replace(undefined) = %q, want \"Hello !\"", result)
	}
	if secret {
		t.Error("undefined variable should not mark result as secret")
	}
}

// TestReplace_MultipleOccurrences tests replacing the same variable multiple times
func TestReplace_MultipleOccurrences(t *testing.T) {
	vars := map[string]*runtime.Variables{
		"NAME": {Name: "NAME", Value: "Alice", Secret: false},
	}
	result, _ := replace("${{NAME}} and ${{NAME}}", vars)
	expected := "Alice and Alice"
	if result != expected {
		t.Errorf("replace(multiple occurrences) = %q, want %q", result, expected)
	}
}

// TestReplaceMaps_WithSecret tests replaceMaps with secret variables
func TestReplaceMaps_WithSecret(t *testing.T) {
	vars := map[string]*runtime.Variables{
		"API_KEY": {Name: "API_KEY", Value: "secret-key-123", Secret: true},
		"HOST":    {Name: "HOST", Value: "api.example.com", Secret: false},
	}
	envs := map[string]string{
		"API_HOST": "${{HOST}}",
		"API_KEY":  "${{API_KEY}}",
	}
	result := replaceMaps(envs, vars)
	if result["API_HOST"] != "api.example.com" {
		t.Errorf("replaceMaps API_HOST = %q, want \"api.example.com\"", result["API_HOST"])
	}
	// Secret values should be masked when mustShow is not passed (defaults to true in replaceMaps)
	if result["API_KEY"] != "secret-key-123" {
		t.Errorf("replaceMaps API_KEY = %q, want \"secret-key-123\" (mustShow=true in replaceMaps)", result["API_KEY"])
	}
}

// TestReplaceStep_AllFields tests replaceStep with all replaceable fields
func TestReplaceStep_AllFields(t *testing.T) {
	vars := map[string]*runtime.Variables{
		"PLUGIN":  {Name: "PLUGIN", Value: "docker-build", Secret: false},
		"IMAGE":   {Name: "IMAGE", Value: "golang:1.22", Secret: false},
		"REPO":    {Name: "REPO", Value: "https://github.com/example/repo", Secret: false},
		"DISPLAY": {Name: "DISPLAY", Value: "Build Step", Secret: false},
	}
	step := &bean.Step{
		Step:        "${{PLUGIN}}",
		Name:        "step-${{PLUGIN}}",
		DisplayName: "${{DISPLAY}}",
		Image:       "${{IMAGE}}",
		Repo:        "${{REPO}}",
		Env: map[string]string{
			"GO_IMAGE": "${{IMAGE}}",
		},
		Input: map[string]string{
			"repo_url": "${{REPO}}",
		},
	}
	replaceStep(step, vars)

	if step.Step != "docker-build" {
		t.Errorf("step.Step = %q, want \"docker-build\"", step.Step)
	}
	if step.Name != "step-docker-build" {
		t.Errorf("step.Name = %q, want \"step-docker-build\"", step.Name)
	}
	if step.DisplayName != "Build Step" {
		t.Errorf("step.DisplayName = %q, want \"Build Step\"", step.DisplayName)
	}
	if step.Image != "golang:1.22" {
		t.Errorf("step.Image = %q, want \"golang:1.22\"", step.Image)
	}
	if step.Repo != "https://github.com/example/repo" {
		t.Errorf("step.Repo = %q, want \"https://github.com/example/repo\"", step.Repo)
	}
	if step.Env["GO_IMAGE"] != "golang:1.22" {
		t.Errorf("step.Env[GO_IMAGE] = %q, want \"golang:1.22\"", step.Env["GO_IMAGE"])
	}
	if step.Input["repo_url"] != "https://github.com/example/repo" {
		t.Errorf("step.Input[repo_url] = %q, want \"https://github.com/example/repo\"", step.Input["repo_url"])
	}
}

// TestReplaceSteps_MultipleSteps tests replacing variables in multiple steps
func TestReplaceSteps_MultipleSteps(t *testing.T) {
	vars := map[string]*runtime.Variables{
		"VERSION": {Name: "VERSION", Value: "v1.0.0", Secret: false},
	}
	steps := []*bean.Step{
		{Name: "build-${{VERSION}}"},
		{Name: "test-${{VERSION}}"},
		{Name: "deploy-${{VERSION}}"},
	}
	replaceSteps(steps, vars)

	if steps[0].Name != "build-v1.0.0" {
		t.Errorf("steps[0].Name = %q, want \"build-v1.0.0\"", steps[0].Name)
	}
	if steps[1].Name != "test-v1.0.0" {
		t.Errorf("steps[1].Name = %q, want \"test-v1.0.0\"", steps[1].Name)
	}
	if steps[2].Name != "deploy-v1.0.0" {
		t.Errorf("steps[2].Name = %q, want \"deploy-v1.0.0\"", steps[2].Name)
	}
}

// TestReplaceStages_MultipleStagesComprehensive tests replacing variables in multiple stages
func TestReplaceStages_MultipleStagesComprehensive(t *testing.T) {
	vars := map[string]*runtime.Variables{
		"ENV": {Name: "ENV", Value: "production", Secret: false},
	}
	stages := []*bean.Stage{
		{Stage: "build", Name: "Build", DisplayName: "Build for ${{ENV}}"},
		{Stage: "test", Name: "Test", DisplayName: "Test for ${{ENV}}"},
		{Stage: "deploy", Name: "Deploy", DisplayName: "Deploy to ${{ENV}}"},
	}
	replaceStages(stages, vars)

	if stages[0].DisplayName != "Build for production" {
		t.Errorf("stages[0].DisplayName = %q, want \"Build for production\"", stages[0].DisplayName)
	}
	if stages[1].DisplayName != "Test for production" {
		t.Errorf("stages[1].DisplayName = %q, want \"Test for production\"", stages[1].DisplayName)
	}
	if stages[2].DisplayName != "Deploy to production" {
		t.Errorf("stages[2].DisplayName = %q, want \"Deploy to production\"", stages[2].DisplayName)
	}
}

// TestReplaceStage_WithSteps tests that replaceStage also replaces variables in nested steps
func TestReplaceStage_WithSteps(t *testing.T) {
	vars := map[string]*runtime.Variables{
		"IMAGE": {Name: "IMAGE", Value: "node:18", Secret: false},
	}
	stage := &bean.Stage{
		Stage:       "build",
		Name:        "Build Stage",
		DisplayName: "Build with ${{IMAGE}}",
		Steps: []*bean.Step{
			{
				Step:  "npm-build",
				Name:  "Build Step",
				Image: "${{IMAGE}}",
			},
		},
	}
	replaceStage(stage, vars)

	if stage.DisplayName != "Build with node:18" {
		t.Errorf("stage.DisplayName = %q, want \"Build with node:18\"", stage.DisplayName)
	}
	if len(stage.Steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(stage.Steps))
	}
	if stage.Steps[0].Image != "node:18" {
		t.Errorf("stage.Steps[0].Image = %q, want \"node:18\"", stage.Steps[0].Image)
	}
}

// TestReplace_WithEmptyVarsMap tests behavior with an empty variables map
func TestReplace_WithEmptyVarsMap(t *testing.T) {
	vars := map[string]*runtime.Variables{}
	result, secret := replace("Hello ${{NAME}}", vars)
	if result != "Hello " {
		t.Errorf("replace(empty vars) = %q, want \"Hello \"", result)
	}
	if secret {
		t.Error("empty vars map should not mark result as secret")
	}
}

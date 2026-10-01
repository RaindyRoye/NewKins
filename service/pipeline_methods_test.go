package service

import (
	"testing"

	"github.com/gokins/core/runtime"
	"github.com/gokins/gokins/bean"
	"github.com/gokins/gokins/comm"
)

// TestReplaceFunction tests the variable replacement logic
func TestReplaceFunction(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		vars       map[string]*runtime.Variables
		mustShow   bool
		expected   string
		secretFlag bool
	}{
		{
			name:     "empty string",
			input:    "",
			vars:     map[string]*runtime.Variables{},
			expected: "",
		},
		{
			name:     "no variables",
			input:    "plain text",
			vars:     map[string]*runtime.Variables{},
			expected: "plain text",
		},
		{
			name:  "single variable replacement",
			input: "Hello ${{NAME}}",
			vars: map[string]*runtime.Variables{
				"NAME": {Name: "NAME", Value: "World", Secret: false},
			},
			expected: "Hello World",
		},
		{
			name:  "multiple variables",
			input: "${{GREETING}} ${{NAME}}, welcome to ${{PLACE}}",
			vars: map[string]*runtime.Variables{
				"GREETING": {Name: "GREETING", Value: "Hi", Secret: false},
				"NAME":     {Name: "NAME", Value: "Alice", Secret: false},
				"PLACE":    {Name: "PLACE", Value: "Gokins", Secret: false},
			},
			expected: "Hi Alice, welcome to Gokins",
		},
		{
			name:  "secret variable masked",
			input: "Password: ${{PASS}}",
			vars: map[string]*runtime.Variables{
				"PASS": {Name: "PASS", Value: "secret123", Secret: true},
			},
			mustShow:   false,
			expected:   "Password: " + comm.MaskedValue,
			secretFlag: true,
		},
		{
			name:  "secret variable mustShow=true",
			input: "Password: ${{PASS}}",
			vars: map[string]*runtime.Variables{
				"PASS": {Name: "PASS", Value: "secret123", Secret: true},
			},
			mustShow:   true,
			expected:   "Password: secret123",
			secretFlag: true,
		},
		{
			name:  "mixed secret and non-secret",
			input: "User: ${{USER}}, Pass: ${{PASS}}",
			vars: map[string]*runtime.Variables{
				"USER": {Name: "USER", Value: "admin", Secret: false},
				"PASS": {Name: "PASS", Value: "pass123", Secret: true},
			},
			mustShow:   false,
			expected:   "User: admin, Pass: " + comm.MaskedValue,
			secretFlag: true,
		},
		{
			name:     "undefined variable",
			input:    "Value: ${{UNDEFINED}}",
			vars:     map[string]*runtime.Variables{},
			expected: "Value: ",
		},
		{
			name:  "variable with special characters",
			input: "URL: ${{URL}}",
			vars: map[string]*runtime.Variables{
				"URL": {Name: "URL", Value: "https://example.com/path?query=1", Secret: false},
			},
			expected: "URL: https://example.com/path?query=1",
		},
		{
			name:  "repeated variable usage",
			input: "${{VAR}} and ${{VAR}} again",
			vars: map[string]*runtime.Variables{
				"VAR": {Name: "VAR", Value: "test", Secret: false},
			},
			expected: "test and test again",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, secret := replace(tt.input, tt.vars, tt.mustShow)
			if result != tt.expected {
				t.Errorf("replace() result = %q, want %q", result, tt.expected)
			}
			if secret != tt.secretFlag {
				t.Errorf("replace() secret = %v, want %v", secret, tt.secretFlag)
			}
		})
	}
}

// TestReplaceStages tests variable replacement in pipeline stages
func TestReplaceStages(t *testing.T) {
	stages := []*bean.Stage{
		{
			Name:        "build-${{ENV}}",
			Stage:       "build-${{ENV}}-stage",
			DisplayName: "Build ${{ENV}}",
			Repo:        "${{REPO_URL}}",
			Steps: []*bean.Step{
				{
					Name:        "compile-${{LANG}}",
					Step:        "compile-${{LANG}}-step",
					DisplayName: "Compile ${{LANG}}",
					Image:       "${{IMAGE}}",
					Env: map[string]string{
						"BUILD_ENV": "${{ENV}}",
						"LANG":      "${{LANG}}",
					},
				},
			},
		},
	}

	vars := map[string]*runtime.Variables{
		"ENV":      {Name: "ENV", Value: "production", Secret: false},
		"REPO_URL": {Name: "REPO_URL", Value: "https://github.com/test/repo", Secret: false},
		"LANG":     {Name: "LANG", Value: "go", Secret: false},
		"IMAGE":    {Name: "IMAGE", Value: "golang:1.22", Secret: false},
	}

	replaceStages(stages, vars)

	// Verify stage replacements
	if stages[0].Name != "build-production" {
		t.Errorf("stage name replacement failed: got %q", stages[0].Name)
	}
	if stages[0].Stage != "build-production-stage" {
		t.Errorf("stage identifier replacement failed: got %q", stages[0].Stage)
	}
	if stages[0].DisplayName != "Build production" {
		t.Errorf("stage display name replacement failed: got %q", stages[0].DisplayName)
	}
	if stages[0].Repo != "https://github.com/test/repo" {
		t.Errorf("stage repo replacement failed: got %q", stages[0].Repo)
	}

	// Verify step replacements
	step := stages[0].Steps[0]
	if step.Name != "compile-go" {
		t.Errorf("step name replacement failed: got %q", step.Name)
	}
	if step.Step != "compile-go-step" {
		t.Errorf("step identifier replacement failed: got %q", step.Step)
	}
	if step.DisplayName != "Compile go" {
		t.Errorf("step display name replacement failed: got %q", step.DisplayName)
	}
	if step.Image != "golang:1.22" {
		t.Errorf("step image replacement failed: got %q", step.Image)
	}

	// Verify env replacements
	if step.Env["BUILD_ENV"] != "production" {
		t.Errorf("step env replacement failed: got %q", step.Env["BUILD_ENV"])
	}
	if step.Env["LANG"] != "go" {
		t.Errorf("step env LANG replacement failed: got %q", step.Env["LANG"])
	}
}

// TestReplaceMaps tests variable replacement in environment maps
func TestReplaceMaps(t *testing.T) {
	envs := map[string]string{
		"DB_HOST":     "${{HOST}}",
		"DB_PORT":     "${{PORT}}",
		"DB_PASSWORD": "${{PASS}}",
		"STATIC":      "no-variables",
	}

	vars := map[string]*runtime.Variables{
		"HOST": {Name: "HOST", Value: "localhost", Secret: false},
		"PORT": {Name: "PORT", Value: "5432", Secret: false},
		"PASS": {Name: "PASS", Value: "secret", Secret: true},
	}

	result := replaceMaps(envs, vars)

	if result["DB_HOST"] != "localhost" {
		t.Errorf("DB_HOST replacement failed: got %q", result["DB_HOST"])
	}
	if result["DB_PORT"] != "5432" {
		t.Errorf("DB_PORT replacement failed: got %q", result["DB_PORT"])
	}
	// Secrets should be masked in replaceMaps (mustShow=true passed internally)
	if result["DB_PASSWORD"] != "secret" {
		t.Errorf("DB_PASSWORD should be revealed (mustShow=true): got %q", result["DB_PASSWORD"])
	}
	if result["STATIC"] != "no-variables" {
		t.Errorf("STATIC value should remain unchanged: got %q", result["STATIC"])
	}
}

// CheckUPermission tests already exist in perms_test.go

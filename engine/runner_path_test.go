package engine

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidatePath_ValidPaths(t *testing.T) {
	tests := []struct {
		name      string
		base      string
		requested string
	}{
		{
			name:      "simple filename",
			base:      "/home/user/build",
			requested: "file.txt",
		},
		{
			name:      "subdirectory file",
			base:      "/home/user/build",
			requested: "subdir/file.txt",
		},
		{
			name:      "deep nested path",
			base:      "/home/user/build",
			requested: "a/b/c/d/file.txt",
		},
		{
			name:      "current directory reference",
			base:      "/home/user/build",
			requested: "./file.txt",
		},
		{
			name:      "mixed current directory",
			base:      "/home/user/build",
			requested: "./subdir/./file.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := validatePath(tt.base, tt.requested)
			if err != nil {
				t.Errorf("validatePath() error = %v, want nil", err)
				return
			}
			if result == "" {
				t.Error("validatePath() returned empty result")
			}
			// Verify result is under base
			if !strings.HasPrefix(result, tt.base) {
				t.Errorf("validatePath() result %q is not under base %q", result, tt.base)
			}
		})
	}
}

func TestValidatePath_TraversalAttacks(t *testing.T) {
	tests := []struct {
		name      string
		base      string
		requested string
	}{
		{
			name:      "simple parent traversal",
			base:      "/home/user/build",
			requested: "../etc/passwd",
		},
		{
			name:      "double parent traversal",
			base:      "/home/user/build",
			requested: "../../etc/passwd",
		},
		{
			name:      "triple parent traversal",
			base:      "/home/user/build",
			requested: "../../../etc/passwd",
		},
		{
			name:      "hidden parent in middle",
			base:      "/home/user/build",
			requested: "subdir/../../etc/passwd",
		},
		{
			name:      "multiple parents mixed with current",
			base:      "/home/user/build",
			requested: "./subdir/../../../etc/passwd",
		},
		{
			name:      "absolute path attack",
			base:      "/home/user/build",
			requested: "/etc/passwd",
		},
		{
			name:      "absolute path with parent",
			base:      "/home/user/build",
			requested: "/home/user/../../etc/passwd",
		},
		{
			name:      "backslash traversal on Unix",
			base:      "/home/user/build",
			requested: "..\\etc\\passwd",
		},
		{
			name:      "double dot only",
			base:      "/home/user/build",
			requested: "..",
		},
		{
			name:      "double dot with slash",
			base:      "/home/user/build",
			requested: "../",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := validatePath(tt.base, tt.requested)
			if err == nil {
				t.Errorf("validatePath() should reject %q but returned %q", tt.requested, result)
				return
			}
			if !errors.Is(err, ErrPathTraversal) {
				t.Errorf("validatePath() error = %v, want ErrPathTraversal", err)
			}
			// Ensure no traversal patterns in result
			if result != "" && (strings.Contains(result, "..") || !strings.HasPrefix(result, tt.base)) {
				t.Errorf("validatePath() unsafe result %q for input %q", result, tt.requested)
			}
		})
	}
}

func TestValidatePath_SafeEdgeCases(t *testing.T) {
	// These paths look suspicious but actually resolve safely within base
	tests := []struct {
		name      string
		base      string
		requested string
	}{
		{
			name:      "parent after valid path resolves within base",
			base:      "/home/user/build",
			requested: "valid/../escape",
		},
		{
			name:      "null byte in middle - Go cleans to safe path",
			base:      "/home/user/build",
			requested: "file.txt\x00../../etc/passwd",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := validatePath(tt.base, tt.requested)
			if err != nil {
				t.Errorf("validatePath() should accept %q but got error: %v", tt.requested, err)
				return
			}
			if !strings.HasPrefix(result, tt.base) {
				t.Errorf("validatePath() result %q not under base %q", result, tt.base)
			}
		})
	}
}

func TestValidatePath_EdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		base      string
		requested string
		wantErr   bool
	}{
		{
			name:      "empty requested path",
			base:      "/home/user/build",
			requested: "",
			wantErr:   false, // Empty becomes current dir
		},
		{
			name:      "single dot",
			base:      "/home/user/build",
			requested: ".",
			wantErr:   false,
		},
		{
			name:      "trailing slash",
			base:      "/home/user/build",
			requested: "subdir/",
			wantErr:   false,
		},
		{
			name:      "multiple slashes",
			base:      "/home/user/build",
			requested: "subdir///file.txt",
			wantErr:   false,
		},
		{
			name:      "base with trailing slash",
			base:      "/home/user/build/",
			requested: "file.txt",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := validatePath(tt.base, tt.requested)
			if tt.wantErr {
				if err == nil {
					t.Errorf("validatePath() expected error but got result %q", result)
				}
			} else {
				if err != nil {
					t.Errorf("validatePath() unexpected error = %v", err)
				}
				if result == "" && tt.requested != "" && tt.requested != "." && tt.requested != ".." {
					t.Error("validatePath() returned empty result")
				}
			}
		})
	}
}

func TestValidatePath_RelativeBase(t *testing.T) {
	// Test with relative base path
	base := "build"
	requested := "subdir/file.txt"
	
	result, err := validatePath(base, requested)
	if err != nil {
		t.Errorf("validatePath() error = %v", err)
		return
	}
	
	expected := filepath.Join(base, requested)
	if result != expected {
		t.Errorf("validatePath() = %q, want %q", result, expected)
	}
}

func TestValidatePath_RealWorldScenarios(t *testing.T) {
	tests := []struct {
		name      string
		base      string
		requested string
		wantErr   bool
	}{
		{
			name:      "build artifacts - valid",
			base:      "/var/lib/gokins/builds/build123/artifacts",
			requested: "dist/app.jar",
			wantErr:   false,
		},
		{
			name:      "build artifacts - escape attempt",
			base:      "/var/lib/gokins/builds/build123/artifacts",
			requested: "../../../etc/passwd",
			wantErr:   true,
		},
		{
			name:      "workspace - valid nested",
			base:      "/workspace",
			requested: "project/src/main/java/App.java",
			wantErr:   false,
		},
		{
			name:      "workspace - escape via parent",
			base:      "/workspace",
			requested: "../other-project/secret.txt",
			wantErr:   true,
		},
		{
			name:      "home directory - valid",
			base:      "/home/runner/build",
			requested: "output/results.xml",
			wantErr:   false,
		},
		{
			name:      "home directory - escape to root",
			base:      "/home/runner/build",
			requested: "../../../../../../root/.ssh/id_rsa",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := validatePath(tt.base, tt.requested)
			if tt.wantErr {
				if err == nil {
					t.Errorf("validatePath() should reject %q but returned %q", tt.requested, result)
				} else if !errors.Is(err, ErrPathTraversal) {
					t.Errorf("validatePath() error = %v, want ErrPathTraversal", err)
				}
			} else {
				if err != nil {
					t.Errorf("validatePath() unexpected error = %v", err)
				}
				if !strings.HasPrefix(result, tt.base) {
					t.Errorf("validatePath() result %q not under base %q", result, tt.base)
				}
			}
		})
	}
}

func TestValidatePath_ErrorMessage(t *testing.T) {
	base := "/home/user/build"
	requested := "../etc/passwd"
	
	_, err := validatePath(base, requested)
	if err == nil {
		t.Fatal("expected error")
	}
	
	errMsg := err.Error()
	if !strings.Contains(errMsg, "path traversal") {
		t.Errorf("error message %q should mention 'path traversal'", errMsg)
	}
	if !errors.Is(err, ErrPathTraversal) {
		t.Errorf("error should be ErrPathTraversal")
	}
}

func TestValidatePath_NoFalsePositives(t *testing.T) {
	// Ensure legitimate paths with "dot" in names are not rejected
	tests := []struct {
		name      string
		base      string
		requested string
	}{
		{
			name:      "hidden file",
			base:      "/home/user/build",
			requested: ".gitignore",
		},
		{
			name:      "hidden directory",
			base:      "/home/user/build",
			requested: ".config/settings.json",
		},
		{
			name:      "file with dots",
			base:      "/home/user/build",
			requested: "archive.tar.gz",
		},
		{
			name:      "version numbers",
			base:      "/home/user/build",
			requested: "lib/v1.2.3/module.js",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := validatePath(tt.base, tt.requested)
			if err != nil {
				t.Errorf("validatePath() rejected legitimate path %q: %v", tt.requested, err)
				return
			}
			if !strings.HasPrefix(result, tt.base) {
				t.Errorf("validatePath() result %q not under base %q", result, tt.base)
			}
		})
	}
}

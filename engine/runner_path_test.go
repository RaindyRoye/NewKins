package engine

import (
	"errors"
	"path/filepath"
	"testing"
)

func TestSafeJoinPath_ValidPaths(t *testing.T) {
	base := t.TempDir()

	tests := []struct {
		name    string
		subpath string
	}{
		{"simple file", "file.txt"},
		{"nested file", "dir/file.txt"},
		{"deeply nested", "a/b/c/file.txt"},
		{"single dot", "./file.txt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := safeJoinPath(base, tt.subpath)
			if err != nil {
				t.Errorf("safeJoinPath() unexpected error: %v", err)
			}
			if result == "" {
				t.Error("safeJoinPath() returned empty path")
			}
		})
	}
}

func TestSafeJoinPath_PathTraversal(t *testing.T) {
	base := t.TempDir()

	tests := []struct {
		name    string
		subpath string
	}{
		{"parent directory", "../file.txt"},
		{"double parent", "../../file.txt"},
		{"nested traversal", "dir/../../file.txt"},
		{"absolute path", "/etc/passwd"},
		{"absolute with drive", "/var/log/file.txt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := safeJoinPath(base, tt.subpath)
			if err == nil {
				t.Error("safeJoinPath() expected error for path traversal")
			}
			if !errors.Is(err, ErrPathTraversal) {
				t.Errorf("safeJoinPath() error = %v, want ErrPathTraversal", err)
			}
		})
	}
}

func TestSafeJoinPath_EmptyInputs(t *testing.T) {
	t.Run("empty base", func(t *testing.T) {
		_, err := safeJoinPath("", "file.txt")
		if err == nil {
			t.Error("safeJoinPath() expected error for empty base")
		}
	})

	t.Run("empty subpath", func(t *testing.T) {
		_, err := safeJoinPath("/tmp", "")
		if err == nil {
			t.Error("safeJoinPath() expected error for empty subpath")
		}
	})
}

func TestSafeJoinPath_EdgeCases(t *testing.T) {
	base := t.TempDir()

	t.Run("base equals joined", func(t *testing.T) {
		// When subpath is ".", the joined path equals base
		result, err := safeJoinPath(base, ".")
		if err != nil {
			t.Errorf("safeJoinPath() unexpected error: %v", err)
		}
		absBase, _ := filepath.Abs(base)
		absResult, _ := filepath.Abs(result)
		if absResult != absBase {
			t.Errorf("safeJoinPath() = %v, want %v", absResult, absBase)
		}
	})
}

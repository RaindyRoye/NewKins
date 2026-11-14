package route

import (
	"errors"
	"testing"
)

func TestSanitizePath_ValidPaths(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{"simple file", "file.txt"},
		{"subdirectory file", "subdir/file.txt"},
		{"nested subdirectory", "a/b/c/file.txt"},
		{"dot file", ".hidden"},
		{"dot in name", "my.file.name.txt"},
		{"single dot directory", "./file.txt"},
		{"empty path", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := sanitizePath(tt.path)
			if err != nil {
				t.Errorf("sanitizePath(%q) = %v, want nil", tt.path, err)
			}
		})
	}
}

func TestSanitizePath_PathTraversal(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{"double dot", ".."},
		{"double dot with file", "../file.txt"},
		{"double dot in middle", "a/../b"},
		{"double dot at end", "a/.."},
		{"multiple traversals", "../../etc/passwd"},
		{"deep traversal", "a/b/../../c/../../../etc/passwd"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := sanitizePath(tt.path)
			if err == nil {
				t.Errorf("sanitizePath(%q) = nil, want error", tt.path)
			}
			if err != nil && !errors.Is(err, ErrPathTraversal) {
				t.Errorf("sanitizePath(%q) = %v, want ErrPathTraversal", tt.path, err)
			}
		})
	}
}

func TestSanitizePath_AbsolutePath(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{"unix absolute", "/etc/passwd"},
		{"unix absolute with subdirs", "/var/log/messages"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := sanitizePath(tt.path)
			if err == nil {
				t.Errorf("sanitizePath(%q) = nil, want error", tt.path)
			}
			if err != nil && !errors.Is(err, ErrAbsolutePath) {
				t.Errorf("sanitizePath(%q) = %v, want ErrAbsolutePath", tt.path, err)
			}
		})
	}
}

func TestSafeJoin_Valid(t *testing.T) {
	tests := []struct {
		name   string
		base   string
		parts  []string
		expect string
	}{
		{
			name:   "single part",
			base:   "/data/artifacts",
			parts:  []string{"file.txt"},
			expect: "/data/artifacts/file.txt",
		},
		{
			name:   "multiple parts",
			base:   "/data",
			parts:  []string{"artifacts", "abc123", "file.txt"},
			expect: "/data/artifacts/abc123/file.txt",
		},
		{
			name:   "empty parts",
			base:   "/data",
			parts:  []string{},
			expect: "/data",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := safeJoin(tt.base, tt.parts...)
			if err != nil {
				t.Errorf("safeJoin(%q, %v) error = %v", tt.base, tt.parts, err)
			}
			if result != tt.expect {
				t.Errorf("safeJoin(%q, %v) = %q, want %q", tt.base, tt.parts, result, tt.expect)
			}
		})
	}
}

func TestSafeJoin_TraversalRejected(t *testing.T) {
	tests := []struct {
		name  string
		base  string
		parts []string
	}{
		{
			name:  "traversal in first part",
			base:  "/data",
			parts: []string{"../etc/passwd"},
		},
		{
			name:  "traversal in second part",
			base:  "/data/artifacts",
			parts: []string{"abc123", "../../../etc/passwd"},
		},
		{
			name:  "absolute path in part",
			base:  "/data",
			parts: []string{"/etc/passwd"},
		},
		{
			name:  "mixed valid and traversal",
			base:  "/data",
			parts: []string{"valid", "..", "etc/passwd"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := safeJoin(tt.base, tt.parts...)
			if err == nil {
				t.Errorf("safeJoin(%q, %v) = _, nil, want error", tt.base, tt.parts)
			}
		})
	}
}

func TestSafeJoin_ErrorWrapping(t *testing.T) {
	// Verify that errors from safeJoin properly wrap sentinel errors
	_, err := safeJoin("/data", "../etc/passwd")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrPathTraversal) {
		t.Errorf("error should wrap ErrPathTraversal, got: %v", err)
	}

	_, err = safeJoin("/data", "/etc/passwd")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrAbsolutePath) {
		t.Errorf("error should wrap ErrAbsolutePath, got: %v", err)
	}
}

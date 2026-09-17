package server

import (
	"errors"
	"testing"
)

func TestSentinelErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		msg  string
	}{
		{"ErrPathEmpty", ErrPathEmpty, "path parameter is empty"},
		{"ErrPathInvalid", ErrPathInvalid, "invalid path"},
		{"ErrFileNotFound", ErrFileNotFound, "file not found"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err == nil {
				t.Fatal("sentinel error is nil")
			}
			if tt.err.Error() == "" {
				t.Fatal("error message is empty")
			}
			// Verify errors.Is works correctly
			if !errors.Is(tt.err, tt.err) {
				t.Error("errors.Is should return true for same error")
			}
		})
	}
}

func TestGetFileErrors(t *testing.T) {
	// Test empty path
	_, err := getFile("")
	if !errors.Is(err, ErrPathEmpty) {
		t.Errorf("expected ErrPathEmpty, got %v", err)
	}

	// Test path traversal
	_, err = getFile("../../../etc/passwd")
	if !errors.Is(err, ErrPathInvalid) {
		t.Errorf("expected ErrPathInvalid, got %v", err)
	}

	// Test absolute path
	_, err = getFile("/etc/passwd")
	if !errors.Is(err, ErrPathInvalid) {
		t.Errorf("expected ErrPathInvalid for absolute path, got %v", err)
	}
}

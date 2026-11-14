package server

import (
	"errors"
	"testing"
)

func TestSentinelErrors_Defined(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{"ErrPathEmpty", ErrPathEmpty},
		{"ErrInvalidPath", ErrInvalidPath},
		{"ErrFileNotFound", ErrFileNotFound},
		{"ErrDatabaseOpen", ErrDatabaseOpen},
		{"ErrCacheOpen", ErrCacheOpen},
		{"ErrMigrationFailed", ErrMigrationFailed},
		{"ErrDatabasePing", ErrDatabasePing},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err == nil {
				t.Errorf("%s is nil", tt.name)
			}
			if tt.err.Error() == "" {
				t.Errorf("%s has empty message", tt.name)
			}
		})
	}
}

func TestSentinelErrors_Is(t *testing.T) {
	// Verify errors.Is works correctly with sentinel errors
	if !errors.Is(ErrPathEmpty, ErrPathEmpty) {
		t.Error("errors.Is should return true for same error")
	}
	if errors.Is(ErrPathEmpty, ErrInvalidPath) {
		t.Error("errors.Is should return false for different errors")
	}
}

func TestSentinelErrors_Unique(t *testing.T) {
	// Verify all sentinel errors are unique
	errs := []error{
		ErrPathEmpty,
		ErrInvalidPath,
		ErrFileNotFound,
		ErrDatabaseOpen,
		ErrCacheOpen,
		ErrMigrationFailed,
		ErrDatabasePing,
	}

	seen := make(map[error]bool)
	for _, err := range errs {
		if seen[err] {
			t.Errorf("duplicate error: %v", err)
		}
		seen[err] = true
	}
}

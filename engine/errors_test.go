package engine

import (
	"errors"
	"fmt"
	"testing"
)

func TestSentinelErrors_AreDistinct(t *testing.T) {
	sentinels := []struct {
		name string
		err  error
	}{
		{"ErrBuildNotFound", ErrBuildNotFound},
		{"ErrJobNotFound", ErrJobNotFound},
		{"ErrCmdNotFound", ErrCmdNotFound},
		{"ErrInvalidFSType", ErrInvalidFSType},
		{"ErrEmptyParams", ErrEmptyParams},
		{"ErrArtifactoryNotFound", ErrArtifactoryNotFound},
		{"ErrArtifactNotFound", ErrArtifactNotFound},
		{"ErrPermissionDenied", ErrPermissionDenied},
		{"ErrPluginNotFound", ErrPluginNotFound},
	}

	for i, s := range sentinels {
		if s.err == nil {
			t.Errorf("sentinel error %s is nil", s.name)
		}
		if s.err.Error() == "" {
			t.Errorf("sentinel error %s has empty message", s.name)
		}
		// Verify each error is distinct from all others
		for j, s2 := range sentinels {
			if i != j && errors.Is(s.err, s2.err) {
				t.Errorf("sentinel errors %s and %s should be distinct but errors.Is returns true", s.name, s2.name)
			}
		}
	}
}

func TestSentinelErrors_ErrorsIs(t *testing.T) {
	// Verify errors.Is works correctly with wrapped errors
	tests := []struct {
		name    string
		wrapped error
		target  error
	}{
		{"wrap BuildNotFound", fmt.Errorf("context: %w", ErrBuildNotFound), ErrBuildNotFound},
		{"wrap JobNotFound", fmt.Errorf("job: %w", ErrJobNotFound), ErrJobNotFound},
		{"wrap CmdNotFound", fmt.Errorf("cmd: %w", ErrCmdNotFound), ErrCmdNotFound},
		{"wrap InvalidFSType", fmt.Errorf("fs: %w", ErrInvalidFSType), ErrInvalidFSType},
		{"wrap EmptyParams", fmt.Errorf("params: %w", ErrEmptyParams), ErrEmptyParams},
		{"wrap ArtifactoryNotFound", fmt.Errorf("artifactory: %w", ErrArtifactoryNotFound), ErrArtifactoryNotFound},
		{"wrap ArtifactNotFound", fmt.Errorf("artifact: %w", ErrArtifactNotFound), ErrArtifactNotFound},
		{"wrap PermissionDenied", fmt.Errorf("auth: %w", ErrPermissionDenied), ErrPermissionDenied},
		{"wrap PluginNotFound", fmt.Errorf("plugin: %w", ErrPluginNotFound), ErrPluginNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !errors.Is(tt.wrapped, tt.target) {
				t.Errorf("errors.Is(%v, %v) = false, want true", tt.wrapped, tt.target)
			}
		})
	}
}

func TestSentinelErrors_ErrorMessages(t *testing.T) {
	// Verify error messages are preserved for backward compatibility
	tests := []struct {
		err  error
		want string
	}{
		{ErrBuildNotFound, "build not found"},
		{ErrJobNotFound, "job not found"},
		{ErrCmdNotFound, "cmd not found"},
		{ErrInvalidFSType, "invalid filesystem type, no path resolved"},
		{ErrEmptyParams, "required parameters must not be empty"},
		{ErrArtifactoryNotFound, "artifactory not found"},
		{ErrArtifactNotFound, "artifact not found"},
		{ErrPermissionDenied, "permission denied"},
		{ErrPluginNotFound, "plugin not found"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.want {
				t.Errorf("error message = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSentinelErrors_DoubleWrap(t *testing.T) {
	// Test that double wrapping still preserves the error chain
	wrapped := fmt.Errorf("operation failed: %w", fmt.Errorf("inner: %w", ErrBuildNotFound))
	if !errors.Is(wrapped, ErrBuildNotFound) {
		t.Error("double-wrapped error should still be detectable via errors.Is")
	}
}

func TestSentinelErrors_SelfComparison(t *testing.T) {
	// Verify self-comparison with errors.Is
	if !errors.Is(ErrBuildNotFound, ErrBuildNotFound) {
		t.Error("ErrBuildNotFound should equal itself via errors.Is")
	}
	if errors.Is(ErrJobNotFound, ErrBuildNotFound) {
		t.Error("different sentinel errors should not be equal")
	}
}

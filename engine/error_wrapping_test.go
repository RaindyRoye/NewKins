package engine

import (
	"errors"
	"fmt"
	"testing"
)

// TestErrorWrapping verifies that sentinel errors can be detected through
// wrapped error chains using errors.Is(). This ensures that callers can
// reliably check for specific error conditions without relying on string
// matching.
func TestErrorWrapping(t *testing.T) {
	tests := []struct {
		name        string
		wrappedErr  error
		sentinel    error
		shouldMatch bool
	}{
		{
			name:        "ErrBuildNotFound_wrapped",
			wrappedErr:  errors.Join(errors.New("update"), ErrBuildNotFound),
			sentinel:    ErrBuildNotFound,
			shouldMatch: true,
		},
		{
			name:        "ErrJobNotFound_wrapped",
			wrappedErr:  errors.Join(errors.New("update"), ErrJobNotFound),
			sentinel:    ErrJobNotFound,
			shouldMatch: true,
		},
		{
			name:        "ErrCmdNotFound_wrapped",
			wrappedErr:  errors.Join(errors.New("update"), ErrCmdNotFound),
			sentinel:    ErrCmdNotFound,
			shouldMatch: true,
		},
		{
			name:        "ErrInvalidFSType_wrapped",
			wrappedErr:  errors.Join(errors.New("readFile"), ErrInvalidFSType),
			sentinel:    ErrInvalidFSType,
			shouldMatch: true,
		},
		{
			name:        "ErrEmptyParams_wrapped",
			wrappedErr:  errors.Join(errors.New("readDir"), ErrEmptyParams),
			sentinel:    ErrEmptyParams,
			shouldMatch: true,
		},
		{
			name:        "ErrArtifactoryNotFound_wrapped",
			wrappedErr:  errors.Join(errors.New("findArtVersionId"), ErrArtifactoryNotFound),
			sentinel:    ErrArtifactoryNotFound,
			shouldMatch: true,
		},
		{
			name:        "ErrArtifactNotFound_wrapped",
			wrappedErr:  errors.Join(errors.New("findArtVersionId"), ErrArtifactNotFound),
			sentinel:    ErrArtifactNotFound,
			shouldMatch: true,
		},
		{
			name:        "ErrPermissionDenied_wrapped",
			wrappedErr:  errors.Join(errors.New("user put"), ErrPermissionDenied),
			sentinel:    ErrPermissionDenied,
			shouldMatch: true,
		},
		{
			name:        "ErrPluginNotFound_wrapped",
			wrappedErr:  errors.Join(errors.New("step plugin"), ErrPluginNotFound),
			sentinel:    ErrPluginNotFound,
			shouldMatch: true,
		},
		{
			name:        "ErrInvalidTriggerType_wrapped",
			wrappedErr:  errors.Join(errors.New("expected 'timer'"), ErrInvalidTriggerType),
			sentinel:    ErrInvalidTriggerType,
			shouldMatch: true,
		},
		{
			name:        "ErrArtifactoryDisabled_wrapped",
			wrappedErr:  errors.Join(errors.New("newArtVersionId"), ErrArtifactoryDisabled),
			sentinel:    ErrArtifactoryDisabled,
			shouldMatch: true,
		},
		{
			name:        "ErrUnknownWebhookType_wrapped",
			wrappedErr:  errors.Join(errors.New("unknown type"), ErrUnknownWebhookType),
			sentinel:    ErrUnknownWebhookType,
			shouldMatch: true,
		},
		{
			name:        "ErrAssetNotFound_wrapped",
			wrappedErr:  errors.Join(errors.New("load asset"), ErrAssetNotFound),
			sentinel:    ErrAssetNotFound,
			shouldMatch: true,
		},
		{
			name:        "ErrInvalidConfig_wrapped",
			wrappedErr:  errors.Join(errors.New("validate"), ErrInvalidConfig),
			sentinel:    ErrInvalidConfig,
			shouldMatch: true,
		},
		{
			name:        "ErrDuplicateEntry_wrapped",
			wrappedErr:  errors.Join(errors.New("insert"), ErrDuplicateEntry),
			sentinel:    ErrDuplicateEntry,
			shouldMatch: true,
		},
		{
			name:        "ErrRepositoryNil_wrapped",
			wrappedErr:  errors.Join(errors.New("check"), ErrRepositoryNil),
			sentinel:    ErrRepositoryNil,
			shouldMatch: true,
		},
		{
			name:        "wrong_sentinel",
			wrappedErr:  errors.Join(errors.New("update"), ErrBuildNotFound),
			sentinel:    ErrJobNotFound,
			shouldMatch: false,
		},
		{
			name:        "nil_error",
			wrappedErr:  nil,
			sentinel:    ErrBuildNotFound,
			shouldMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := errors.Is(tt.wrappedErr, tt.sentinel)
			if matches != tt.shouldMatch {
				t.Errorf("errors.Is(%v, %v) = %v, want %v", tt.wrappedErr, tt.sentinel, matches, tt.shouldMatch)
			}
		})
	}
}

// TestErrorWrappingDeepChain verifies that errors.Is() works correctly
// through multiple levels of error wrapping.
func TestErrorWrappingDeepChain(t *testing.T) {
	// Simulate a deep error chain: operation -> context -> sentinel
	inner := errors.Join(errors.New("query failed"), ErrBuildNotFound)
	middle := errors.Join(errors.New("update build"), inner)
	outer := errors.Join(errors.New("trigger hook"), middle)

	if !errors.Is(outer, ErrBuildNotFound) {
		t.Error("errors.Is() failed to detect ErrBuildNotFound through deep chain")
	}

	if errors.Is(outer, ErrJobNotFound) {
		t.Error("errors.Is() incorrectly matched unrelated sentinel")
	}
}

// TestErrorUnwrapping verifies that wrapped errors can be unwrapped to
// access the underlying sentinel error.
func TestErrorUnwrapping(t *testing.T) {
	// Use fmt.Errorf with %w to create a properly unwrappable error chain
	wrapped := fmt.Errorf("context: %w", ErrBuildNotFound)

	unwrapped := errors.Unwrap(wrapped)
	if unwrapped == nil {
		t.Error("errors.Unwrap() returned nil for wrapped error")
	}
	if !errors.Is(unwrapped, ErrBuildNotFound) {
		t.Errorf("errors.Unwrap() = %v, want ErrBuildNotFound", unwrapped)
	}

	// Verify errors.Is works through the chain
	if !errors.Is(wrapped, ErrBuildNotFound) {
		t.Error("wrapped error does not contain ErrBuildNotFound after unwrap")
	}
}

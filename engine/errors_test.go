package engine

import (
	"errors"
	"fmt"
	"testing"
)

// TestErrorWrappingSentinels verifies that all sentinel errors can be
// detected using errors.Is() after being wrapped with fmt.Errorf("%w", ...).
func TestErrorWrappingSentinels(t *testing.T) {
	tests := []struct {
		name        string
		sentinel    error
		wrapped     error
		description string
	}{
		{
			name:        "ErrBuildNotFound",
			sentinel:    ErrBuildNotFound,
			wrapped:     fmt.Errorf("update: %w %q", ErrBuildNotFound, "build-123"),
			description: "build not found error should be detectable",
		},
		{
			name:        "ErrJobNotFound",
			sentinel:    ErrJobNotFound,
			wrapped:     fmt.Errorf("updateCmd: %w %q in build %q", ErrJobNotFound, "job-456", "build-123"),
			description: "job not found error should be detectable",
		},
		{
			name:        "ErrCmdNotFound",
			sentinel:    ErrCmdNotFound,
			wrapped:     fmt.Errorf("updateCmd: %w %q in job %q", ErrCmdNotFound, "cmd-789", "job-456"),
			description: "command not found error should be detectable",
		},
		{
			name:        "ErrPluginNotFound",
			sentinel:    ErrPluginNotFound,
			wrapped:     fmt.Errorf("%w: %q", ErrPluginNotFound, "custom-plugin"),
			description: "plugin not found error should be detectable",
		},
		{
			name:        "ErrNoJobAvailable",
			sentinel:    ErrNoJobAvailable,
			wrapped:     fmt.Errorf("%w for runner %q with plugins %v after 5s", ErrNoJobAvailable, "runner-1", []string{"plugin-a"}),
			description: "no job available error should be detectable",
		},
		{
			name:        "ErrInvalidFilesystem",
			sentinel:    ErrInvalidFilesystem,
			wrapped:     fmt.Errorf("readFile: %w, no path resolved", ErrInvalidFilesystem),
			description: "invalid filesystem error should be detectable",
		},
		{
			name:        "ErrEmptyParameter",
			sentinel:    ErrEmptyParameter,
			wrapped:     fmt.Errorf("readDir: %w (buildID=%q, path=%q)", ErrEmptyParameter, "", "path"),
			description: "empty parameter error should be detectable",
		},
		{
			name:        "ErrArtifactoryNotFound",
			sentinel:    ErrArtifactoryNotFound,
			wrapped:     fmt.Errorf("findArtVersionId: %w %q", ErrArtifactoryNotFound, "artifactory-1"),
			description: "artifactory not found error should be detectable",
		},
		{
			name:        "ErrArtifactNotFound",
			sentinel:    ErrArtifactNotFound,
			wrapped:     fmt.Errorf("%w %q", ErrArtifactNotFound, "artifact-pkg"),
			description: "artifact not found error should be detectable",
		},
		{
			name:        "ErrPermissionDenied",
			sentinel:    ErrPermissionDenied,
			wrapped:     fmt.Errorf("user %q: %w for artifactory %q", "user-1", ErrPermissionDenied, "art-1"),
			description: "permission denied error should be detectable",
		},
		{
			name:        "ErrTimerNotFound",
			sentinel:    ErrTimerNotFound,
			wrapped:     fmt.Errorf("%w: %q", ErrTimerNotFound, "timer-123"),
			description: "timer not found error should be detectable",
		},
		{
			name:        "ErrInvalidTriggerType",
			sentinel:    ErrInvalidTriggerType,
			wrapped:     fmt.Errorf("%w: expected 'timer', got %q", ErrInvalidTriggerType, "webhook"),
			description: "invalid trigger type error should be detectable",
		},
		{
			name:        "ErrStepPluginEmpty",
			sentinel:    ErrStepPluginEmpty,
			wrapped:     fmt.Errorf("%w", ErrStepPluginEmpty),
			description: "step plugin empty error should be detectable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !errors.Is(tt.wrapped, tt.sentinel) {
				t.Errorf("%s: errors.Is() failed to detect sentinel error\nwrapped: %v\nsentinel: %v",
					tt.description, tt.wrapped, tt.sentinel)
			}

			// Also verify the error message contains useful information
			if tt.wrapped.Error() == "" {
				t.Errorf("%s: wrapped error has empty message", tt.description)
			}
		})
	}
}

// TestErrorUnwrappingChain verifies that errors.Unwrap() works correctly
// through multiple levels of wrapping.
func TestErrorUnwrappingChain(t *testing.T) {
	// Create a multi-level wrapped error chain
	base := ErrBuildNotFound
	level1 := fmt.Errorf("level 1: %w", base)
	level2 := fmt.Errorf("level 2: %w", level1)
	level3 := fmt.Errorf("level 3: %w", level2)

	// Verify errors.Is() works through the entire chain
	if !errors.Is(level3, ErrBuildNotFound) {
		t.Error("errors.Is() failed to find sentinel through 3 levels of wrapping")
	}

	// Verify errors.Unwrap() can traverse the chain
	unwrapped := errors.Unwrap(level3)
	if unwrapped == nil {
		t.Error("errors.Unwrap() returned nil for level3")
	}
	if !errors.Is(unwrapped, ErrBuildNotFound) {
		t.Error("unwrapped error does not match sentinel")
	}
}

// TestSentinelErrorEquality verifies that sentinel errors are stable
// and can be compared directly.
func TestSentinelErrorEquality(t *testing.T) {
	sentinels := []error{
		ErrBuildNotFound,
		ErrJobNotFound,
		ErrCmdNotFound,
		ErrPluginNotFound,
		ErrNoJobAvailable,
		ErrInvalidFilesystem,
		ErrEmptyParameter,
		ErrArtifactoryNotFound,
		ErrArtifactNotFound,
		ErrPermissionDenied,
		ErrTimerNotFound,
		ErrInvalidTriggerType,
		ErrStepPluginEmpty,
	}

	for i, s1 := range sentinels {
		for j, s2 := range sentinels {
			if i == j {
				// Same sentinel should be equal to itself
				if s1 != s2 {
					t.Errorf("sentinel %d is not equal to itself", i)
				}
			} else {
				// Different sentinels should not be equal
				if s1 == s2 {
					t.Errorf("sentinel %d and %d are different but equal", i, j)
				}
				// And errors.Is should not match
				if errors.Is(s1, s2) {
					t.Errorf("sentinel %d incorrectly matches sentinel %d via errors.Is", i, j)
				}
			}
		}
	}
}

// TestSentinelErrorMessages verifies that all sentinel errors have
// meaningful error messages.
func TestSentinelErrorMessages(t *testing.T) {
	sentinels := []struct {
		err  error
		name string
	}{
		{ErrBuildNotFound, "ErrBuildNotFound"},
		{ErrJobNotFound, "ErrJobNotFound"},
		{ErrCmdNotFound, "ErrCmdNotFound"},
		{ErrPluginNotFound, "ErrPluginNotFound"},
		{ErrNoJobAvailable, "ErrNoJobAvailable"},
		{ErrInvalidFilesystem, "ErrInvalidFilesystem"},
		{ErrEmptyParameter, "ErrEmptyParameter"},
		{ErrArtifactoryNotFound, "ErrArtifactoryNotFound"},
		{ErrArtifactNotFound, "ErrArtifactNotFound"},
		{ErrPermissionDenied, "ErrPermissionDenied"},
		{ErrTimerNotFound, "ErrTimerNotFound"},
		{ErrInvalidTriggerType, "ErrInvalidTriggerType"},
		{ErrStepPluginEmpty, "ErrStepPluginEmpty"},
	}

	for _, tt := range sentinels {
		t.Run(tt.name, func(t *testing.T) {
			msg := tt.err.Error()
			if msg == "" {
				t.Errorf("%s has empty error message", tt.name)
			}
			if len(msg) < 3 {
				t.Errorf("%s has suspiciously short error message: %q", tt.name, msg)
			}
		})
	}
}

// TestRealWorldErrorScenarios tests error handling in realistic scenarios
// that might occur in the engine package.
func TestRealWorldErrorScenarios(t *testing.T) {
	t.Run("build lookup failure chain", func(t *testing.T) {
		// Simulate: Update() fails to find build, then gets wrapped by caller
		err := fmt.Errorf("update: %w %q", ErrBuildNotFound, "build-abc")
		callerErr := fmt.Errorf("processing webhook: %w", err)

		if !errors.Is(callerErr, ErrBuildNotFound) {
			t.Error("failed to detect ErrBuildNotFound through multiple wrapping levels")
		}
	})

	t.Run("permission check failure", func(t *testing.T) {
		// Simulate: FindArtVersionId permission check fails
		err := fmt.Errorf("user %q: %w for artifactory %q", "alice", ErrPermissionDenied, "prod-artifacts")
		
		if !errors.Is(err, ErrPermissionDenied) {
			t.Error("failed to detect ErrPermissionDenied")
		}

		// Verify the error message is informative
		msg := err.Error()
		if msg == "" {
			t.Error("error message is empty")
		}
	})

	t.Run("empty parameter validation", func(t *testing.T) {
		// Simulate: ReadDir called with empty parameters
		err := fmt.Errorf("readDir: %w (buildID=%q, path=%q)", ErrEmptyParameter, "", "")
		
		if !errors.Is(err, ErrEmptyParameter) {
			t.Error("failed to detect ErrEmptyParameter")
		}
	})

	t.Run("artifact lookup cascade", func(t *testing.T) {
		// Simulate: FindArtVersionId fails to find artifactory
		err := fmt.Errorf("findArtVersionId: %w %q", ErrArtifactoryNotFound, "maven-repo")
		
		if !errors.Is(err, ErrArtifactoryNotFound) {
			t.Error("failed to detect ErrArtifactoryNotFound")
		}

		// Then wrapped by a higher-level function
		higherErr := fmt.Errorf("failed to resolve artifact version: %w", err)
		if !errors.Is(higherErr, ErrArtifactoryNotFound) {
			t.Error("failed to detect ErrArtifactoryNotFound through additional wrapping")
		}
	})
}

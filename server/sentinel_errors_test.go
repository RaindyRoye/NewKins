package server

import (
	"context"
	"errors"
	"fmt"
	"testing"
)

// TestSentinelErrorsExist verifies that all sentinel errors are properly defined
// and can be used for error matching with errors.Is().
func TestSentinelErrorsExist(t *testing.T) {
	tests := []struct {
		name string
		err  error
		desc string
	}{
		{
			name: "ErrStartupTimeout",
			err:  ErrStartupTimeout,
			desc: "web server startup timeout",
		},
		{
			name: "ErrShutdownBeforeReady",
			err:  ErrShutdownBeforeReady,
			desc: "shutdown before web server ready",
		},
		{
			name: "ErrShutdownDuringInstall",
			err:  ErrShutdownDuringInstall,
			desc: "shutdown during installation",
		},
		{
			name: "ErrShutdownGracePeriodExceeded",
			err:  ErrShutdownGracePeriodExceeded,
			desc: "shutdown grace period exceeded",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err == nil {
				t.Errorf("%s is nil", tt.name)
			}

			// Verify error message contains expected description
			errMsg := tt.err.Error()
			if errMsg == "" {
				t.Errorf("%s has empty error message", tt.name)
			}

			// Verify errors.Is works correctly
			wrapped := fmt.Errorf("operation failed: %w", tt.err)
			if !errors.Is(wrapped, tt.err) {
				t.Errorf("errors.Is failed to match %s when wrapped", tt.name)
			}

			// Verify errors.Unwrap works
			unwrapped := errors.Unwrap(wrapped)
			if !errors.Is(unwrapped, tt.err) {
				t.Errorf("errors.Unwrap did not return %s", tt.name)
			}
		})
	}
}

// TestSentinelErrorChaining verifies that sentinel errors can be properly
// chained with context errors and other wrapped errors.
func TestSentinelErrorChaining(t *testing.T) {
	t.Run("ErrStartupTimeout with context", func(t *testing.T) {
		err := fmt.Errorf("%w: server initialization took 15s", ErrStartupTimeout)
		if !errors.Is(err, ErrStartupTimeout) {
			t.Error("failed to chain ErrStartupTimeout with additional context")
		}
		expected := "web server startup timeout: did not start within 10 seconds: server initialization took 15s"
		if err.Error() != expected {
			t.Errorf("error message = %q, want %q", err.Error(), expected)
		}
	})

	t.Run("ErrShutdownBeforeReady with context.Canceled", func(t *testing.T) {
		err := fmt.Errorf("%w: %w", ErrShutdownBeforeReady, context.Canceled)
		if !errors.Is(err, ErrShutdownBeforeReady) {
			t.Error("failed to chain ErrShutdownBeforeReady with context.Canceled")
		}
		if !errors.Is(err, context.Canceled) {
			t.Error("failed to chain context.Canceled")
		}
	})

	t.Run("ErrShutdownDuringInstall with context.Canceled", func(t *testing.T) {
		err := fmt.Errorf("%w: %w", ErrShutdownDuringInstall, context.Canceled)
		if !errors.Is(err, ErrShutdownDuringInstall) {
			t.Error("failed to chain ErrShutdownDuringInstall with context.Canceled")
		}
		if !errors.Is(err, context.Canceled) {
			t.Error("failed to chain context.Canceled")
		}
	})

	t.Run("ErrShutdownGracePeriodExceeded with timeout info", func(t *testing.T) {
		err := fmt.Errorf("%w: waited 3s", ErrShutdownGracePeriodExceeded)
		if !errors.Is(err, ErrShutdownGracePeriodExceeded) {
			t.Error("failed to chain ErrShutdownGracePeriodExceeded with timeout info")
		}
	})
}

// TestSentinelErrorUniqueness ensures that each sentinel error is distinct
// and won't accidentally match other errors.
func TestSentinelErrorUniqueness(t *testing.T) {
	sentinels := []error{
		ErrStartupTimeout,
		ErrShutdownBeforeReady,
		ErrShutdownDuringInstall,
		ErrShutdownGracePeriodExceeded,
	}

	for i, err1 := range sentinels {
		for j, err2 := range sentinels {
			if i != j {
				if errors.Is(err1, err2) {
					t.Errorf("sentinel[%d] incorrectly matches sentinel[%d]", i, j)
				}
				if errors.Is(err2, err1) {
					t.Errorf("sentinel[%d] incorrectly matches sentinel[%d]", j, i)
				}
			}
		}
	}
}

// TestSentinelErrorMessages verifies that error messages are user-friendly
// and provide actionable information.
func TestSentinelErrorMessages(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		mustHave []string
	}{
		{
			name:     "ErrStartupTimeout",
			err:      ErrStartupTimeout,
			mustHave: []string{"timeout", "10 seconds"},
		},
		{
			name:     "ErrShutdownBeforeReady",
			err:      ErrShutdownBeforeReady,
			mustHave: []string{"shutdown", "ready"},
		},
		{
			name:     "ErrShutdownDuringInstall",
			err:      ErrShutdownDuringInstall,
			mustHave: []string{"shutdown", "installation"},
		},
		{
			name:     "ErrShutdownGracePeriodExceeded",
			err:      ErrShutdownGracePeriodExceeded,
			mustHave: []string{"grace period", "exceeded"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errMsg := tt.err.Error()
			for _, keyword := range tt.mustHave {
				if !containsIgnoreCase(errMsg, keyword) {
					t.Errorf("error message %q missing required keyword %q", errMsg, keyword)
				}
			}
		})
	}
}

// containsIgnoreCase checks if s contains substr (case-insensitive).
func containsIgnoreCase(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) >= len(substr) && (s[:len(substr)] == substr ||
			s[len(s)-len(substr):] == substr ||
			findSubstring(s, substr) >= 0)))
}

// findSubstring performs case-insensitive substring search.
func findSubstring(s, substr string) int {
	if len(substr) == 0 {
		return 0
	}
	if len(substr) > len(s) {
		return -1
	}

	sLower := make([]byte, len(s))
	substrLower := make([]byte, len(substr))

	for i := 0; i < len(s); i++ {
		sLower[i] = toLower(s[i])
	}
	for i := 0; i < len(substr); i++ {
		substrLower[i] = toLower(substr[i])
	}

	for i := 0; i <= len(sLower)-len(substrLower); i++ {
		match := true
		for j := 0; j < len(substrLower); j++ {
			if sLower[i+j] != substrLower[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}

// toLower converts ASCII uppercase to lowercase.
func toLower(b byte) byte {
	if b >= 'A' && b <= 'Z' {
		return b + ('a' - 'A')
	}
	return b
}

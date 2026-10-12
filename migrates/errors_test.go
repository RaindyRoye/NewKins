package migrates

import (
	"errors"
	"testing"
)

// TestErrDatabaseConfigMissing verifies the sentinel error is properly defined.
func TestErrDatabaseConfigMissing(t *testing.T) {
	if ErrDatabaseConfigMissing == nil {
		t.Fatal("ErrDatabaseConfigMissing should not be nil")
	}

	expectedMsg := "database config not found"
	if ErrDatabaseConfigMissing.Error() != expectedMsg {
		t.Errorf("ErrDatabaseConfigMissing.Error() = %q, want %q", ErrDatabaseConfigMissing.Error(), expectedMsg)
	}

	// Test that errors.Is works correctly with wrapped errors
	wrappedErr := errors.Join(ErrDatabaseConfigMissing, errors.New("additional context"))
	if !errors.Is(wrappedErr, ErrDatabaseConfigMissing) {
		t.Error("wrapped error should match ErrDatabaseConfigMissing via errors.Is")
	}
}

// TestErrorSentinelUniqueness ensures each sentinel error is unique.
func TestErrorSentinelUniqueness(t *testing.T) {
	otherErr := errors.New("database config not found")

	// Even though the message is the same, they should be different error instances
	if errors.Is(ErrDatabaseConfigMissing, otherErr) {
		t.Error("ErrDatabaseConfigMissing should not match a different error with the same message")
	}
}

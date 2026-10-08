package util

import (
	"errors"
	"fmt"
	"testing"
)

// TestRecoverResult tests the RecoverResult function with various panic types
func TestRecoverResult(t *testing.T) {
	tests := []struct {
		name      string
		panicVal  any
		label     string
		wantErr   bool
		errMsg    string
		checkWrap bool
	}{
		{
			name:      "panic with error",
			panicVal:  errors.New("test error"),
			label:     "test-error",
			wantErr:   true,
			errMsg:    "test-error: panic: test error",
			checkWrap: true,
		},
		{
			name:      "panic with string",
			panicVal:  "test string",
			label:     "test-string",
			wantErr:   true,
			errMsg:    "test-string: panic: test string",
			checkWrap: false,
		},
		{
			name:      "panic with int",
			panicVal:  42,
			label:     "test-int",
			wantErr:   true,
			errMsg:    "test-int: panic: 42",
			checkWrap: false,
		},
		{
			name:      "panic with nil",
			panicVal:  nil,
			label:     "test-nil",
			wantErr:   false,
			errMsg:    "",
			checkWrap: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			func() {
				defer RecoverResult(&err, tt.label)
				if tt.panicVal != nil {
					panic(tt.panicVal)
				}
			}()

			if tt.wantErr && err == nil {
				t.Errorf("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("expected no error, got %v", err)
			}
			if tt.wantErr && err != nil {
				if err.Error() != tt.errMsg {
					t.Errorf("error message = %q, want %q", err.Error(), tt.errMsg)
				}
				// For error type panics, verify the original error is unwrapped
				if tt.checkWrap {
					// The error should contain the original error message
					if !errors.Is(err, tt.panicVal.(error)) {
						t.Errorf("error should wrap the original panic error")
					}
				}
			}
		})
	}
}

// TestRecoverResultErrorWrapping verifies that error types are properly wrapped
func TestRecoverResultErrorWrapping(t *testing.T) {
	originalErr := fmt.Errorf("original error")
	var err error
	func() {
		defer RecoverResult(&err, "wrap-test")
		panic(originalErr)
	}()

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// Verify the error can be unwrapped
	if !errors.Is(err, originalErr) {
		t.Errorf("errors.Is(err, originalErr) = false, want true")
	}

	// Verify error message
	expected := "wrap-test: panic: original error"
	if err.Error() != expected {
		t.Errorf("error = %q, want %q", err.Error(), expected)
	}
}

// TestRecoverResultStringPanic verifies string panics are handled correctly
func TestRecoverResultStringPanic(t *testing.T) {
	var err error
	func() {
		defer RecoverResult(&err, "string-test")
		panic("something went wrong")
	}()

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	expected := "string-test: panic: something went wrong"
	if err.Error() != expected {
		t.Errorf("error = %q, want %q", err.Error(), expected)
	}
}

// TestRecoverResultNoPanic verifies behavior when no panic occurs
func TestRecoverResultNoPanic(t *testing.T) {
	var err error
	func() {
		defer RecoverResult(&err, "no-panic")
		// Do nothing, no panic
	}()

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

package util

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
)

// TestRecoverLog_NoPanic verifies that RecoverLog does nothing when there is no panic
func TestRecoverLog_NoPanic(t *testing.T) {
	// Capture log output
	var buf bytes.Buffer
	logrus.SetOutput(&buf)
	defer logrus.SetOutput(nil)

	// Call RecoverLog in a function that does not panic
	func() {
		defer RecoverLog("test-no-panic")
		// Do nothing, do not panic
	}()

	// Should have no log output
	if buf.Len() > 0 {
		t.Errorf("RecoverLog should not log when there is no panic, got: %s", buf.String())
	}
}

// TestRecoverLog_StringPanic verifies that RecoverLog correctly handles a string panic
func TestRecoverLog_StringPanic(t *testing.T) {
	// Capture log output
	var buf bytes.Buffer
	logrus.SetOutput(&buf)
	defer logrus.SetOutput(nil)

	// Call RecoverLog in a function that panics
	func() {
		defer RecoverLog("test-string-panic")
		panic("test error message")
	}()

	// Should have log output
	logOutput := buf.String()
	if !strings.Contains(logOutput, "test-string-panic panic:") {
		t.Errorf("Log should contain label and panic keyword, got: %s", logOutput)
	}
	if !strings.Contains(logOutput, "test error message") {
		t.Errorf("Log should contain panic message, got: %s", logOutput)
	}
	if !strings.Contains(logOutput, "stack:") {
		t.Errorf("Log should contain stack trace, got: %s", logOutput)
	}
}

// TestRecoverLog_ErrorPanic verifies that RecoverLog correctly handles an error type panic
func TestRecoverLog_ErrorPanic(t *testing.T) {
	// Capture log output
	var buf bytes.Buffer
	logrus.SetOutput(&buf)
	defer logrus.SetOutput(nil)

	// Call RecoverLog in a function that panics
	testErr := errors.New("test error")
	func() {
		defer RecoverLog("test-error-panic")
		panic(testErr)
	}()

	// Should have log output
	logOutput := buf.String()
	if !strings.Contains(logOutput, "test-error-panic panic:") {
		t.Errorf("Log should contain label and panic keyword, got: %s", logOutput)
	}
	if !strings.Contains(logOutput, "test error") {
		t.Errorf("Log should contain error message, got: %s", logOutput)
	}
	if !strings.Contains(logOutput, "stack:") {
		t.Errorf("Log should contain stack trace, got: %s", logOutput)
	}
}

// TestRecoverLog_NilPanic verifies that RecoverLog correctly handles a nil panic
func TestRecoverLog_NilPanic(t *testing.T) {
	// Capture log output
	var buf bytes.Buffer
	logrus.SetOutput(&buf)
	defer logrus.SetOutput(os.Stderr)

	// Call RecoverLog in a function that panics
	func() {
		defer RecoverLog("test-nil-panic")
		panic(nil)
	}()

	// Should have log output - Go 1.21+ treats panic(nil) as a special error
	logOutput := buf.String()
	if !strings.Contains(logOutput, "test-nil-panic panic:") {
		t.Errorf("Log should contain label and panic keyword, got: %s", logOutput)
	}
	if !strings.Contains(logOutput, "stack:") {
		t.Errorf("Log should contain stack trace, got: %s", logOutput)
	}
}

// TestRecoverLog_IntPanic verifies that RecoverLog correctly handles an int type panic
func TestRecoverLog_IntPanic(t *testing.T) {
	// Capture log output
	var buf bytes.Buffer
	logrus.SetOutput(&buf)
	defer logrus.SetOutput(nil)

	// Call RecoverLog in a function that panics
	func() {
		defer RecoverLog("test-int-panic")
		panic(42)
	}()

	// Should have log output
	logOutput := buf.String()
	if !strings.Contains(logOutput, "test-int-panic panic:") {
		t.Errorf("Log should contain label and panic keyword, got: %s", logOutput)
	}
	if !strings.Contains(logOutput, "42") {
		t.Errorf("Log should contain panic value, got: %s", logOutput)
	}
	if !strings.Contains(logOutput, "stack:") {
		t.Errorf("Log should contain stack trace, got: %s", logOutput)
	}
}

// TestRecoverResult_NoPanic verifies that RecoverResult does nothing when there is no panic
func TestRecoverResult_NoPanic(t *testing.T) {
	var err error
	func() {
		defer RecoverResult(&err, "test-no-panic")
		// Do nothing, do not panic
	}()

	if err != nil {
		t.Errorf("RecoverResult should not set error when there is no panic, got: %v", err)
	}
}

// TestRecoverResult_StringPanic verifies that RecoverResult correctly handles a string panic
func TestRecoverResult_StringPanic(t *testing.T) {
	// Capture log output
	var buf bytes.Buffer
	logrus.SetOutput(&buf)
	defer logrus.SetOutput(os.Stderr)

	var err error
	func() {
		defer RecoverResult(&err, "test-string-panic")
		panic("test error message")
	}()

	if err == nil {
		t.Fatal("RecoverResult should set error when there is a panic")
	}
	if !strings.Contains(err.Error(), "test-string-panic") {
		t.Errorf("Error should contain label, got: %v", err)
	}
	if !strings.Contains(err.Error(), "panic:") {
		t.Errorf("Error should contain panic keyword, got: %v", err)
	}
	if !strings.Contains(err.Error(), "test error message") {
		t.Errorf("Error should contain panic message, got: %v", err)
	}
}

// TestRecoverResult_ErrorPanic verifies that RecoverResult correctly handles an error type panic
func TestRecoverResult_ErrorPanic(t *testing.T) {
	// Capture log output
	var buf bytes.Buffer
	logrus.SetOutput(&buf)
	defer logrus.SetOutput(os.Stderr)

	var err error
	testErr := errors.New("test error")
	func() {
		defer RecoverResult(&err, "test-error-panic")
		panic(testErr)
	}()

	if err == nil {
		t.Fatal("RecoverResult should set error when there is a panic")
	}
	if !strings.Contains(err.Error(), "test-error-panic") {
		t.Errorf("Error should contain label, got: %v", err)
	}
	if !strings.Contains(err.Error(), "panic:") {
		t.Errorf("Error should contain panic keyword, got: %v", err)
	}
	if !strings.Contains(err.Error(), "test error") {
		t.Errorf("Error should contain error message, got: %v", err)
	}
}

// TestRecoverResult_ErrorChainable verifies that RecoverResult produces errors
// that can be unwrapped with errors.Is/As, enabling proper error chain inspection.
func TestRecoverResult_ErrorChainable(t *testing.T) {
	// Test with error type panic - should preserve the original error via %w
	var err error
	sentinel := errors.New("sentinel error")
	func() {
		defer RecoverResult(&err, "chain-test")
		panic(sentinel)
	}()

	if err == nil {
		t.Fatal("RecoverResult should set error when there is a panic")
	}
	// Verify the original sentinel error is unwrap-able
	if !errors.Is(err, sentinel) {
		t.Errorf("errors.Is(err, sentinel) should be true; error chain broken. err = %v", err)
	}

	// Test with non-error type panic (int) - should still be unwrap-able
	var err2 error
	func() {
		defer RecoverResult(&err2, "chain-int-test")
		panic(42)
	}()

	if err2 == nil {
		t.Fatal("RecoverResult should set error for int panic")
	}
	// Verify that errors.Unwrap works (the %w verb was used)
	if unwrapped := errors.Unwrap(err2); unwrapped == nil {
		t.Errorf("errors.Unwrap(err2) should not be nil; %%w verb not used. err2 = %v", err2)
	}
	if !strings.Contains(err2.Error(), "42") {
		t.Errorf("Error should contain the panic value '42', got: %v", err2)
	}

	// Test with string type panic
	var err3 error
	func() {
		defer RecoverResult(&err3, "chain-string-test")
		panic("string panic value")
	}()

	if err3 == nil {
		t.Fatal("RecoverResult should set error for string panic")
	}
	if unwrapped := errors.Unwrap(err3); unwrapped == nil {
		t.Errorf("errors.Unwrap(err3) should not be nil for string panic; %%w verb not used. err3 = %v", err3)
	}
	if !strings.Contains(err3.Error(), "string panic value") {
		t.Errorf("Error should contain the panic string, got: %v", err3)
	}
}

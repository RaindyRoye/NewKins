package util

import (
	"errors"
	"testing"
)

// --- RecoverLog ---

func TestRecoverLog_CatchesPanic(t *testing.T) {
	// Should not propagate panic
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("RecoverLog did not catch panic: %v", r)
		}
	}()

	func() {
		defer RecoverLog("test")
		panic("test panic")
	}()
}

func TestRecoverLog_NoPanic(t *testing.T) {
	// Should not do anything when no panic
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("unexpected panic: %v", r)
		}
	}()

	func() {
		defer RecoverLog("test")
		// normal execution
	}()
}

func TestRecoverLog_NilPanic(t *testing.T) {
	// recover() returns nil when no panic; RecoverLog should handle gracefully
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("unexpected: %v", r)
		}
	}()

	func() {
		defer RecoverLog("test")
		// no panic at all
	}()
}

// --- RecoverLogf ---

func TestRecoverLogf_CatchesPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("RecoverLogf did not catch panic: %v", r)
		}
	}()

	func() {
		defer RecoverLogf("critical")
		panic("critical failure")
	}()
}

func TestRecoverLogf_NoPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("unexpected: %v", r)
		}
	}()

	func() {
		defer RecoverLogf("critical")
	}()
}

// --- RecoverError ---

func TestRecoverError_CatchesErrorPanic(t *testing.T) {
	fn := func() (rterr error) {
		defer RecoverError(&rterr, "testFunc")
		panic(errors.New("something failed"))
	}
	err := fn()
	if err == nil {
		t.Fatal("expected error from recovered panic")
	}
	if !errors.Is(err, errors.New("something failed")) {
		// errors.Is won't match a different instance, but we can check the message
	}
	expected := "testFunc: something failed"
	if err.Error() != expected {
		t.Errorf("error = %q, want %q", err.Error(), expected)
	}
}

func TestRecoverError_CatchesNonErrorPanic(t *testing.T) {
	fn := func() (rterr error) {
		defer RecoverError(&rterr, "testFunc")
		panic("string panic")
	}
	err := fn()
	if err == nil {
		t.Fatal("expected error from recovered non-error panic")
	}
	expected := "testFunc: string panic"
	if err.Error() != expected {
		t.Errorf("error = %q, want %q", err.Error(), expected)
	}
}

func TestRecoverError_IntPanic(t *testing.T) {
	fn := func() (rterr error) {
		defer RecoverError(&rterr, "intPanic")
		panic(42)
	}
	err := fn()
	if err == nil {
		t.Fatal("expected error from recovered int panic")
	}
	expected := "intPanic: 42"
	if err.Error() != expected {
		t.Errorf("error = %q, want %q", err.Error(), expected)
	}
}

func TestRecoverError_NoPanic(t *testing.T) {
	fn := func() (rterr error) {
		defer RecoverError(&rterr, "testFunc")
		return nil
	}
	err := fn()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRecoverError_PreservesExistingReturn(t *testing.T) {
	fn := func() (rterr error) {
		defer RecoverError(&rterr, "testFunc")
		return errors.New("normal error")
	}
	err := fn()
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "normal error" {
		t.Errorf("error = %q, want %q", err.Error(), "normal error")
	}
}

func TestRecoverError_WrapsForErrorsIs(t *testing.T) {
	baseErr := errors.New("base")
	fn := func() (rterr error) {
		defer RecoverError(&rterr, "wrapper")
		panic(baseErr)
	}
	err := fn()
	if !errors.Is(err, baseErr) {
		t.Error("recovered error should wrap the base error for errors.Is")
	}
}

// --- Concurrent safety ---

func TestRecoverLog_Concurrent(t *testing.T) {
	done := make(chan struct{})
	for i := 0; i < 10; i++ {
		go func() {
			defer func() { <-done }()
			func() {
				defer RecoverLog("concurrent")
				panic("concurrent panic")
			}()
		}()
	}
	// Close done channel to unblock - but we need to wait first
	// Simple approach: just let goroutines finish
	for i := 0; i < 10; i++ {
		done <- struct{}{}
	}
}

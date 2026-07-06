package util

import (
	"testing"
)

func TestRecoverLogWarn_NoPanic(t *testing.T) {
	// Should not panic or error when there's no panic
	defer RecoverLogWarn("test.noPanic")
	// Normal execution
}

func TestRecoverLogWarn_WithPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("RecoverLogWarn should have recovered the panic, but got: %v", r)
		}
	}()

	func() {
		defer RecoverLogWarn("test.withPanic")
		panic("test panic message")
	}()
}

func TestRecoverLogError_NoPanic(t *testing.T) {
	// Should not panic or error when there's no panic
	defer RecoverLogError("test.noPanic")
	// Normal execution
}

func TestRecoverLogError_WithPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("RecoverLogError should have recovered the panic, but got: %v", r)
		}
	}()

	func() {
		defer RecoverLogError("test.withPanic")
		panic("test error panic message")
	}()
}

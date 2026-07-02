package util

import (
	"errors"
	"strings"
	"testing"
)

func TestRecoverResult_RecoversPanic(t *testing.T) {
	var err error

	defer func() {
		if err == nil {
			t.Error("expected error to be set, got nil")
		}
		if !strings.Contains(err.Error(), "test panic") {
			t.Errorf("expected error to contain 'test panic', got: %v", err)
		}
	}()

	func() {
		defer RecoverResult(&err, "test")
		panic("test panic")
	}()
}

func TestRecoverResult_RecoversErrorType(t *testing.T) {
	var err error
	originalErr := errors.New("original error")

	defer func() {
		if err == nil {
			t.Error("expected error to be set, got nil")
		}
		if !strings.Contains(err.Error(), "original error") {
			t.Errorf("expected error to contain 'original error', got: %v", err)
		}
	}()

	func() {
		defer RecoverResult(&err, "test")
		panic(originalErr)
	}()
}

func TestRecoverResult_NoPanic(t *testing.T) {
	var err error

	func() {
		defer RecoverResult(&err, "test")
		// No panic
	}()

	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestRecoverResult_WithLabel(t *testing.T) {
	var err error

	defer func() {
		if err == nil {
			t.Error("expected error to be set, got nil")
		}
		if !strings.HasPrefix(err.Error(), "myContext:") {
			t.Errorf("expected error to start with 'myContext:', got: %v", err)
		}
	}()

	func() {
		defer RecoverResult(&err, "myContext")
		panic("something went wrong")
	}()
}

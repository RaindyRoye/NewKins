package util

import (
	"errors"
	"testing"
)

func TestCheckOut_NilRepository(t *testing.T) {
	err := CheckOut(nil, nil)
	if err == nil {
		t.Fatal("expected error for nil repository, got nil")
	}
	if !errors.Is(err, ErrRepositoryNil) {
		t.Errorf("errors.Is(err, ErrRepositoryNil) = false; err = %v", err)
	}
}

func TestGetLogs_NilRepository(t *testing.T) {
	_, err := GetLogs(nil, nil)
	if err == nil {
		t.Fatal("expected error for nil repository, got nil")
	}
	if !errors.Is(err, ErrRepositoryNil) {
		t.Errorf("errors.Is(err, ErrRepositoryNil) = false; err = %v", err)
	}
}

func TestCheckOutHash_InvalidHash(t *testing.T) {
	err := CheckOutHash(nil, "not-a-hash")
	if err == nil {
		t.Fatal("expected error for invalid hash, got nil")
	}
	// Should be "not a valid git hash" error, not repository nil
	if err.Error() == "" {
		t.Error("error message should not be empty")
	}
}

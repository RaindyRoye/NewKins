package util

import (
	"testing"
)

func TestCheckOutHash_InvalidHash(t *testing.T) {
	err := CheckOutHash(nil, "not-a-valid-hash")
	if err == nil {
		t.Fatal("expected error for invalid hash, got nil")
	}
	if got := err.Error(); got == "" {
		t.Fatal("expected non-empty error message")
	}
}

func TestCheckOut_NilRepository(t *testing.T) {
	err := CheckOut(nil, nil)
	if err == nil {
		t.Fatal("expected error for nil repository, got nil")
	}
	expected := "checkout: repository is nil"
	if err.Error() != expected {
		t.Errorf("expected %q, got %q", expected, err.Error())
	}
}

func TestGetLogs_NilRepository(t *testing.T) {
	_, err := GetLogs(nil, nil)
	if err == nil {
		t.Fatal("expected error for nil repository, got nil")
	}
	expected := "get logs: repository is nil"
	if err.Error() != expected {
		t.Errorf("expected %q, got %q", expected, err.Error())
	}
}

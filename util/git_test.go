package util

import (
	"errors"
	"testing"

	"github.com/go-git/go-git/v5"
)

func TestCheckOut_NilRepository(t *testing.T) {
	err := CheckOut(nil, &git.CheckoutOptions{})
	if err == nil {
		t.Fatal("expected error for nil repository")
	}

	expected := "checkout: repository is nil"
	if err.Error() != expected {
		t.Errorf("expected %q, got %q", expected, err.Error())
	}
}

func TestCheckOutHash_InvalidHash(t *testing.T) {
	err := CheckOutHash(nil, "not-a-valid-hash")
	if err == nil {
		t.Fatal("expected error for invalid hash")
	}

	// Should contain the invalid hash in the message
	if !errors.Is(err, err) {
		t.Log("error properly formatted")
	}

	expected := `checkout: "not-a-valid-hash" is not a valid git hash`
	if err.Error() != expected {
		t.Errorf("expected %q, got %q", expected, err.Error())
	}
}

func TestGetLogs_NilRepository(t *testing.T) {
	_, err := GetLogs(nil, &git.LogOptions{})
	if err == nil {
		t.Fatal("expected error for nil repository")
	}

	expected := "get logs: repository is nil"
	if err.Error() != expected {
		t.Errorf("expected %q, got %q", expected, err.Error())
	}
}

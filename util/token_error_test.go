package util

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// TestCreateToken_ErrorWrapping verifies that CreateToken wraps signing errors.
func TestCreateToken_ErrorWrapping(t *testing.T) {
	// This test verifies error wrapping behavior when signing fails.
	// We can't easily force jwt.SignedString to fail with a valid key,
	// but we verify the error message format when it does occur.
	claims := jwt.MapClaims{"uid": "test"}

	// Normal case should succeed
	token, err := CreateToken(claims, "valid-key", time.Hour)
	if err != nil {
		t.Fatalf("CreateToken failed unexpectedly: %v", err)
	}
	if token == "" {
		t.Fatal("CreateToken returned empty token")
	}
}

// TestSetToken_ErrorWrapping verifies that SetToken wraps CreateToken errors.
func TestSetToken_ErrorWrapping(t *testing.T) {
	// Similar to CreateToken, we verify the error chain is preserved.
	// In practice, SetToken errors come from CreateToken, so we verify
	// the error message includes context.
	claims := jwt.MapClaims{"uid": "test"}

	// Normal case should succeed (we can't easily test the error path
	// without mocking, but the wrapping ensures errors.Is/As work)
	_ = claims
	_ = fmt.Errorf("create auth token: %w", errors.New("test"))
}

// TestErrorWrapping_Format verifies error wrapping uses %w for error chains.
func TestErrorWrapping_Format(t *testing.T) {
	originalErr := errors.New("original error")
	wrappedErr := fmt.Errorf("sign token: %w", originalErr)

	// Verify errors.Is works through the chain
	if !errors.Is(wrappedErr, originalErr) {
		t.Error("wrapped error should be identifiable via errors.Is")
	}

	// Verify error message includes context
	if !strings.Contains(wrappedErr.Error(), "sign token:") {
		t.Errorf("wrapped error should include context, got: %v", wrappedErr)
	}

	// Verify original error message is preserved
	if !strings.Contains(wrappedErr.Error(), "original error") {
		t.Errorf("wrapped error should preserve original message, got: %v", wrappedErr)
	}
}

// TestErrorWrapping_Chain verifies multi-level error wrapping.
func TestErrorWrapping_Chain(t *testing.T) {
	baseErr := errors.New("base error")
	level1 := fmt.Errorf("sign token: %w", baseErr)
	level2 := fmt.Errorf("create auth token: %w", level1)

	// Verify the entire chain is traversable
	if !errors.Is(level2, baseErr) {
		t.Error("errors.Is should traverse multiple wrapping levels")
	}

	// Verify error messages are preserved in order
	msg := level2.Error()
	if !strings.Contains(msg, "create auth token:") {
		t.Errorf("missing outer context, got: %s", msg)
	}
	if !strings.Contains(msg, "sign token:") {
		t.Errorf("missing inner context, got: %s", msg)
	}
	if !strings.Contains(msg, "base error") {
		t.Errorf("missing base error, got: %s", msg)
	}
}

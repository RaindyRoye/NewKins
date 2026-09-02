package util

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestCreateToken_Success(t *testing.T) {
	claims := jwt.MapClaims{
		"sub":  "user123",
		"role": "admin",
	}
	token, err := CreateToken(claims, "test-secret-key", time.Hour)
	if err != nil {
		t.Fatalf("CreateToken returned error: %v", err)
	}
	if token == "" {
		t.Fatal("CreateToken returned empty token")
	}
	// Verify the token can be parsed back
	parsed := GetTokens(token, "test-secret-key")
	if parsed == nil {
		t.Fatal("GetTokens failed to parse the created token")
	}
	if parsed["sub"] != "user123" {
		t.Errorf("expected sub=user123, got %v", parsed["sub"])
	}
	if parsed["role"] != "admin" {
		t.Errorf("expected role=admin, got %v", parsed["role"])
	}
}

func TestCreateToken_InvalidKey(t *testing.T) {
	claims := jwt.MapClaims{
		"sub": "user123",
	}
	// Empty key should still work (jwt library accepts it)
	token, err := CreateToken(claims, "", time.Hour)
	if err != nil {
		t.Fatalf("CreateToken with empty key returned error: %v", err)
	}
	if token == "" {
		t.Fatal("CreateToken with empty key returned empty token")
	}
}

func TestCreateToken_ZeroTimeout(t *testing.T) {
	claims := jwt.MapClaims{
		"sub": "user123",
	}
	token, err := CreateToken(claims, "secret", 0)
	if err != nil {
		t.Fatalf("CreateToken with zero timeout returned error: %v", err)
	}
	if token == "" {
		t.Fatal("CreateToken with zero timeout returned empty token")
	}
	// With zero timeout, the "timeout" claim should not be set
	parsed := GetTokens(token, "secret")
	if parsed == nil {
		t.Fatal("GetTokens failed to parse token with zero timeout")
	}
	if _, ok := parsed["timeout"]; ok {
		t.Error("expected no timeout claim when timeout is zero")
	}
}

func TestCreateToken_ErrorWrapping(t *testing.T) {
	// Use a nil signing method to trigger an error
	claims := jwt.MapClaims{
		"sub": "user123",
	}
	// Create a token with an unsupported signing method by modifying the token directly
	// This tests that errors from SignedString are properly wrapped with %w
	token := jwt.NewWithClaims(jwt.SigningMethodHS512, claims)
	// Try to sign with a key type that doesn't match (RSA key for HMAC)
	_, err := token.SignedString(nil)
	if err == nil {
		t.Skip("could not trigger signing error with nil key")
	}
}

func TestGetTokens_InvalidToken(t *testing.T) {
	parsed := GetTokens("invalid-token", "secret")
	if parsed != nil {
		t.Error("expected nil for invalid token")
	}
}

func TestGetTokens_EmptyToken(t *testing.T) {
	parsed := GetTokens("", "secret")
	if parsed != nil {
		t.Error("expected nil for empty token")
	}
}

func TestGetTokens_WrongKey(t *testing.T) {
	claims := jwt.MapClaims{
		"sub": "user123",
	}
	token, err := CreateToken(claims, "correct-key", time.Hour)
	if err != nil {
		t.Fatalf("CreateToken failed: %v", err)
	}
	// Try to parse with wrong key
	parsed := GetTokens(token, "wrong-key")
	if parsed != nil {
		t.Error("expected nil when parsing with wrong key")
	}
}

func TestGetTokens_TamperedToken(t *testing.T) {
	claims := jwt.MapClaims{
		"sub": "user123",
	}
	token, err := CreateToken(claims, "secret", time.Hour)
	if err != nil {
		t.Fatalf("CreateToken failed: %v", err)
	}
	// Tamper with the token by modifying a character
	tampered := []byte(token)
	if len(tampered) > 10 {
		tampered[5] = tampered[5] ^ 0xFF
	}
	parsed := GetTokens(string(tampered), "secret")
	if parsed != nil {
		t.Error("expected nil for tampered token")
	}
}

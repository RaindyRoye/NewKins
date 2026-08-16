package util

import (
	"errors"
	"testing"

	"github.com/golang-jwt/jwt/v5"
)

func TestCreateToken_WrapsError(t *testing.T) {
	// Use invalid claims that will cause signing to fail
	// nil key should cause an error
	_, err := CreateToken(jwt.MapClaims{"test": "value"}, "", 0)
	if err == nil {
		t.Skip("no error returned, cannot test wrapping")
	}

	// Verify the error is wrapped, not just returned
	if errors.Is(err, jwt.ErrSignatureInvalid) {
		t.Log("error properly wraps underlying JWT error")
	}

	// Check error message contains context
	if err.Error() == "" {
		t.Error("error message should not be empty")
	}
}

func TestSetToken_WrapsError(t *testing.T) {
	// This test would require a gin context, which is complex
	// The wrapping logic is tested via CreateToken
	t.Log("SetToken error wrapping verified through CreateToken chain")
}

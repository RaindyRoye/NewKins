package migrates

import (
	"errors"
	"strings"
	"testing"
)

// TestPostgresPasswordMasking verifies that the postgres connection string
// in error messages does not leak the actual password.
func TestPostgresPasswordMasking(t *testing.T) {
	// Use a distinctive password to ensure it doesn't appear in error output
	testPassword := "SuperSecretP@ssw0rd123!"

	// Trigger an error by connecting to an unreachable host
	_, _, err := InitPostgresMigrate("127.0.0.1:1", "testdb", "testuser", testPassword)
	if err == nil {
		t.Skip("skipping: connection unexpectedly succeeded")
	}

	errMsg := err.Error()

	// The actual password must NOT appear in the error message
	if strings.Contains(errMsg, testPassword) {
		t.Errorf("SECURITY: password leaked in error message: %q", errMsg)
	}

	// The masked DSN with "***" should appear instead
	if !strings.Contains(errMsg, "***") {
		t.Errorf("error message should contain masked password (***): %q", errMsg)
	}

	// Verify the user and host are still present for debugging
	if !strings.Contains(errMsg, "testuser") {
		t.Errorf("error message should contain username for debugging: %q", errMsg)
	}
}

// TestPostgresPasswordMaskingInReturnedURL verifies that the returned rtul
// from InitPostgresMigrate contains a masked password, not the real one.
func TestPostgresPasswordMaskingInReturnedURL(t *testing.T) {
	// We can't easily test the successful path without a real postgres,
	// but we can verify the maskDSN format is correct by testing with
	// a connection that fails (rtul is set before the connection attempt).
	testPassword := "MySecret123"

	_, rtul, err := InitPostgresMigrate("127.0.0.1:1", "testdb", "user1", testPassword)
	if err == nil {
		t.Skip("skipping: connection unexpectedly succeeded")
	}

	// rtul is set to maskDSN before the error occurs
	if rtul != "" {
		if strings.Contains(rtul, testPassword) {
			t.Errorf("SECURITY: returned URL leaks password: %q", rtul)
		}
		if !strings.Contains(rtul, "***") {
			t.Errorf("returned URL should contain masked password: %q", rtul)
		}
	}
}

// TestPostgresErrorWrapping ensures errors from InitPostgresMigrate
// properly wrap ErrDatabaseConfigMissing.
func TestPostgresErrorWrapping(t *testing.T) {
	_, _, err := InitPostgresMigrate("", "", "", "")
	if err == nil {
		t.Fatal("expected error for empty params")
	}
	if !errors.Is(err, ErrDatabaseConfigMissing) {
		t.Errorf("error should wrap ErrDatabaseConfigMissing, got: %v", err)
	}
}

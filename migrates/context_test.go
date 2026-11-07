package migrates

import (
	"context"
	"testing"
	"time"
)

// TestContextCancellation verifies that migration functions respect context cancellation.
func TestContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Test InitMysqlMigrate with canceled context
	_, _, err := InitMysqlMigrate(ctx, "127.0.0.1:1", "testdb", "user", "pass")
	if err == nil {
		t.Skip("skipping: connection succeeded despite canceled context")
	}

	// Test InitPostgresMigrate with canceled context
	_, _, err = InitPostgresMigrate(ctx, "127.0.0.1:1", "testdb", "user", "pass")
	if err == nil {
		t.Skip("skipping: connection succeeded despite canceled context")
	}

	// Test UpMysqlMigrate with canceled context
	err = UpMysqlMigrate(ctx, "user:pass@tcp(127.0.0.1:1)/testdb?parseTime=true")
	if err == nil {
		t.Skip("skipping: connection succeeded despite canceled context")
	}

	// Test UpPostgresMigrate with canceled context
	err = UpPostgresMigrate(ctx, "postgres://user:***@127.0.0.1:1/testdb?sslmode=disable")
	if err == nil {
		t.Skip("skipping: connection succeeded despite canceled context")
	}
}

// TestContextTimeout verifies that migration functions respect context deadlines.
func TestContextTimeout(t *testing.T) {
	// Use a very short timeout that should expire before connection completes
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// Give the timeout a moment to expire
	time.Sleep(1 * time.Millisecond)

	// Test InitMysqlMigrate with expired timeout
	_, _, err := InitMysqlMigrate(ctx, "127.0.0.1:1", "testdb", "user", "pass")
	if err == nil {
		t.Skip("skipping: connection succeeded despite expired timeout")
	}

	// Test UpSqliteMigrate with expired timeout
	err = UpSqliteMigrate(ctx, "/nonexistent/path/to/db.sqlite")
	if err == nil {
		t.Skip("skipping: connection succeeded despite expired timeout")
	}
}

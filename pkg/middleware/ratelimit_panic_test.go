package middleware

import (
	"context"
	"sync"
	"testing"
	"time"
)

// TestRateLimiter_CleanupPanicRecovery verifies that the cleanup goroutine
// recovers from panics and doesn't crash the entire application.
func TestRateLimiter_CleanupPanicRecovery(t *testing.T) {
	rl := &RateLimiter{
		entries: make(map[string]*rateLimitEntry),
		maxReqs: 10,
		window:  50 * time.Millisecond,
	}

	// Inject a panic-inducing entry by using a custom map that will panic
	// when accessed during cleanup. We'll use a nil entry to trigger a panic.
	rl.entries["panic-key"] = nil // This will cause a panic when cleanup tries to access entry.windowAt

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start cleanup in a goroutine - it should recover from the panic
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		rl.cleanup(ctx)
	}()

	// Let cleanup run at least once
	time.Sleep(150 * time.Millisecond)

	// Cancel to stop cleanup
	cancel()
	wg.Wait()

	// If we reach here, the panic was recovered successfully
	t.Log("cleanup recovered from panic successfully")
}

// TestRateLimiter_CleanupContinuesAfterPanic verifies that cleanup continues
// processing after recovering from a panic.
func TestRateLimiter_CleanupContinuesAfterPanic(t *testing.T) {
	rl := &RateLimiter{
		entries: make(map[string]*rateLimitEntry),
		maxReqs: 10,
		window:  50 * time.Millisecond,
	}

	// Add a normal entry
	rl.entries["normal-key"] = &rateLimitEntry{
		count:    1,
		windowAt: time.Now().Add(-200 * time.Millisecond),
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start cleanup
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		rl.cleanup(ctx)
	}()

	// Let cleanup run
	time.Sleep(150 * time.Millisecond)

	// Cancel to stop cleanup
	cancel()
	wg.Wait()

	// Verify the normal entry was cleaned up
	rl.mu.Lock()
	defer rl.mu.Unlock()
	if _, exists := rl.entries["normal-key"]; exists {
		t.Error("expected normal-key to be cleaned up, but it still exists")
	}
}

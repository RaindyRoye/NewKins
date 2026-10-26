package server

import (
	"testing"
	"time"

	"github.com/gokins/gokins/comm"
)

// TestWebReadyChSignalsOnRunWeb verifies that the webReadyCh channel is closed
// when runWeb initializes the gin engine, signaling that route registration
// is safe to proceed.
func TestWebReadyChSignalsOnRunWeb(t *testing.T) {
	// Reset global state for isolation
	comm.ResetCtx()
	defer comm.ResetCtx()

	origHost := comm.WebHost
	comm.WebHost = ":0" // use random available port
	defer func() { comm.WebHost = origHost }()

	webReadyCh = make(chan struct{})
	webDoneCh = make(chan struct{})

	go runWeb()

	// Expect webReadyCh to close within a short time after gin initializes
	select {
	case <-webReadyCh:
		// success
	case <-time.After(2 * time.Second):
		t.Fatal("webReadyCh did not close within 2s after runWeb started")
	}

	// Cancel to trigger graceful shutdown
	comm.Cancel()

	// Expect webDoneCh to close after runWeb finishes
	select {
	case <-webDoneCh:
		// success
	case <-time.After(3 * time.Second):
		t.Fatal("webDoneCh did not close within 3s after cancel")
	}
}

// TestWebDoneChClosesAfterShutdown ensures that webDoneCh is closed when
// the runWeb goroutine exits, allowing the main loop to proceed without
// relying on fixed-duration sleeps.
func TestWebDoneChClosesAfterShutdown(t *testing.T) {
	comm.ResetCtx()
	defer comm.ResetCtx()

	origHost := comm.WebHost
	comm.WebHost = ":0"
	defer func() { comm.WebHost = origHost }()

	webReadyCh = make(chan struct{})
	webDoneCh = make(chan struct{})

	go runWeb()

	// Wait for web to be ready
	<-webReadyCh

	// Cancel context to trigger shutdown
	comm.Cancel()

	// Verify webDoneCh closes within the shutdown timeout window
	select {
	case <-webDoneCh:
		// success — shutdown completed without sleeping
	case <-time.After(shutdownTimeout + time.Second):
		t.Fatalf("webDoneCh did not close within %v + 1s", shutdownTimeout)
	}
}

// TestWebReadyChConstantDefinitions verifies that the UI-related constants
// are set to reasonable values to catch accidental misconfiguration.
func TestWebReadyChConstantDefinitions(t *testing.T) {
	if uiBufferSize <= 0 {
		t.Error("uiBufferSize must be positive")
	}
	if uiBufferSize > 1<<20 {
		t.Errorf("uiBufferSize=%d is suspiciously large (1 MiB limit)", uiBufferSize)
	}
	if uiCacheMaxAge <= 0 {
		t.Error("uiCacheMaxAge must be positive")
	}
	if uiMaxSize <= 0 {
		t.Error("uiMaxSize must be positive")
	}
	if uiMaxSize < 1<<20 {
		t.Errorf("uiMaxSize=%d is suspiciously small (< 1 MiB)", uiMaxSize)
	}
	if shutdownGracePeriod <= 0 {
		t.Error("shutdownGracePeriod must be positive")
	}
}

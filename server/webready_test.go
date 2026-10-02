package server

import (
	"testing"
	"time"

	"github.com/gokins/core"
	"github.com/gokins/gokins/comm"
)

// TestWebReadyCh_SignalsAfterBind verifies that WebReadyCh is closed
// after the web server has bound to its port, eliminating the race condition
// that existed with time.Sleep(10ms).
func TestWebReadyCh_SignalsAfterBind(t *testing.T) {
	// Reset state for test isolation
	comm.ResetCtx()
	comm.WebReadyCh = make(chan struct{})
	comm.WebEgn = nil
	comm.WebHost = ":0" // Use any available port
	comm.Installed = true

	origDebug := core.Debug
	t.Cleanup(func() {
		core.Debug = origDebug
	})
	core.Debug = false

	// Start web server in background
	go runWeb()

	// Wait for WebReadyCh to be closed (with timeout)
	select {
	case <-comm.WebReadyCh:
		// Success: channel was closed, server is ready
	case <-time.After(2 * time.Second):
		t.Fatal("WebReadyCh was not closed within 2 seconds - server may not have started")
	}

	// Verify WebEgn was initialized
	if comm.WebEgn == nil {
		t.Error("WebEgn was not initialized after WebReadyCh signaled")
	}

	// Clean up
	comm.Cancel()
	time.Sleep(100 * time.Millisecond)
}

// TestWebReadyCh_ClosedOnBindError verifies that WebReadyCh is closed
// even when the server fails to bind (e.g., port already in use).
func TestWebReadyCh_ClosedOnBindError(t *testing.T) {
	comm.ResetCtx()
	comm.WebReadyCh = make(chan struct{})
	comm.WebEgn = nil
	// Use an invalid address to force bind error
	comm.WebHost = "999.999.999.999:99999"
	comm.Installed = true

	// Start web server in background
	go runWeb()

	// WebReadyCh should still be closed even on error
	select {
	case <-comm.WebReadyCh:
		// Success: channel was closed despite bind error
	case <-time.After(1 * time.Second):
		t.Fatal("WebReadyCh was not closed on bind error - may cause deadlock")
	}

	comm.Cancel()
}

// TestWebReadyCh_MultipleWaits verifies that multiple goroutines can
// wait on WebReadyCh simultaneously without issues.
func TestWebReadyCh_MultipleWaits(t *testing.T) {
	comm.ResetCtx()
	comm.WebReadyCh = make(chan struct{})
	comm.WebEgn = nil
	comm.WebHost = ":0"
	comm.Installed = true

	// Pre-close the channel to simulate already-ready state
	close(comm.WebReadyCh)

	// Multiple goroutines should be able to read from a closed channel
	done := make(chan bool, 3)
	for i := 0; i < 3; i++ {
		go func() {
			select {
			case <-comm.WebReadyCh:
				done <- true
			case <-time.After(100 * time.Millisecond):
				done <- false
			}
		}()
	}

	// All should succeed
	for i := 0; i < 3; i++ {
		if !<-done {
			t.Errorf("goroutine %d failed to read from closed WebReadyCh", i)
		}
	}
}

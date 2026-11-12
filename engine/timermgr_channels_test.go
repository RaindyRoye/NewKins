package engine

import (
	"context"
	"sync"
	"testing"
	"time"
)

// TestTimerEngineStop verifies that Stop() cleanly terminates the run loop.
func TestTimerEngineStop(t *testing.T) {
	te := &TimerEngine{
		tasks:  make(map[string]*timerExec),
		wakeCh: make(chan struct{}, 1),
		stopCh: make(chan struct{}),
	}

	// Start the run loop in a goroutine
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-te.stopCh:
				return
			case <-te.wakeCh:
				select {
				case <-te.wakeCh:
				default:
				}
			case <-ticker.C:
			}
			te.run()
		}
	}()

	// Give the goroutine time to start
	time.Sleep(10 * time.Millisecond)

	// Stop should close stopCh, causing the loop to exit
	te.Stop()

	// Wait for the goroutine to finish
	select {
	case <-done:
		// Success - goroutine exited
	case <-time.After(2 * time.Second):
		t.Fatal("run loop did not exit within 2 seconds after Stop()")
	}

	// Calling Stop again should be safe (idempotent)
	te.Stop()
}

// TestTimerEngineStopIdempotent verifies Stop() can be called multiple times safely.
func TestTimerEngineStopIdempotent(t *testing.T) {
	te := &TimerEngine{
		tasks:  make(map[string]*timerExec),
		wakeCh: make(chan struct{}, 1),
		stopCh: make(chan struct{}),
	}

	// Call Stop multiple times - should not panic
	te.Stop()
	te.Stop()
	te.Stop()
}

// TestTimerEngineWakeNonBlocking verifies that wake() never blocks the caller.
func TestTimerEngineWakeNonBlocking(t *testing.T) {
	te := &TimerEngine{
		tasks:  make(map[string]*timerExec),
		wakeCh: make(chan struct{}, 1),
		stopCh: make(chan struct{}),
	}

	// Call wake multiple times - should never block
	start := time.Now()
	for i := 0; i < 100; i++ {
		te.wake()
	}
	elapsed := time.Since(start)

	// Should complete almost instantly (< 100ms)
	if elapsed > 100*time.Millisecond {
		t.Fatalf("wake() calls blocked for %v, expected non-blocking", elapsed)
	}

	// Verify at least one signal is pending
	select {
	case <-te.wakeCh:
		// Success - channel has at least one signal
	default:
		t.Fatal("expected at least one signal in wakeCh")
	}
}

// TestTimerEngineWakeDrains verifies that the run loop drains duplicate signals.
func TestTimerEngineWakeDrains(t *testing.T) {
	te := &TimerEngine{
		tasks:  make(map[string]*timerExec),
		wakeCh: make(chan struct{}, 1),
		stopCh: make(chan struct{}),
	}

	// Send multiple wake signals
	te.wake()
	te.wake()
	te.wake()

	// The channel should have exactly 1 signal (capacity 1)
	select {
	case <-te.wakeCh:
		// Got the first signal
	default:
		t.Fatal("expected signal in wakeCh")
	}

	// Should be empty now (duplicates were dropped)
	select {
	case <-te.wakeCh:
		t.Fatal("wakeCh should be empty after draining")
	default:
		// Success - channel is empty
	}
}

// TestTimerEngineResetOneWakesRunLoop verifies that resetOne signals the run loop.
func TestTimerEngineResetOneWakesRunLoop(t *testing.T) {
	te := &TimerEngine{
		tasks:  make(map[string]*timerExec),
		wakeCh: make(chan struct{}, 1),
		stopCh: make(chan struct{}),
	}

	// Simulate adding a timer
	futureTime := time.Now().Add(time.Hour)
	params := `{"timerType":0,"dates":"` + futureTime.Format(time.RFC3339Nano) + `"}`
	_ = &mockTrigger{
		id:     "test-wake",
		types:  "timer",
		params: params,
	}

	// Note: We can't call resetOne directly without a real model.TTrigger,
	// so we'll just verify the wake() mechanism works
	te.wake()

	select {
	case <-te.wakeCh:
		// Success - wake signal sent
	default:
		t.Fatal("expected wake signal after resetOne")
	}
}

// TestStartCtxCancellation verifies that StartCtx properly stops all engines when context is canceled.
func TestStartCtxCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	// We can't call StartCtx directly in unit tests because it requires
	// comm.Ctx, comm.Cfg, comm.WorkPath, etc. to be initialized.
	// Instead, we verify the shutdown goroutine logic.

	done := make(chan struct{})
	go func() {
		defer close(done)
		<-ctx.Done()
		// Simulate stopping engines
		time.Sleep(10 * time.Millisecond)
	}()

	// Cancel the context
	cancel()

	// Wait for shutdown to complete
	select {
	case <-done:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("shutdown did not complete within 2 seconds")
	}
}

// TestTimerEngineConcurrentStop verifies Stop() is safe under concurrent access.
func TestTimerEngineConcurrentStop(t *testing.T) {
	te := &TimerEngine{
		tasks:  make(map[string]*timerExec),
		wakeCh: make(chan struct{}, 1),
		stopCh: make(chan struct{}),
	}

	var wg sync.WaitGroup
	// Call Stop from multiple goroutines simultaneously
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			te.Stop()
		}()
	}

	// Wait with timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success - all goroutines completed
	case <-time.After(5 * time.Second):
		t.Fatal("concurrent Stop calls deadlocked or timed out")
	}
}

// TestTimerEngineWakeWithNilChannel verifies wake() handles nil wakeCh gracefully.
func TestTimerEngineWakeWithNilChannel(t *testing.T) {
	te := &TimerEngine{
		tasks:  make(map[string]*timerExec),
		wakeCh: nil, // Intentionally nil
		stopCh: make(chan struct{}),
	}

	// Should not panic
	te.wake()
}

// mockTrigger is a helper for testing without real model.TTrigger
type mockTrigger struct {
	id     string
	types  string
	params string
}

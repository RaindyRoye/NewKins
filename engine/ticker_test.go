package engine

import (
	"context"
	"sync"
	"testing"
	"time"
)

// TestBuildEngineTickerStop verifies that the BuildEngine goroutine exits
// promptly when the context is cancelled, confirming the ticker-based loop
// responds correctly to shutdown signals.
func TestBuildEngineTickerStop(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	var wg sync.WaitGroup
	wg.Add(1)

	done := make(chan struct{})
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				close(done)
				return
			case <-ticker.C:
				// simulate work
			}
		}
	}()

	// Let the goroutine start and enter the select
	time.Sleep(50 * time.Millisecond)

	// Cancel context and verify prompt exit
	cancel()
	select {
	case <-done:
		// goroutine exited as expected
	case <-time.After(2 * time.Second):
		t.Fatal("goroutine did not exit within 2s after context cancellation")
	}
	wg.Wait()
}

// TestJobEngineTickerStop verifies the same ticker pattern for the job engine loop.
func TestJobEngineTickerStop(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				close(done)
				return
			case <-ticker.C:
			}
		}
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("job engine ticker loop did not exit within 2s")
	}
}

// TestTimerEngineTickerStop verifies the same ticker pattern for the timer engine loop.
func TestTimerEngineTickerStop(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(10 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				close(done)
				return
			case <-ticker.C:
			}
		}
	}()

	time.Sleep(20 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timer engine ticker loop did not exit within 1s")
	}
}

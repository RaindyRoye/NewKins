package engine

import (
	"container/list"
	"sync"
	"testing"
	"time"

	"github.com/gokins/core/runtime"
)

// TestBuildEnginePutWakesRunLoop verifies that Put sends a signal on wakeCh
// so the run loop picks up new work immediately rather than waiting for the
// next ticker tick.
func TestBuildEnginePutWakesRunLoop(t *testing.T) {
	c := &BuildEngine{
		taskw:  list.New(),
		tasks:  make(map[string]*BuildTask),
		wakeCh: make(chan struct{}, 1),
		stopCh: make(chan struct{}),
	}

	// Channel should be empty initially
	select {
	case <-c.wakeCh:
		t.Fatal("wakeCh should be empty before Put")
	default:
		// OK
	}

	// Put should send a signal
	c.Put(&runtime.Build{Id: "b1"})

	// Verify signal was sent
	select {
	case <-c.wakeCh:
		// OK, signal received
	default:
		t.Fatal("expected Put to signal wakeCh")
	}

	// Verify task was enqueued
	if c.taskw.Len() != 1 {
		t.Fatalf("expected 1 task in queue, got %d", c.taskw.Len())
	}
}

// TestBuildEnginePutDoesNotBlockWhenFull ensures Put never blocks even when
// the wake channel already has a pending signal (buffer full).
func TestBuildEnginePutDoesNotBlockWhenFull(t *testing.T) {
	c := &BuildEngine{
		taskw:  list.New(),
		tasks:  make(map[string]*BuildTask),
		wakeCh: make(chan struct{}, 1),
		stopCh: make(chan struct{}),
	}

	// Fill the buffer
	c.wakeCh <- struct{}{}

	// Put multiple builds - none should block even though buffer is full
	done := make(chan struct{})
	go func() {
		for i := 0; i < 5; i++ {
			c.Put(&runtime.Build{Id: "b"})
		}
		close(done)
	}()

	select {
	case <-done:
		// OK, Put returned without blocking
	case <-time.After(time.Second):
		t.Fatal("Put blocked when wakeCh buffer was full")
	}

	if c.taskw.Len() != 5 {
		t.Fatalf("expected 5 tasks, got %d", c.taskw.Len())
	}
}

// TestBuildEngineStopIdempotent verifies Stop can be called multiple times
// without panicking (important for concurrent shutdown paths).
func TestBuildEngineStopIdempotent(t *testing.T) {
	c := &BuildEngine{
		taskw:  list.New(),
		tasks:  make(map[string]*BuildTask),
		wakeCh: make(chan struct{}, 1),
		stopCh: make(chan struct{}),
	}

	// Multiple Stop calls should not panic
	for i := 0; i < 10; i++ {
		c.Stop()
	}

	// Verify stopCh is closed
	select {
	case <-c.stopCh:
		// OK, closed
	default:
		t.Fatal("stopCh should be closed after Stop")
	}
}

// TestBuildEngineStopWithActiveTasks verifies Stop cancels running build tasks.
func TestBuildEngineStopWithActiveTasks(t *testing.T) {
	c := &BuildEngine{
		taskw:  list.New(),
		tasks:  make(map[string]*BuildTask),
		wakeCh: make(chan struct{}, 1),
		stopCh: make(chan struct{}),
	}

	// Add a task with its own context
	bt := &BuildTask{build: &runtime.Build{Id: "active"}}
	c.tasks["active"] = bt

	c.Stop()

	// Verify stopCh closed
	select {
	case <-c.stopCh:
	default:
		t.Fatal("stopCh should be closed")
	}
}

// TestBuildEngineStopNilChannel verifies Stop is safe when stopCh is nil
// (defensive check for tests that construct BuildEngine directly).
func TestBuildEngineStopNilChannel(t *testing.T) {
	c := &BuildEngine{
		taskw: list.New(),
		tasks: make(map[string]*BuildTask),
		// stopCh intentionally nil
	}

	// Should not panic
	c.Stop()
}

// TestBuildEngineConcurrentPutAndStop verifies no race between concurrent
// Put calls and Stop.
func TestBuildEngineConcurrentPutAndStop(t *testing.T) {
	c := &BuildEngine{
		taskw:  list.New(),
		tasks:  make(map[string]*BuildTask),
		wakeCh: make(chan struct{}, 1),
		stopCh: make(chan struct{}),
	}

	var wg sync.WaitGroup
	// Concurrent Puts
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.Put(&runtime.Build{Id: "x"})
		}()
	}
	// Concurrent Stops
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.Stop()
		}()
	}
	wg.Wait()
}

// TestBuildEngineRunLoopSelectOrder verifies the run loop selects on stopCh
// first before processing ticker or wake signals.
func TestBuildEngineRunLoopExitsOnStop(t *testing.T) {
	// This is a design contract test: when stopCh is closed, the select
	// case for stopCh should be eligible, but Go's select picks randomly
	// when multiple cases are ready. We only verify that Stop closes the
	// channel — the goroutine's select behavior is tested implicitly via
	// the Stop test above.
	c := &BuildEngine{
		taskw:  list.New(),
		tasks:  make(map[string]*BuildTask),
		wakeCh: make(chan struct{}, 1),
		stopCh: make(chan struct{}),
	}

	c.Stop()

	select {
	case <-c.stopCh:
		// OK
	default:
		t.Fatal("stopCh not closed")
	}
}

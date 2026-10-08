package engine

import (
	"container/list"
	"sync"
	"testing"
	"time"

	"github.com/gokins/core/runtime"
)

// TestJobEnginePutWakesRunLoop verifies that Put sends a signal on wakeCh
// so the run loop picks up new jobs immediately.
func TestJobEnginePutWakesRunLoop(t *testing.T) {
	c := &JobEngine{
		execs:  make(map[string]*executer),
		jobs:   make(map[string]*jobSync),
		wakeCh: make(chan struct{}, 1),
	}

	// Channel should be empty initially
	select {
	case <-c.wakeCh:
		t.Fatal("wakeCh should be empty before Put")
	default:
		// OK
	}

	// Add an executer first
	ex := &executer{
		plug:  "plugin-a",
		jobwt: list.New(),
	}
	c.execs["plugin-a"] = ex

	// Put should send a signal
	job := &jobSync{
		step: &runtime.Step{
			Step: "plugin-a",
		},
		cmdmp: make(map[string]*cmdSync),
	}
	err := c.Put(job)
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	// Verify signal was sent
	select {
	case <-c.wakeCh:
		// OK, signal received
	default:
		t.Fatal("expected Put to signal wakeCh")
	}

	// Verify job was enqueued
	ex.RLock()
	ln := ex.jobwt.Len()
	ex.RUnlock()
	if ln != 1 {
		t.Fatalf("expected 1 job in queue, got %d", ln)
	}
}

// TestJobEnginePutDoesNotBlockWhenFull ensures Put never blocks even when
// the wake channel already has a pending signal (buffer full).
func TestJobEnginePutDoesNotBlockWhenFull(t *testing.T) {
	c := &JobEngine{
		execs:  make(map[string]*executer),
		jobs:   make(map[string]*jobSync),
		wakeCh: make(chan struct{}, 1),
	}

	ex := &executer{
		plug:  "plugin-a",
		jobwt: list.New(),
	}
	c.execs["plugin-a"] = ex

	// Fill the buffer
	c.wakeCh <- struct{}{}

	// Put multiple jobs - none should block even though buffer is full
	done := make(chan struct{})
	go func() {
		for i := 0; i < 5; i++ {
			job := &jobSync{
				step:  &runtime.Step{Step: "plugin-a"},
				cmdmp: make(map[string]*cmdSync),
			}
			_ = c.Put(job)
		}
		close(done)
	}()

	select {
	case <-done:
		// OK, Put returned without blocking
	case <-time.After(time.Second):
		t.Fatal("Put blocked when wakeCh buffer was full")
	}

	ex.RLock()
	ln := ex.jobwt.Len()
	ex.RUnlock()
	if ln != 5 {
		t.Fatalf("expected 5 jobs, got %d", ln)
	}
}

// TestJobEngineConcurrentPut verifies no race between concurrent Put calls.
func TestJobEngineConcurrentPut(t *testing.T) {
	c := &JobEngine{
		execs:  make(map[string]*executer),
		jobs:   make(map[string]*jobSync),
		wakeCh: make(chan struct{}, 1),
	}

	ex := &executer{
		plug:  "plugin-a",
		jobwt: list.New(),
	}
	c.execs["plugin-a"] = ex

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			job := &jobSync{
				step:  &runtime.Step{Step: "plugin-a"},
				cmdmp: make(map[string]*cmdSync),
			}
			_ = c.Put(job)
		}()
	}
	wg.Wait()

	ex.RLock()
	ln := ex.jobwt.Len()
	ex.RUnlock()
	if ln != 50 {
		t.Fatalf("expected 50 jobs, got %d", ln)
	}
}

package engine

import (
	"container/list"
	"context"
	"testing"
	"time"

	"github.com/gokins/core/runtime"
)

// TestBuildEngineStopCancelsActiveTaskContexts verifies that Stop cancels the
// context of all active build tasks.
func TestBuildEngineStopCancelsActiveTaskContexts(t *testing.T) {
	c := &BuildEngine{
		taskw:  list.New(),
		tasks:  make(map[string]*BuildTask),
		wakeCh: make(chan struct{}, 1),
		stopCh: make(chan struct{}),
	}

	ctx1, cancel1 := context.WithCancel(context.Background())
	defer cancel1()
	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel2()

	c.tasks["b1"] = &BuildTask{
		build: &runtime.Build{Id: "b1"},
		ctx:   ctx1,
		cncl:  cancel1,
	}
	c.tasks["b2"] = &BuildTask{
		build: &runtime.Build{Id: "b2"},
		ctx:   ctx2,
		cncl:  cancel2,
	}

	c.Stop()

	// Both contexts should be canceled
	select {
	case <-ctx1.Done():
		// OK
	case <-time.After(time.Second):
		t.Error("expected ctx1 to be canceled")
	}
	select {
	case <-ctx2.Done():
		// OK
	case <-time.After(time.Second):
		t.Error("expected ctx2 to be canceled")
	}
}

// TestBuildEngineStopIdempotentAcrossGoroutines verifies Stop is safe to call
// concurrently from many goroutines.
func TestBuildEngineStopIdempotentAcrossGoroutines(t *testing.T) {
	c := &BuildEngine{
		taskw:  list.New(),
		tasks:  make(map[string]*BuildTask),
		wakeCh: make(chan struct{}, 1),
		stopCh: make(chan struct{}),
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 100; i++ {
			c.Stop()
		}
	}()

	for i := 0; i < 100; i++ {
		c.Stop()
	}
	<-done
}

// TestBuildEngineGetNilReceiver verifies Get is safe on a nil receiver.
func TestBuildEngineGetNilReceiver(t *testing.T) {
	var c *BuildEngine
	_, ok := c.Get("anything")
	if ok {
		t.Error("expected Get on nil BuildEngine to return false")
	}
}

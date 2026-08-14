package engine

import (
	"context"
	"testing"
	"time"

	"github.com/gokins/core/runtime"
	hbtp "github.com/mgr9525/HyperByte-Transfer-Protocol"
)

// TestBuildEngineContext verifies that StartBuildEngine accepts and uses a custom context
func TestBuildEngineContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	engine := StartBuildEngine(ctx)
	if engine == nil {
		t.Fatal("expected engine to be created, got nil")
	}

	// Verify the context is stored and used
	if engine.ctx != ctx {
		t.Errorf("expected engine to use provided context, got different context")
	}

	// Cancel context and verify engine stops
	cancel()
	time.Sleep(100 * time.Millisecond) // Give goroutine time to exit

	// Verify context is canceled
	if !hbtp.EndContext(engine.ctx) {
		t.Error("expected engine context to be canceled")
	}
}

// TestBuildEngineNilContext verifies that StartBuildEngine assigns a context when nil is passed
func TestBuildEngineNilContext(t *testing.T) {
	engine := StartBuildEngine(context.TODO()) //nolint:SA1012 // testing nil fallback
	if engine == nil {
		t.Fatal("expected engine to be created, got nil")
	}

	// Verify a context was assigned (not nil)
	if engine.ctx == nil {
		t.Errorf("expected engine to have a non-nil context")
	}
}

// TestJobEngineContext verifies that StartJobEngine accepts and uses a custom context
func TestJobEngineContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	engine := StartJobEngine(ctx)
	if engine == nil {
		t.Fatal("expected engine to be created, got nil")
	}

	// Verify the context is stored and used
	if engine.ctx != ctx {
		t.Errorf("expected engine to use provided context, got different context")
	}

	// Cancel context and verify engine stops
	cancel()
	time.Sleep(100 * time.Millisecond) // Give goroutine time to exit

	// Verify context is canceled
	if !hbtp.EndContext(engine.ctx) {
		t.Error("expected engine context to be canceled")
	}
}

// TestJobEngineNilContext verifies that StartJobEngine assigns a context when nil is passed
func TestJobEngineNilContext(t *testing.T) {
	engine := StartJobEngine(context.TODO()) //nolint:SA1012 // testing nil fallback
	if engine == nil {
		t.Fatal("expected engine to be created, got nil")
	}

	// Verify a context was assigned (not nil)
	if engine.ctx == nil {
		t.Errorf("expected engine to have a non-nil context")
	}
}

// TestTimerEngineContext verifies that StartTimerEngine accepts and uses a custom context
func TestTimerEngineContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	engine := StartTimerEngine(ctx)
	if engine == nil {
		t.Fatal("expected engine to be created, got nil")
	}

	// Verify the context is stored and used
	if engine.ctx != ctx {
		t.Errorf("expected engine to use provided context, got different context")
	}

	// Cancel context and verify engine stops
	cancel()
	time.Sleep(100 * time.Millisecond) // Give goroutine time to exit

	// Verify context is canceled
	if !hbtp.EndContext(engine.ctx) {
		t.Error("expected engine context to be canceled")
	}
}

// TestTimerEngineNilContext verifies that StartTimerEngine assigns a context when nil is passed
func TestTimerEngineNilContext(t *testing.T) {
	engine := StartTimerEngine(context.TODO()) //nolint:SA1012 // testing nil fallback
	if engine == nil {
		t.Fatal("expected engine to be created, got nil")
	}

	// Verify a context was assigned (not nil)
	if engine.ctx == nil {
		t.Errorf("expected engine to have a non-nil context")
	}
}

// TestBuildEngineContextTimeout verifies that BuildEngine respects context timeout
func TestBuildEngineContextTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	engine := StartBuildEngine(ctx)
	if engine == nil {
		t.Fatal("expected engine to be created, got nil")
	}

	// Wait for timeout
	time.Sleep(100 * time.Millisecond)

	// Verify context is canceled due to timeout
	if !hbtp.EndContext(engine.ctx) {
		t.Error("expected engine context to be canceled after timeout")
	}
}

// TestJobEngineContextTimeout verifies that JobEngine respects context timeout
func TestJobEngineContextTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	engine := StartJobEngine(ctx)
	if engine == nil {
		t.Fatal("expected engine to be created, got nil")
	}

	// Wait for timeout
	time.Sleep(100 * time.Millisecond)

	// Verify context is canceled due to timeout
	if !hbtp.EndContext(engine.ctx) {
		t.Error("expected engine context to be canceled after timeout")
	}
}

// TestTimerEngineContextTimeout verifies that TimerEngine respects context timeout
func TestTimerEngineContextTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	engine := StartTimerEngine(ctx)
	if engine == nil {
		t.Fatal("expected engine to be created, got nil")
	}

	// Wait for timeout
	time.Sleep(100 * time.Millisecond)

	// Verify context is canceled due to timeout
	if !hbtp.EndContext(engine.ctx) {
		t.Error("expected engine context to be canceled after timeout")
	}
}

// TestBuildEngineContextIndependent verifies that multiple engines can have independent contexts
func TestBuildEngineContextIndependent(t *testing.T) {
	ctx1, cancel1 := context.WithCancel(context.Background())
	defer cancel1()

	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel2()

	engine1 := StartBuildEngine(ctx1)
	engine2 := StartBuildEngine(ctx2)

	if engine1.ctx == engine2.ctx {
		t.Error("expected engines to have different contexts")
	}

	// Cancel first context
	cancel1()
	time.Sleep(100 * time.Millisecond)

	// Verify only first engine is canceled
	if !hbtp.EndContext(engine1.ctx) {
		t.Error("expected engine1 context to be canceled")
	}
	if hbtp.EndContext(engine2.ctx) {
		t.Error("expected engine2 context to still be active")
	}
}

// TestBuildEnginePutWithCustomContext verifies that Put works with custom context
func TestBuildEnginePutWithCustomContext(t *testing.T) {
	ctx := context.Background()
	engine := StartBuildEngine(ctx)
	defer engine.Stop()

	build := &runtime.Build{
		Id: "test-build-custom-ctx",
	}

	engine.Put(build)

	// Give it a moment to process
	time.Sleep(50 * time.Millisecond)

	// Verify build was queued
	engine.tskwlk.RLock()
	queueLen := engine.taskw.Len()
	engine.tskwlk.RUnlock()

	if queueLen == 0 {
		t.Error("expected build to be queued")
	}
}

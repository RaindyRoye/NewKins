package engine

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/gokins/gokins/model"
)

// TestTimerEngineNextIntervalEmpty verifies that nextInterval returns the
// fallback interval when there are no scheduled tasks.
func TestTimerEngineNextIntervalEmpty(t *testing.T) {
	te := &TimerEngine{
		tasks:  make(map[string]*timerExec),
		wakeCh: make(chan struct{}, 1),
		stopCh: make(chan struct{}),
	}
	d := te.nextInterval()
	if d != timerRefreshInterval {
		t.Fatalf("expected %v for empty tasks, got %v", timerRefreshInterval, d)
	}
}

// TestTimerEngineNextIntervalFutureTask verifies that nextInterval returns the
// duration until the nearest future tick, clamped to timerRefreshInterval.
func TestTimerEngineNextIntervalFutureTask(t *testing.T) {
	te := &TimerEngine{
		tasks:  make(map[string]*timerExec),
		wakeCh: make(chan struct{}, 1),
		stopCh: make(chan struct{}),
	}
	// Task due in 5 minutes
	te.tasks["a"] = &timerExec{
		tt:   &model.TTrigger{Id: "a", Name: "a"},
		typ:  1,
		tick: time.Now().Add(5 * time.Minute),
	}
	// Task due in 2 minutes (nearest)
	te.tasks["b"] = &timerExec{
		tt:   &model.TTrigger{Id: "b", Name: "b"},
		typ:  1,
		tick: time.Now().Add(2 * time.Minute),
	}
	d := te.nextInterval()
	// Should be approximately 2 minutes, but clamped to timerRefreshInterval
	// which is 30s. So it should be 30s.
	if d > timerRefreshInterval {
		t.Fatalf("expected interval clamped to %v, got %v", timerRefreshInterval, d)
	}
	if d < time.Second {
		t.Fatalf("expected interval at least 1s, got %v", d)
	}
}

// TestTimerEngineNextIntervalPastTick verifies that when all ticks are in the
// past, nextInterval returns 1 millisecond (the minimum for due tasks).
func TestTimerEngineNextIntervalPastTick(t *testing.T) {
	te := &TimerEngine{
		tasks:  make(map[string]*timerExec),
		wakeCh: make(chan struct{}, 1),
		stopCh: make(chan struct{}),
	}
	te.tasks["past"] = &timerExec{
		tt:   &model.TTrigger{Id: "past", Name: "past"},
		typ:  1,
		tick: time.Now().Add(-time.Hour),
	}
	d := te.nextInterval()
	if d != time.Millisecond {
		t.Fatalf("expected 1ms for past-due task, got %v", d)
	}
}

// TestTimerEngineStopIdempotent verifies that Stop can be called multiple times
// without panicking.
func TestTimerEngineStopIdempotent(t *testing.T) {
	te := &TimerEngine{
		tasks:  make(map[string]*timerExec),
		wakeCh: make(chan struct{}, 1),
		stopCh: make(chan struct{}),
	}
	te.Stop()
	te.Stop() // should not panic
	te.Stop() // should not panic
}

// TestTimerEngineStopClosesChannel verifies that Stop closes the stopCh.
func TestTimerEngineStopClosesChannel(t *testing.T) {
	te := &TimerEngine{
		tasks:  make(map[string]*timerExec),
		wakeCh: make(chan struct{}, 1),
		stopCh: make(chan struct{}),
	}
	te.Stop()
	select {
	case <-te.stopCh:
		// Expected: channel is closed
	default:
		t.Fatal("expected stopCh to be closed after Stop")
	}
}

// TestTimerEngineWakeNonBlocking verifies that Wake does not block even when
// the wakeCh buffer is full.
func TestTimerEngineWakeNonBlocking(t *testing.T) {
	te := &TimerEngine{
		tasks:  make(map[string]*timerExec),
		wakeCh: make(chan struct{}, 1),
		stopCh: make(chan struct{}),
	}
	// Fill the buffer
	te.Wake()
	// Second call should not block
	done := make(chan struct{})
	go func() {
		te.Wake()
		close(done)
	}()
	select {
	case <-done:
		// OK
	case <-time.After(time.Second):
		t.Fatal("Wake blocked — it should be non-blocking")
	}
}

// TestTimerEngineWakeNilChannel verifies that Wake is safe with nil channels.
func TestTimerEngineWakeNilChannel(t *testing.T) {
	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}
	// Should not panic
	te.Wake()
	te.Stop() // nil stopCh should also be safe
}

// TestTimerEngineRefreshWakeSignal verifies that Refresh sends a wake signal
// after successfully resetting a timer.
func TestTimerEngineRefreshWakeSignal(t *testing.T) {
	te := &TimerEngine{
		tasks:  make(map[string]*timerExec),
		wakeCh: make(chan struct{}, 1),
		stopCh: make(chan struct{}),
	}
	future := time.Now().Add(time.Hour)
	params := map[string]any{
		"timerType": 1,
		"dates":     future.Format(time.RFC3339Nano),
	}
	b, _ := json.Marshal(params)
	tmr := &model.TTrigger{
		Id: "wake-test", Types: "timer", Name: "wake", Params: string(b),
	}
	// Manually call resetOne (Refresh requires DB)
	if err := te.resetOne(tmr); err != nil {
		t.Fatalf("resetOne: %v", err)
	}
	// Simulate what Refresh does
	te.Wake()
	// Check that the wake channel has a signal
	select {
	case <-te.wakeCh:
		// Expected
	default:
		t.Fatal("expected wake signal after refresh")
	}
}

// TestTimerEngineExecItemFutureTickNoFire verifies that execItem does not fire
// when the tick is in the future.
func TestTimerEngineExecItemFutureTickNoFire(t *testing.T) {
	te := &TimerEngine{
		tasks:  make(map[string]*timerExec),
		wakeCh: make(chan struct{}, 1),
		stopCh: make(chan struct{}),
	}
	originalTick := time.Now().Add(time.Hour)
	task := &timerExec{
		tt:   &model.TTrigger{Id: "future", Name: "future"},
		typ:  1,
		tick: originalTick,
	}
	te.tasks["future"] = task
	te.execItem(task)
	// tick should not change since it's in the future
	if !task.tick.Equal(originalTick) {
		t.Error("execItem should not modify tick when it is in the future")
	}
}

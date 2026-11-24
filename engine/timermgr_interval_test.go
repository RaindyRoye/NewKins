package engine

import (
	"testing"
	"time"

	"github.com/gokins/gokins/model"
)

// TestTimerEngineNextIntervalZeroTick verifies nextInterval skips zero-value ticks.
func TestTimerEngineNextIntervalZeroTick(t *testing.T) {
	te := &TimerEngine{
		tasks:  make(map[string]*timerExec),
		wakeCh: make(chan struct{}, 1),
		stopCh: make(chan struct{}),
	}
	// Task with zero tick (should be skipped)
	te.tasks["zero"] = &timerExec{
		tt:   &model.TTrigger{Id: "zero", Name: "zero"},
		typ:  1,
		tick: time.Time{},
	}
	// Task with future tick
	te.tasks["future"] = &timerExec{
		tt:   &model.TTrigger{Id: "future", Name: "future"},
		typ:  1,
		tick: time.Now().Add(10 * time.Second),
	}
	d := te.nextInterval()
	// Should pick the 10s task, not the zero tick
	if d < 9*time.Second || d > 11*time.Second {
		t.Fatalf("expected ~10s interval, got %v", d)
	}
}

// TestTimerEngineNextIntervalAllZeroTicks verifies nextInterval falls back to
// timerRefreshInterval when all tasks have zero-value ticks.
func TestTimerEngineNextIntervalAllZeroTicks(t *testing.T) {
	te := &TimerEngine{
		tasks:  make(map[string]*timerExec),
		wakeCh: make(chan struct{}, 1),
		stopCh: make(chan struct{}),
	}
	te.tasks["a"] = &timerExec{
		tt:   &model.TTrigger{Id: "a", Name: "a"},
		typ:  1,
		tick: time.Time{},
	}
	te.tasks["b"] = &timerExec{
		tt:   &model.TTrigger{Id: "b", Name: "b"},
		typ:  2,
		tick: time.Time{},
	}
	d := te.nextInterval()
	// All zero ticks → fallback to timerRefreshInterval
	if d != timerRefreshInterval {
		t.Fatalf("expected %v when all ticks are zero, got %v", timerRefreshInterval, d)
	}
}

// TestTimerEngineNextIntervalNearFutureTick verifies that a tick very close to
// now returns at least 1ms (never 0 or negative).
func TestTimerEngineNextIntervalNearFutureTick(t *testing.T) {
	te := &TimerEngine{
		tasks:  make(map[string]*timerExec),
		wakeCh: make(chan struct{}, 1),
		stopCh: make(chan struct{}),
	}
	// Tick 500 microseconds from now
	te.tasks["near"] = &timerExec{
		tt:   &model.TTrigger{Id: "near", Name: "near"},
		typ:  1,
		tick: time.Now().Add(500 * time.Microsecond),
	}
	d := te.nextInterval()
	if d < time.Millisecond {
		t.Fatalf("expected at least 1ms, got %v", d)
	}
}

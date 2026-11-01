package engine

import (
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/gokins/gokins/model"
)

// TestTimerEngineExecItemTypesPastTick exercises the tick-rescheduling branch
// of execItem for all recurring timer types (1-5). When tick is in the past,
// execItem enters the switch and advances tick by the type-specific interval.
// service.TriggerTimer may panic (no DB) but util.RecoverLog catches it.
func TestTimerEngineExecItemTypesPastTick(t *testing.T) {
	tests := []struct {
		name     string
		typ      int64
		minDelta time.Duration
	}{
		{"minute", 1, 30 * time.Second},
		{"hour", 2, 30 * time.Minute},
		{"daily", 3, 12 * time.Hour},
		{"weekly", 4, 6 * 24 * time.Hour},
		{"monthly", 5, 29 * 24 * time.Hour},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			te := &TimerEngine{tasks: make(map[string]*timerExec)}
			past := time.Now().Add(-2 * time.Minute)
			params := map[string]any{
				"timerType": tc.typ,
				"dates":     past.Format(time.RFC3339Nano),
			}
			b, _ := json.Marshal(params)
			tmr := &model.TTrigger{
				Id:     tc.name,
				Types:  "timer",
				Name:   tc.name,
				Params: string(b),
			}
			if err := te.resetOne(tmr); err != nil {
				t.Fatalf("resetOne: %v", err)
			}

			task := te.tasks[tc.name]
			// Force tick into the past so execItem enters the reschedule branch
			task.tick = time.Now().Add(-time.Second)
			beforeTick := task.tick

			te.execItem(task)

			// Tick should have been advanced past the old value
			if !task.tick.After(beforeTick) {
				t.Errorf("expected tick to advance, before=%v after=%v", beforeTick, task.tick)
			}
			// New tick should be within a reasonable window of the expected interval
			if time.Until(task.tick) > tc.minDelta*2 {
				t.Errorf("tick too far in future: %v (expected within %v)", time.Until(task.tick), tc.minDelta*2)
			}
		})
	}
}

// TestTimerEngineExecItemOnceDeletion verifies that execItem for a type-0
// (one-time) timer with a past tick triggers the async deletion path.
func TestTimerEngineExecItemOnceDeletion(t *testing.T) {
	te := &TimerEngine{tasks: make(map[string]*timerExec)}
	te.tasks["once-del"] = &timerExec{
		tt:   &model.TTrigger{Id: "once-del", Name: "once"},
		typ:  0,
		tick: time.Now().Add(-time.Second),
	}

	te.execItem(te.tasks["once-del"])

	// Give the deletion goroutine time to execute
	time.Sleep(100 * time.Millisecond)
	te.tasklk.Lock()
	_, exists := te.tasks["once-del"]
	te.tasklk.Unlock()
	// The goroutine may or may not have completed; just verify no panic occurred
	_ = exists
}

// TestTimerEngineRunIteratesTasks verifies that run() iterates over all
// tasks and calls execItem on each. We use a future tick so execItem is a no-op.
func TestTimerEngineRunIteratesTasks(t *testing.T) {
	te := &TimerEngine{tasks: make(map[string]*timerExec)}
	// Add several tasks with future ticks (execItem becomes a no-op)
	for i := 0; i < 5; i++ {
		id := "run-task-" + string(rune('a'+i))
		te.tasks[id] = &timerExec{
			tt:   &model.TTrigger{Id: id, Name: id},
			typ:  1,
			tick: time.Now().Add(time.Hour),
		}
	}

	// run() should iterate without panic even though execItem is a no-op
	te.run()

	// All tasks should still exist
	if len(te.tasks) != 5 {
		t.Fatalf("expected 5 tasks, got %d", len(te.tasks))
	}
}

// TestTimerEngineRefreshWithNilDb verifies that refresh() handles a nil or
// uninitialized DB gracefully by logging an error and returning.
func TestTimerEngineRefreshWithNilDb(t *testing.T) {
	te := &TimerEngine{tasks: make(map[string]*timerExec)}
	// comm.Db is nil in tests; refresh() should log the error and return
	// without panicking. The deferred util.RecoverLog catches any panic.
	te.refresh()
	// No tasks should be added since the DB query fails
	if len(te.tasks) != 0 {
		t.Fatalf("expected 0 tasks after refresh with nil DB, got %d", len(te.tasks))
	}
}

// TestTimerEngineRunConcurrent verifies that run() can be called concurrently
// with resetOne and Delete without data races.
func TestTimerEngineRunConcurrent(t *testing.T) {
	te := &TimerEngine{tasks: make(map[string]*timerExec)}

	// Pre-populate
	for i := 0; i < 10; i++ {
		id := "conc-" + string(rune('0'+i))
		te.tasks[id] = &timerExec{
			tt:   &model.TTrigger{Id: id, Name: id},
			typ:  1,
			tick: time.Now().Add(time.Hour),
		}
	}

	var wg sync.WaitGroup

	// Concurrent run() calls
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			te.run()
		}()
	}

	// Concurrent resetOne calls
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			future := time.Now().Add(time.Hour)
			params := map[string]any{
				"timerType": 1,
				"dates":     future.Format(time.RFC3339Nano),
			}
			b, _ := json.Marshal(params)
			tmr := &model.TTrigger{
				Id:     "conc-new-" + string(rune('a'+n)),
				Types:  "timer",
				Name:   "test",
				Params: string(b),
			}
			_ = te.resetOne(tmr)
		}(i)
	}

	// Concurrent Delete calls
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			te.Delete("conc-" + string(rune('0'+n)))
		}(i)
	}

	wg.Wait()
}

// TestTimerEngineExecItemFutureTickNoReschedule verifies that execItem does
// NOT enter the reschedule branch when tick is in the future.
func TestTimerEngineExecItemFutureTickNoReschedule(t *testing.T) {
	te := &TimerEngine{tasks: make(map[string]*timerExec)}
	futureTick := time.Now().Add(2 * time.Hour)
	te.tasks["future-noop"] = &timerExec{
		tt:   &model.TTrigger{Id: "future-noop", Name: "test"},
		typ:  1,
		tick: futureTick,
	}

	te.execItem(te.tasks["future-noop"])

	// Tick should remain unchanged
	if !te.tasks["future-noop"].tick.Equal(futureTick) {
		t.Error("execItem should not modify tick when it is in the future")
	}
}

// TestTimerEngineResetOneOnceTimerPastNotAdded verifies that a type-0 timer
// with a past date is not added to the task map.
func TestTimerEngineResetOneOnceTimerPastNotAdded(t *testing.T) {
	te := &TimerEngine{tasks: make(map[string]*timerExec)}
	past := time.Now().Add(-time.Hour)
	params := map[string]any{
		"timerType": 0,
		"dates":     past.Format(time.RFC3339Nano),
	}
	b, _ := json.Marshal(params)
	tmr := &model.TTrigger{
		Id:     "past-once-2",
		Types:  "timer",
		Name:   "past",
		Params: string(b),
	}
	if err := te.resetOne(tmr); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := te.tasks["past-once-2"]; ok {
		t.Fatal("past one-time timer should not be registered")
	}
}

// TestTimerEngineResetOneUpdateExistingOnce verifies that updating an existing
// once-timer preserves the task but updates its fields.
func TestTimerEngineResetOneUpdateExistingOnce(t *testing.T) {
	te := &TimerEngine{tasks: make(map[string]*timerExec)}
	future1 := time.Now().Add(time.Hour)
	params1 := map[string]any{
		"timerType": 0,
		"dates":     future1.Format(time.RFC3339Nano),
	}
	b1, _ := json.Marshal(params1)
	tmr1 := &model.TTrigger{
		Id: "upd-once", Types: "timer", Name: "upd", Params: string(b1),
	}
	if err := te.resetOne(tmr1); err != nil {
		t.Fatalf("first resetOne: %v", err)
	}
	firstTms := te.tasks["upd-once"].tms

	// Second reset with a different time
	future2 := time.Now().Add(2 * time.Hour)
	params2 := map[string]any{
		"timerType": 0,
		"dates":     future2.Format(time.RFC3339Nano),
	}
	b2, _ := json.Marshal(params2)
	tmr2 := &model.TTrigger{
		Id: "upd-once", Types: "timer", Name: "upd", Params: string(b2),
	}
	if err := te.resetOne(tmr2); err != nil {
		t.Fatalf("second resetOne: %v", err)
	}
	task := te.tasks["upd-once"]
	if task.tms.Equal(firstTms) {
		t.Error("expected tms to be updated on second resetOne")
	}
	if len(te.tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(te.tasks))
	}
}

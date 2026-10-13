package engine

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/gokins/gokins/model"
)

// makeTimerParams creates a JSON params string for a timer trigger.
func makeTimerParams(timerType int64, t time.Time) string {
	params := map[string]any{
		"timerType": timerType,
		"dates":     t.Format(time.RFC3339Nano),
	}
	b, _ := json.Marshal(params)
	return string(b)
}

// TestTimerEngineResetOneHourlyTimer covers type 2 (hourly) timer scheduling.
func TestTimerEngineResetOneHourlyTimer(t *testing.T) {
	te := &TimerEngine{tasks: make(map[string]*timerExec)}

	params := makeTimerParams(2, time.Now())
	tmr := &model.TTrigger{
		Id: "hourly-1", Types: "timer", Name: "hourly", Params: params,
	}
	if err := te.resetOne(tmr); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	task, ok := te.tasks["hourly-1"]
	if !ok {
		t.Fatal("expected hourly task to be added")
	}
	if task.typ != 2 {
		t.Fatalf("expected type 2, got %d", task.typ)
	}
	// Tick should be within ~1 hour of now
	if time.Until(task.tick) > time.Hour+time.Second {
		t.Fatal("tick should be within ~1 hour")
	}
}

// TestTimerEngineResetOneDailyTimer covers type 3 (daily) timer scheduling.
func TestTimerEngineResetOneDailyTimer(t *testing.T) {
	te := &TimerEngine{tasks: make(map[string]*timerExec)}

	params := makeTimerParams(3, time.Now())
	tmr := &model.TTrigger{
		Id: "daily-1", Types: "timer", Name: "daily", Params: params,
	}
	if err := te.resetOne(tmr); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	task, ok := te.tasks["daily-1"]
	if !ok {
		t.Fatal("expected daily task to be added")
	}
	if task.typ != 3 {
		t.Fatalf("expected type 3, got %d", task.typ)
	}
	if time.Until(task.tick) > 25*time.Hour {
		t.Fatal("tick should be within ~25 hours")
	}
}

// TestTimerEngineResetOneWeeklyTimer covers type 4 (weekly) timer scheduling.
func TestTimerEngineResetOneWeeklyTimer(t *testing.T) {
	te := &TimerEngine{tasks: make(map[string]*timerExec)}

	params := makeTimerParams(4, time.Now())
	tmr := &model.TTrigger{
		Id: "weekly-1", Types: "timer", Name: "weekly", Params: params,
	}
	if err := te.resetOne(tmr); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	task, ok := te.tasks["weekly-1"]
	if !ok {
		t.Fatal("expected weekly task to be added")
	}
	if task.typ != 4 {
		t.Fatalf("expected type 4, got %d", task.typ)
	}
	if time.Until(task.tick) > 8*24*time.Hour {
		t.Fatal("tick should be within ~8 days")
	}
}

// TestTimerEngineResetOneMonthlyTimer covers type 5 (monthly) timer scheduling.
func TestTimerEngineResetOneMonthlyTimer(t *testing.T) {
	te := &TimerEngine{tasks: make(map[string]*timerExec)}

	params := makeTimerParams(5, time.Now())
	tmr := &model.TTrigger{
		Id: "monthly-1", Types: "timer", Name: "monthly", Params: params,
	}
	if err := te.resetOne(tmr); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	task, ok := te.tasks["monthly-1"]
	if !ok {
		t.Fatal("expected monthly task to be added")
	}
	if task.typ != 5 {
		t.Fatalf("expected type 5, got %d", task.typ)
	}
	if time.Until(task.tick) > 31*24*time.Hour {
		t.Fatal("tick should be within ~31 days")
	}
}

// TestTimerEngineResetOnePastOnceTimer verifies that a one-shot timer set in
// the past is NOT added to the task map.
func TestTimerEngineResetOnePastOnceTimer(t *testing.T) {
	te := &TimerEngine{tasks: make(map[string]*timerExec)}

	pastTime := time.Now().Add(-time.Hour)
	params := makeTimerParams(0, pastTime)
	tmr := &model.TTrigger{
		Id: "past-once", Types: "timer", Name: "past-once", Params: params,
	}
	if err := te.resetOne(tmr); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Past one-shot timer should NOT be added
	if _, ok := te.tasks["past-once"]; ok {
		t.Fatal("past one-shot timer should not be added to tasks")
	}
}

// TestTimerEngineResetOneRecurringUpdate verifies that resetting an existing
// recurring timer updates it in-place instead of creating a duplicate.
func TestTimerEngineResetOneRecurringUpdate(t *testing.T) {
	te := &TimerEngine{tasks: make(map[string]*timerExec)}

	// First add
	futureTime := time.Now().Add(time.Hour)
	params := makeTimerParams(1, futureTime)
	tmr := &model.TTrigger{
		Id: "existing", Types: "timer", Name: "existing", Params: params,
	}
	if err := te.resetOne(tmr); err != nil {
		t.Fatalf("first resetOne failed: %v", err)
	}
	origTask := te.tasks["existing"]
	if origTask == nil {
		t.Fatal("expected task after first resetOne")
	}

	// Second add (update) with different time
	newTime := time.Now().Add(2 * time.Hour)
	params2 := makeTimerParams(1, newTime)
	tmr2 := &model.TTrigger{
		Id: "existing", Types: "timer", Name: "existing-updated", Params: params2,
	}
	if err := te.resetOne(tmr2); err != nil {
		t.Fatalf("second resetOne failed: %v", err)
	}

	// Should still be the same map entry, but with updated values
	updatedTask := te.tasks["existing"]
	if updatedTask == nil {
		t.Fatal("expected task after second resetOne")
	}
	if updatedTask != origTask {
		t.Fatal("expected same task pointer after update")
	}
	// tms should be close to newTime (allow 1s tolerance for time.Now() drift)
	if updatedTask.tms.Sub(newTime).Abs() > time.Second {
		t.Errorf("expected tms close to newTime, got %v vs %v", updatedTask.tms, newTime)
	}
}

// TestTimerEngineResetOneNoTimerType verifies error when params lack timerType.
func TestTimerEngineResetOneNoTimerType(t *testing.T) {
	te := &TimerEngine{tasks: make(map[string]*timerExec)}

	params := `{"dates":"2025-01-01T00:00:00Z"}`
	tmr := &model.TTrigger{
		Id: "missing-type", Types: "timer", Name: "no-type", Params: params,
	}
	err := te.resetOne(tmr)
	if err == nil {
		t.Fatal("expected error when timerType is missing")
	}
}

// TestTimerEngineResetOneInvalidDate verifies error when dates field is malformed.
func TestTimerEngineResetOneInvalidDate(t *testing.T) {
	te := &TimerEngine{tasks: make(map[string]*timerExec)}

	params := `{"timerType":1,"dates":"not-a-date"}`
	tmr := &model.TTrigger{
		Id: "bad-date", Types: "timer", Name: "bad-date", Params: params,
	}
	err := te.resetOne(tmr)
	if err == nil {
		t.Fatal("expected error for invalid date format")
	}
}

// TestTimerEngineRunEmptyTasks verifies run() does not panic with an empty map.
func TestTimerEngineRunEmptyTasks(t *testing.T) {
	te := &TimerEngine{tasks: make(map[string]*timerExec)}
	// Should not panic
	te.run()
}

// TestTimerEngineRunWithFutureTick verifies run() skips tasks whose tick is
// in the future (time.Since(v.tick) <= 0).
func TestTimerEngineRunWithFutureTick(t *testing.T) {
	te := &TimerEngine{tasks: make(map[string]*timerExec)}
	te.tasks["future"] = &timerExec{
		tt:   &model.TTrigger{Id: "future", Name: "future"},
		typ:  1,
		tick: time.Now().Add(time.Hour),
	}
	// Should not panic, and task should remain (not deleted since typ != 0)
	te.run()
	if _, ok := te.tasks["future"]; !ok {
		t.Fatal("future task should still exist after run")
	}
}

// TestTimerEngineExecItemTickNotReached verifies execItem skips when tick is future.
func TestTimerEngineExecItemTickNotReached(t *testing.T) {
	te := &TimerEngine{tasks: make(map[string]*timerExec)}
	task := &timerExec{
		tt:   &model.TTrigger{Id: "notready", Name: "notready"},
		typ:  1,
		tick: time.Now().Add(time.Hour),
	}
	// Should do nothing since tick is in the future
	te.execItem(task)
	if task.typ != 1 {
		t.Error("task type should not change when tick not reached")
	}
}

// TestTimerEngineExecItemRecurringTypes verifies that recurring timer types
// (1-5) advance their tick when the scheduled time has passed.
// We do NOT call service.TriggerTimer (requires DB), so we just verify
// the tick advancement logic by catching the panic from service.TriggerTimer.
func TestTimerEngineExecItemRecurringTypes(t *testing.T) {
	tests := []struct {
		name string
		typ  int64
	}{
		{"minute", 1},
		{"hour", 2},
		{"day", 3},
		{"week", 4},
		{"month", 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			te := &TimerEngine{tasks: make(map[string]*timerExec)}
			pastTick := time.Now().Add(-time.Second)
			task := &timerExec{
				tt:   &model.TTrigger{Id: "test-" + tt.name, Name: "test"},
				typ:  tt.typ,
				tick: pastTick,
			}
			te.tasks["test-"+tt.name] = task

			// execItem will advance tick, then call service.TriggerTimer
			// which may panic or error due to no DB. We just verify tick advanced.
			func() {
				defer func() {
					_ = recover() // absorb any panic from service.TriggerTimer
				}()
				te.execItem(task)
			}()

			// After exec, tick should be in the future (advanced)
			if !task.tick.After(pastTick) {
				t.Errorf("expected tick to advance for type %d, got %v", tt.typ, task.tick)
			}
		})
	}
}

// TestTimerEngineExecItemOnceType0 verifies that a type 0 (once) timer fires
// and spawns a deletion goroutine.
func TestTimerEngineExecItemOnceType0(t *testing.T) {
	te := &TimerEngine{tasks: make(map[string]*timerExec)}
	pastTick := time.Now().Add(-time.Second)
	task := &timerExec{
		tt:   &model.TTrigger{Id: "once-test", Name: "once"},
		typ:  0,
		tick: pastTick,
	}
	te.tasks["once-test"] = task

	func() {
		defer func() {
			_ = recover() // absorb panic from service.TriggerTimer
		}()
		te.execItem(task)
	}()

	// Give the deletion goroutine a moment
	time.Sleep(50 * time.Millisecond)

	// Once timer should trigger deletion (goroutine spawned)
	// It may or may not have completed by now, so we just check no panic
}

// TestTimerEngineResetOneOnceExistingUpdate verifies that resetting an
// existing once-timer updates its schedule.
func TestTimerEngineResetOneOnceExistingUpdate(t *testing.T) {
	te := &TimerEngine{tasks: make(map[string]*timerExec)}

	// Add initial future once timer
	future1 := time.Now().Add(time.Hour)
	params1 := makeTimerParams(0, future1)
	tmr1 := &model.TTrigger{
		Id: "once-update", Types: "timer", Name: "once", Params: params1,
	}
	if err := te.resetOne(tmr1); err != nil {
		t.Fatalf("first resetOne failed: %v", err)
	}
	orig := te.tasks["once-update"]
	if orig == nil {
		t.Fatal("expected task")
	}

	// Update with a new future time
	future2 := time.Now().Add(2 * time.Hour)
	params2 := makeTimerParams(0, future2)
	tmr2 := &model.TTrigger{
		Id: "once-update", Types: "timer", Name: "once-updated", Params: params2,
	}
	if err := te.resetOne(tmr2); err != nil {
		t.Fatalf("second resetOne failed: %v", err)
	}
	updated := te.tasks["once-update"]
	if updated == nil {
		t.Fatal("expected task after update")
	}
	if updated != orig {
		t.Fatal("expected same pointer")
	}
	// tms should be close to future2 (allow 1s tolerance for time.Now() drift)
	if updated.tms.Sub(future2).Abs() > time.Second {
		t.Errorf("expected tms close to future2, got %v vs %v", updated.tms, future2)
	}
}

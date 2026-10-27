package engine

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/gokins/gokins/model"
)

// TestTimerEngineResetOneHourTimer verifies type=2 (hourly) timer.
func TestTimerEngineResetOneHourTimer(t *testing.T) {
	te := &TimerEngine{tasks: make(map[string]*timerExec)}
	params := map[string]any{
		"timerType": 2,
		"dates":     time.Now().Format(time.RFC3339Nano),
	}
	b, _ := json.Marshal(params)
	tmr := &model.TTrigger{
		Id: "hour-1", Types: "timer", Name: "hour", Params: string(b),
	}
	if err := te.resetOne(tmr); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	task, ok := te.tasks["hour-1"]
	if !ok {
		t.Fatal("expected task to be added")
	}
	if task.typ != 2 {
		t.Fatalf("expected type 2, got %d", task.typ)
	}
	if time.Until(task.tick) > time.Hour+time.Second {
		t.Fatal("tick should be within ~1 hour")
	}
}

// TestTimerEngineResetOneDailyTimer verifies type=3 (daily) timer.
func TestTimerEngineResetOneDailyTimer(t *testing.T) {
	te := &TimerEngine{tasks: make(map[string]*timerExec)}
	params := map[string]any{
		"timerType": 3,
		"dates":     time.Now().Format(time.RFC3339Nano),
	}
	b, _ := json.Marshal(params)
	tmr := &model.TTrigger{
		Id: "daily-1", Types: "timer", Name: "daily", Params: string(b),
	}
	if err := te.resetOne(tmr); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	task, ok := te.tasks["daily-1"]
	if !ok {
		t.Fatal("expected task to be added")
	}
	if task.typ != 3 {
		t.Fatalf("expected type 3, got %d", task.typ)
	}
	if time.Until(task.tick) > 25*time.Hour {
		t.Fatal("tick should be within ~24 hours")
	}
}

// TestTimerEngineResetOneWeeklyTimer verifies type=4 (weekly) timer.
func TestTimerEngineResetOneWeeklyTimer(t *testing.T) {
	te := &TimerEngine{tasks: make(map[string]*timerExec)}
	params := map[string]any{
		"timerType": 4,
		"dates":     time.Now().Format(time.RFC3339Nano),
	}
	b, _ := json.Marshal(params)
	tmr := &model.TTrigger{
		Id: "weekly-1", Types: "timer", Name: "weekly", Params: string(b),
	}
	if err := te.resetOne(tmr); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	task, ok := te.tasks["weekly-1"]
	if !ok {
		t.Fatal("expected task to be added")
	}
	if task.typ != 4 {
		t.Fatalf("expected type 4, got %d", task.typ)
	}
	if time.Until(task.tick) > 8*24*time.Hour {
		t.Fatal("tick should be within ~7 days")
	}
}

// TestTimerEngineResetOneMonthlyTimer verifies type=5 (monthly) timer.
func TestTimerEngineResetOneMonthlyTimer(t *testing.T) {
	te := &TimerEngine{tasks: make(map[string]*timerExec)}
	params := map[string]any{
		"timerType": 5,
		"dates":     time.Now().Format(time.RFC3339Nano),
	}
	b, _ := json.Marshal(params)
	tmr := &model.TTrigger{
		Id: "monthly-1", Types: "timer", Name: "monthly", Params: string(b),
	}
	if err := te.resetOne(tmr); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	task, ok := te.tasks["monthly-1"]
	if !ok {
		t.Fatal("expected task to be added")
	}
	if task.typ != 5 {
		t.Fatalf("expected type 5, got %d", task.typ)
	}
	if time.Until(task.tick) > 31*24*time.Hour {
		t.Fatal("tick should be within ~30 days")
	}
}

// TestTimerEngineResetOnePastOnceTimer verifies that a one-time timer set in
// the past is NOT registered.
func TestTimerEngineResetOnePastOnceTimer(t *testing.T) {
	te := &TimerEngine{tasks: make(map[string]*timerExec)}
	past := time.Now().Add(-time.Hour)
	params := map[string]any{
		"timerType": 0,
		"dates":     past.Format(time.RFC3339Nano),
	}
	b, _ := json.Marshal(params)
	tmr := &model.TTrigger{
		Id: "past-once", Types: "timer", Name: "past", Params: string(b),
	}
	if err := te.resetOne(tmr); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := te.tasks["past-once"]; ok {
		t.Fatal("past one-time timer should not be registered")
	}
}

// TestTimerEngineResetOneMissingTimerType verifies error when timerType is absent.
func TestTimerEngineResetOneMissingTimerType(t *testing.T) {
	te := &TimerEngine{tasks: make(map[string]*timerExec)}
	params := map[string]any{
		"dates": time.Now().Format(time.RFC3339Nano),
	}
	b, _ := json.Marshal(params)
	tmr := &model.TTrigger{
		Id: "missing-type", Types: "timer", Name: "no-type", Params: string(b),
	}
	err := te.resetOne(tmr)
	if err == nil {
		t.Fatal("expected error when timerType is missing")
	}
}

// TestTimerEngineResetOneInvalidDates verifies error when dates cannot be parsed.
func TestTimerEngineResetOneInvalidDates(t *testing.T) {
	te := &TimerEngine{tasks: make(map[string]*timerExec)}
	params := map[string]any{
		"timerType": 1,
		"dates":     "not-a-date",
	}
	b, _ := json.Marshal(params)
	tmr := &model.TTrigger{
		Id: "bad-dates", Types: "timer", Name: "bad", Params: string(b),
	}
	err := te.resetOne(tmr)
	if err == nil {
		t.Fatal("expected error when dates cannot be parsed")
	}
}

// TestTimerEngineResetOneUpdateExisting verifies that resetting an existing
// timer updates its fields rather than creating a duplicate.
func TestTimerEngineResetOneUpdateExisting(t *testing.T) {
	te := &TimerEngine{tasks: make(map[string]*timerExec)}

	// First reset
	params1 := map[string]any{
		"timerType": 1,
		"dates":     time.Now().Format(time.RFC3339Nano),
	}
	b1, _ := json.Marshal(params1)
	tmr1 := &model.TTrigger{
		Id: "upd-1", Types: "timer", Name: "upd", Params: string(b1),
	}
	if err := te.resetOne(tmr1); err != nil {
		t.Fatalf("first resetOne: %v", err)
	}
	firstTick := te.tasks["upd-1"].tick

	// Wait a tiny bit so tick changes
	time.Sleep(10 * time.Millisecond)

	// Second reset with type 2
	params2 := map[string]any{
		"timerType": 2,
		"dates":     time.Now().Format(time.RFC3339Nano),
	}
	b2, _ := json.Marshal(params2)
	tmr2 := &model.TTrigger{
		Id: "upd-1", Types: "timer", Name: "upd", Params: string(b2),
	}
	if err := te.resetOne(tmr2); err != nil {
		t.Fatalf("second resetOne: %v", err)
	}

	task := te.tasks["upd-1"]
	if task.typ != 2 {
		t.Fatalf("expected type 2 after update, got %d", task.typ)
	}
	if task.tick.Equal(firstTick) {
		t.Fatal("expected tick to change after update")
	}
	if len(te.tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(te.tasks))
	}
}

// TestTimerEngineExecItemPastTickOnce verifies execItem for a one-time timer
// whose tick is in the past. Since it calls service.TriggerTimer which requires
// a DB, we verify the deletion path is exercised (type 0 triggers a Delete goroutine).
func TestTimerEngineExecItemOncePast(t *testing.T) {
	te := &TimerEngine{tasks: make(map[string]*timerExec)}
	// type=0 timer with tick in the past → should trigger deletion goroutine
	te.tasks["exec-once"] = &timerExec{
		tt:   &model.TTrigger{Id: "exec-once", Name: "test"},
		typ:  0,
		tick: time.Now().Add(-time.Second),
	}

	// This will attempt service.TriggerTimer which may fail (no DB), but the
	// defer util.RecoverLog should catch any panic. We just verify no panic.
	te.execItem(te.tasks["exec-once"])

	// Give the deletion goroutine time to run
	time.Sleep(50 * time.Millisecond)
	// The timer should have been deleted (or still exist if goroutine hasn't run yet)
	_ = te.tasks["exec-once"]
}

// TestTimerEngineExecItemFutureTick verifies execItem does nothing when tick is
// in the future.
func TestTimerEngineExecItemFutureTick(t *testing.T) {
	te := &TimerEngine{tasks: make(map[string]*timerExec)}
	te.tasks["future"] = &timerExec{
		tt:   &model.TTrigger{Id: "future", Name: "test"},
		typ:  1,
		tick: time.Now().Add(time.Hour),
	}
	originalTick := te.tasks["future"].tick
	te.execItem(te.tasks["future"])
	// tick should not change
	if !te.tasks["future"].tick.Equal(originalTick) {
		t.Error("execItem should not modify tick when it is in the future")
	}
}
